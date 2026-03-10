package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/oauth2"
)

type RepoResult struct {
	Name              string             `json:"name" bson:"name"`
	Stars             int                `json:"stars" bson:"stars"`
	Forks             int                `json:"forks" bson:"forks"`
	URL               string             `json:"url" bson:"url"`
	Description       string             `json:"description" bson:"description"`
	PrimaryLanguage   string             `json:"primary_language" bson:"primary_language"`
	OpenIssues        int                `json:"open_issues" bson:"open_issues"`
	LanguageBreakdown map[string]float64 `json:"language_breakdown" bson:"language_breakdown"`
	ValidTags         []string           `json:"valid_tags" bson:"valid_tags"`
}

type CachedQuery struct {
	Key      string       `bson:"_id"`
	Results  []RepoResult `bson:"results"`
	CachedAt time.Time    `bson:"cached_at"`
}

var cacheCollection *mongo.Collection

func initMongo() {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Println("MONGODB_URI not set — caching disabled")
		return
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Printf("MongoDB connection failed: %v — caching disabled", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("MongoDB ping failed: %v — caching disabled", err)
		return
	}

	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		dbName = "gitmatch"
	}

	cacheCollection = client.Database(dbName).Collection("harvest_cache")

	// Create TTL index: entries expire after 24 hours
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "cached_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(86400),
	}
	if _, err := cacheCollection.Indexes().CreateOne(ctx, indexModel); err != nil {
		log.Printf("TTL index creation note: %v", err)
	}

	log.Println("MongoDB cache connected")
}

// normalizeCacheKey produces a stable key from a query string.
// It extracts language:/topic: tokens, sorts them, and joins.
func normalizeCacheKey(q string) string {
	tokens := strings.Fields(q)
	sort.Strings(tokens)
	return strings.Join(tokens, " ")
}

func main() {
	// Try to load .env from current directory, back up to parent if missing
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}

	initMongo()

	http.HandleFunc("/harvest", corsMiddleware(handleHarvestRequest))
	port := "8082"
	fmt.Printf("Harvester server running on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func handleHarvestRequest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		topic := r.URL.Query().Get("topic")
		if topic != "" {
			q = fmt.Sprintf("topic:%s", topic)
		} else {
			http.Error(w, "Missing 'q' or 'topic' query parameter", http.StatusBadRequest)
			return
		}
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	cacheKey := fmt.Sprintf("%s|page=%d", normalizeCacheKey(q), page)

	// Try cache first
	if cacheCollection != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var cached CachedQuery
		err := cacheCollection.FindOne(ctx, bson.M{"_id": cacheKey}).Decode(&cached)
		if err == nil {
			log.Printf("Cache HIT for %q", cacheKey)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			response := struct {
				Results []RepoResult `json:"results"`
				HasMore bool         `json:"has_more"`
				Page    int          `json:"page"`
			}{
				Results: cached.Results,
				HasMore: true, // Assume more pages available from cache
				Page:    page,
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	ctx := context.Background()
	token := os.Getenv("GITHUB_TOKEN")
	var client *github.Client

	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	fullQuery := fmt.Sprintf("%s stars:>0", q)
	opts := &github.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: 30,
		},
	}

	result, resp, err := client.Search.Repositories(ctx, fullQuery, opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	liteMode := r.URL.Query().Get("lite") == "true"

	var results []RepoResult
	for _, repo := range result.Repositories {
		// Skip repos with no open issues
		if repo.GetOpenIssues() == 0 {
			continue
		}

		if liteMode {
			results = append(results, RepoResult{
				Name:            repo.GetFullName(),
				Stars:           repo.GetStargazersCount(),
				Forks:           repo.GetForksCount(),
				URL:             repo.GetHTMLURL(),
				Description:     repo.GetDescription(),
				PrimaryLanguage: repo.GetLanguage(),
				OpenIssues:      repo.GetOpenIssues(),
			})
		} else {
			res := processRepo(ctx, client, repo)
			if res != nil {
				results = append(results, *res)
			}
		}
	}

	hasMore := resp != nil && resp.NextPage > 0

	// Save to cache
	if cacheCollection != nil && len(results) > 0 {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		doc := CachedQuery{
			Key:      cacheKey,
			Results:  results,
			CachedAt: time.Now(),
		}
		_, err := cacheCollection.ReplaceOne(cacheCtx, bson.M{"_id": cacheKey}, doc,
			options.Replace().SetUpsert(true))
		if err != nil {
			log.Printf("Cache write failed: %v", err)
		} else {
			log.Printf("Cache MISS — saved %d results for %q", len(results), cacheKey)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	response := struct {
		Results []RepoResult `json:"results"`
		HasMore bool         `json:"has_more"`
		Page    int          `json:"page"`
	}{
		Results: results,
		HasMore: hasMore,
		Page:    page,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func processRepo(ctx context.Context, client *github.Client, repo *github.Repository) *RepoResult {
	langs, _, err := client.Repositories.ListLanguages(ctx, repo.GetOwner().GetLogin(), repo.GetName())
	if err != nil {
		log.Printf("Skipping %s: %v", repo.GetFullName(), err)
		return nil
	}

	validTags := []string{}
	totalBytes := 0
	for _, bytes := range langs {
		totalBytes += bytes
	}

	if totalBytes == 0 {
		return nil
	}

	breakdown := make(map[string]float64)
	for lang, bytes := range langs {
		percentage := (float64(bytes) / float64(totalBytes)) * 100
		breakdown[lang] = percentage
		if percentage >= 10.0 {
			validTags = append(validTags, lang)
		}
	}

	if len(validTags) > 0 {
		return &RepoResult{
			Name:              repo.GetFullName(),
			Stars:             repo.GetStargazersCount(),
			Forks:             repo.GetForksCount(),
			URL:               repo.GetHTMLURL(),
			Description:       repo.GetDescription(),
			PrimaryLanguage:   repo.GetLanguage(),
			OpenIssues:        repo.GetOpenIssues(),
			LanguageBreakdown: breakdown,
			ValidTags:         validTags,
		}
	}
	return nil
}
