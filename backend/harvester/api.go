package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func handleIssuesRequest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		topic := r.URL.Query().Get("topic")
		lang := r.URL.Query().Get("language")

		if topic != "" || lang != "" {
			var queryParts []string
			if topic != "" {
				queryParts = append(queryParts, fmt.Sprintf("topic:%s", topic))
			}
			if lang != "" {
				queryParts = append(queryParts, fmt.Sprintf("language:%s", lang))
			}
			q = strings.Join(queryParts, " ")
		} else {
			http.Error(w, "Missing search parameters", http.StatusBadRequest)
			return
		}
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	experience := r.URL.Query().Get("experience")
	if experience == "" {
		experience = "beginner"
	}
	repoCountStr := r.URL.Query().Get("repoCount")
	if repoCountStr == "" {
		repoCountStr = "0"
	}

	var issueQualifiers string
	if experience == "beginner" {
		issueQualifiers = "is:issue is:open label:\"good first issue\""
	} else if experience == "intermediate" {
		issueQualifiers = "is:issue is:open label:\"help wanted\""
	} else {
		issueQualifiers = "is:issue is:open"
	}

	fullQuery := fmt.Sprintf("%s %s", q, issueQualifiers)

	cursor := r.URL.Query().Get("after")
	cacheKey := fmt.Sprintf("issues|%s|exp=%s|repos=%s|page=%d|after=%s", normalizeCacheKey(fullQuery), experience, repoCountStr, page, cursor)

	if cacheCollection != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var cached CachedIssuesQuery
		err := cacheCollection.FindOne(ctx, bson.M{"_id": cacheKey}).Decode(&cached)
		if err == nil {
			log.Printf("Cache HIT for %q", cacheKey)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			json.NewEncoder(w).Encode(struct {
				Results   []IssueResult `json:"results"`
				HasMore   bool          `json:"has_more"`
				Page      int           `json:"page"`
				EndCursor string        `json:"end_cursor"`
			}{
				Results:   cached.Results,
				HasMore:   cached.HasMore,
				Page:      page,
				EndCursor: cached.EndCursor,
			})
			return
		}
	}

	token := r.Header.Get("Authorization")
	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	token = strings.TrimSpace(token)

	if token == "" {
		http.Error(w, "GITHUB_TOKEN not configured or provided", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	searchData, err := graphqlSearchIssues(ctx, token, fullQuery, 30, cursor)
	if err != nil {
		log.Printf("GraphQL search failed: %v", err)
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	var results []IssueResult

	for _, node := range searchData.Nodes {
		if res := graphqlIssueNodeToResult(node); res != nil {
			results = append(results, *res)
		}
	}

	hasMore := searchData.PageInfo.HasNextPage
	endCursor := searchData.PageInfo.EndCursor

	if cacheCollection != nil && len(results) > 0 {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		doc := CachedIssuesQuery{
			Key:       cacheKey,
			Results:   results,
			HasMore:   hasMore,
			EndCursor: endCursor,
			CachedAt:  time.Now(),
		}
		_, err := cacheCollection.ReplaceOne(cacheCtx, bson.M{"_id": cacheKey}, doc, options.Replace().SetUpsert(true))
		if err != nil {
			log.Printf("Cache write failed: %v", err)
		} else {
			log.Printf("Cache MISS — saved %d results for %q", len(results), cacheKey)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(struct {
		Results   []IssueResult `json:"results"`
		HasMore   bool          `json:"has_more"`
		Page      int           `json:"page"`
		EndCursor string        `json:"end_cursor"`
	}{
		Results:   results,
		HasMore:   hasMore,
		Page:      page,
		EndCursor: endCursor,
	})
}
