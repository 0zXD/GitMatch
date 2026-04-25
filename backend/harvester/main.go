package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/oauth2"
)

var repoApiCallCount uint64

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
	Key       string       `bson:"_id"`
	Results   []RepoResult `bson:"results"`
	HasMore   bool         `bson:"has_more"`
	EndCursor string       `bson:"end_cursor"`
	CachedAt  time.Time    `bson:"cached_at"`
}

// GraphQL response types for GitHub API v4
type graphQLResponse struct {
	Data   graphQLData    `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLData struct {
	Search graphQLSearch `json:"search"`
}

type graphQLSearch struct {
	RepositoryCount int             `json:"repositoryCount"`
	PageInfo        graphQLPageInfo `json:"pageInfo"`
	Nodes           []graphQLNode   `json:"nodes"`
}

type graphQLPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type graphQLNode struct {
	NameWithOwner   string `json:"nameWithOwner"`
	StargazerCount  int    `json:"stargazerCount"`
	ForkCount       int    `json:"forkCount"`
	URL             string `json:"url"`
	Description     string `json:"description"`
	PrimaryLanguage *struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
	Issues struct {
		TotalCount int `json:"totalCount"`
	} `json:"issues"`
	Languages struct {
		TotalSize int `json:"totalSize"`
		Edges     []struct {
			Size int `json:"size"`
			Node struct {
				Name string `json:"name"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"languages"`
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

const graphQLQueryTemplate = `query($query: String!, $first: Int!, $after: String) {
  search(query: $query, type: REPOSITORY, first: $first, after: $after) {
    repositoryCount
    pageInfo {
      hasNextPage
      endCursor
    }
    nodes {
      ... on Repository {
        nameWithOwner
        stargazerCount
        forkCount
        url
        description
        primaryLanguage {
          name
        }
        issues(states: OPEN) {
          totalCount
        }
        languages(first: 10, orderBy: {field: SIZE, direction: DESC}) {
          totalSize
          edges {
            size
            node {
              name
            }
          }
        }
      }
    }
  }
}`

// graphqlSearchRepos executes a single GraphQL query that fetches repos + languages
// in one API call, replacing the N+1 REST calls (1 search + N language fetches).
func graphqlSearchRepos(ctx context.Context, token, query string, first int, after string) (*graphQLSearch, error) {
	variables := map[string]interface{}{
		"query": query,
		"first": first,
	}
	if after != "" {
		variables["after"] = after
	}

	body := map[string]interface{}{
		"query":     graphQLQueryTemplate,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	currentCount := atomic.AddUint64(&repoApiCallCount, 1)
	log.Printf("Executing API Call #%d ", currentCount)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql request failed with status %d", resp.StatusCode)
	}

	var gqlResp graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	return &gqlResp.Data.Search, nil
}

// graphqlNodeToResult converts a GraphQL repository node to a RepoResult,
// extracting language breakdown from the same response (no extra API call).
func graphqlNodeToResult(node graphQLNode) *RepoResult {
	if node.Issues.TotalCount == 0 {
		return nil
	}

	primaryLang := ""
	if node.PrimaryLanguage != nil {
		primaryLang = node.PrimaryLanguage.Name
	}

	totalBytes := node.Languages.TotalSize
	if totalBytes == 0 {
		return nil
	}

	breakdown := make(map[string]float64)
	var validTags []string
	for _, edge := range node.Languages.Edges {
		percentage := (float64(edge.Size) / float64(totalBytes)) * 100
		breakdown[edge.Node.Name] = percentage
		if percentage >= 10.0 {
			validTags = append(validTags, edge.Node.Name)
		}
	}

	if len(validTags) == 0 {
		return nil
	}

	return &RepoResult{
		Name:              node.NameWithOwner,
		Stars:             node.StargazerCount,
		Forks:             node.ForkCount,
		URL:               node.URL,
		Description:       node.Description,
		PrimaryLanguage:   primaryLang,
		OpenIssues:        node.Issues.TotalCount,
		LanguageBreakdown: breakdown,
		ValidTags:         validTags,
	}
}

func main() {
	// Try to load .env from current directory, back up to parent if missing
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}

	initMongo()

	http.HandleFunc("/harvest", corsMiddleware(handleHarvestRequest))
	http.HandleFunc("/issues", corsMiddleware(handleIssuesRequest))
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

	cursor := r.URL.Query().Get("cursor")
	token := os.Getenv("GITHUB_TOKEN")
	useGraphQL := token != ""

	// Build cache key: GraphQL uses cursor-based, REST uses page-based
	var cacheKey string
	if useGraphQL {
		cacheKey = fmt.Sprintf("gql|%s|cursor=%s", normalizeCacheKey(q), cursor)
	} else {
		cacheKey = fmt.Sprintf("%s|page=%d", normalizeCacheKey(q), page)
	}

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
			json.NewEncoder(w).Encode(struct {
				Results   []RepoResult `json:"results"`
				HasMore   bool         `json:"has_more"`
				Page      int          `json:"page"`
				EndCursor string       `json:"end_cursor,omitempty"`
			}{
				Results:   cached.Results,
				HasMore:   cached.HasMore,
				Page:      page,
				EndCursor: cached.EndCursor,
			})
			return
		}
	}

	var results []RepoResult
	var hasMore bool
	var endCursor string

	if useGraphQL {
		// GraphQL path: 1 API call replaces N+1 REST calls
		log.Printf("GraphQL search: %q (cursor=%q)", q, cursor)
		fullQuery := fmt.Sprintf("%s stars:>0 sort:updated-desc", q)

		searchResult, err := graphqlSearchRepos(r.Context(), token, fullQuery, 30, cursor)
		if err != nil {
			http.Error(w, fmt.Sprintf("GraphQL search failed: %v", err), http.StatusInternalServerError)
			return
		}

		for _, node := range searchResult.Nodes {
			if res := graphqlNodeToResult(node); res != nil {
				results = append(results, *res)
			}
		}
		hasMore = searchResult.PageInfo.HasNextPage
		endCursor = searchResult.PageInfo.EndCursor
		log.Printf("GraphQL returned %d repos → %d valid results (1 API call)", len(searchResult.Nodes), len(results))
	} else {
		// REST fallback: N+1 API calls (no token = no GraphQL)
		log.Println("REST fallback (no GITHUB_TOKEN set — consider adding one for GraphQL efficiency)")
		results, hasMore = restSearchRepos(r.Context(), q, page, r.URL.Query().Get("lite") == "true")
	}

	// Save to cache
	if cacheCollection != nil && len(results) > 0 {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		doc := CachedQuery{
			Key:       cacheKey,
			Results:   results,
			HasMore:   hasMore,
			EndCursor: endCursor,
			CachedAt:  time.Now(),
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
	if useGraphQL {
		w.Header().Set("X-API-Mode", "graphql")
	} else {
		w.Header().Set("X-API-Mode", "rest")
	}
	response := struct {
		Results   []RepoResult `json:"results"`
		HasMore   bool         `json:"has_more"`
		Page      int          `json:"page"`
		EndCursor string       `json:"end_cursor,omitempty"`
	}{
		Results:   results,
		HasMore:   hasMore,
		Page:      page,
		EndCursor: endCursor,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// restSearchRepos is the original REST-based search (N+1 API calls).
// Used as fallback when GITHUB_TOKEN is not set.
func restSearchRepos(ctx context.Context, q string, page int, liteMode bool) ([]RepoResult, bool) {
	var client *github.Client
	token := os.Getenv("GITHUB_TOKEN")
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
		log.Printf("REST search failed: %v", err)
		return nil, false
	}

	var results []RepoResult
	for _, repo := range result.Repositories {
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
	return results, hasMore
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
