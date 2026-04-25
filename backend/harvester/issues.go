package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var issuesApiCallCount uint64

type IssueResult struct {
	ID                string             `json:"id" bson:"id"`
	Title             string             `json:"title" bson:"title"`
	URL               string             `json:"url" bson:"url"`
	Number            int                `json:"number" bson:"number"`
	State             string             `json:"state" bson:"state"`
	Body              string             `json:"body" bson:"body"`
	Comments          int                `json:"comments" bson:"comments"`
	Labels            []string           `json:"labels" bson:"labels"`
	CreatedAt         time.Time          `json:"created_at" bson:"created_at"`
	RepoName          string             `json:"name" bson:"repo_name"`
	RepoURL           string             `json:"repo_url" bson:"repo_url"`
	RepoStars         int                `json:"stars" bson:"repo_stars"`
	RepoDescription   string             `json:"description" bson:"repo_description"`
	PrimaryLanguage   string             `json:"primary_language" bson:"primary_language"`
	LanguageBreakdown map[string]float64 `json:"language_breakdown" bson:"language_breakdown"`
	ValidTags         []string           `json:"valid_tags" bson:"valid_tags"`
}

type CachedIssuesQuery struct {
	Key       string        `bson:"_id"`
	Results   []IssueResult `bson:"results"`
	HasMore   bool          `bson:"has_more"`
	EndCursor string        `bson:"end_cursor"`
	CachedAt  time.Time     `bson:"cached_at"`
}

// GraphQL response types for GitHub API v4 Issues
type graphQLIssueResponse struct {
	Data   graphQLIssueData `json:"data"`
	Errors []graphQLError   `json:"errors"`
}

type graphQLIssueData struct {
	Search graphQLIssueSearch `json:"search"`
}

type graphQLIssueSearch struct {
	IssueCount int                `json:"issueCount"`
	PageInfo   graphQLPageInfo    `json:"pageInfo"`
	Nodes      []graphQLIssueNode `json:"nodes"`
}

type graphQLIssueNode struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Number    int       `json:"number"`
	State     string    `json:"state"`
	BodyText  string    `json:"bodyText"`
	CreatedAt time.Time `json:"createdAt"`
	Comments  struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Repository struct {
		NameWithOwner   string `json:"nameWithOwner"`
		StargazerCount  int    `json:"stargazerCount"`
		URL             string `json:"url"`
		Description     string `json:"description"`
		PrimaryLanguage *struct {
			Name string `json:"name"`
		} `json:"primaryLanguage"`
		Languages struct {
			TotalSize int `json:"totalSize"`
			Edges     []struct {
				Size int `json:"size"`
				Node struct {
					Name string `json:"name"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"languages"`
	} `json:"repository"`
}

const graphQLIssueQueryTemplate = `query($query: String!, $first: Int!, $after: String) {
  search(query: $query, type: ISSUE, first: $first, after: $after) {
    issueCount
    pageInfo {
      hasNextPage
      endCursor
    }
    nodes {
      ... on Issue {
        id
        title
        url
        number
        state
        bodyText
        createdAt
        comments {
          totalCount
        }
        labels(first: 10) {
          nodes {
            name
          }
        }
        repository {
          nameWithOwner
          stargazerCount
          url
          description
          primaryLanguage {
            name
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
  }
}`

func graphqlSearchIssues(ctx context.Context, token, query string, first int, after string) (*graphQLIssueSearch, error) {
	variables := map[string]interface{}{
		"query": query,
		"first": first,
	}
	if after != "" {
		variables["after"] = after
	}

	body := map[string]interface{}{
		"query":     graphQLIssueQueryTemplate,
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

	currentCount := atomic.AddUint64(&issuesApiCallCount, 1)
	log.Printf("Executing API Call #%d ", currentCount)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql request failed with status %d", resp.StatusCode)
	}

	var gqlResp graphQLIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	return &gqlResp.Data.Search, nil
}

func graphqlIssueNodeToResult(node graphQLIssueNode) *IssueResult {
	if node.Repository.NameWithOwner == "" {
		return nil
	}

	primaryLang := ""
	if node.Repository.PrimaryLanguage != nil {
		primaryLang = node.Repository.PrimaryLanguage.Name
	}

	totalBytes := node.Repository.Languages.TotalSize
	breakdown := make(map[string]float64)
	var validTags []string

	if totalBytes > 0 {
		for _, edge := range node.Repository.Languages.Edges {
			percentage := (float64(edge.Size) / float64(totalBytes)) * 100
			breakdown[edge.Node.Name] = percentage
			if percentage >= 10.0 {
				validTags = append(validTags, edge.Node.Name)
			}
		}
	}

	var labels []string
	for _, l := range node.Labels.Nodes {
		labels = append(labels, l.Name)
	}

	bodyStr := node.BodyText
	if len(bodyStr) > 500 {
		bodyStr = bodyStr[:500] + "..."
	}

	return &IssueResult{
		ID:                node.ID,
		Title:             node.Title,
		URL:               node.URL,
		Number:            node.Number,
		State:             node.State,
		Body:              bodyStr,
		Comments:          node.Comments.TotalCount,
		Labels:            labels,
		CreatedAt:         node.CreatedAt,
		RepoName:          node.Repository.NameWithOwner,
		RepoURL:           node.Repository.URL,
		RepoStars:         node.Repository.StargazerCount,
		RepoDescription:   node.Repository.Description,
		PrimaryLanguage:   primaryLang,
		LanguageBreakdown: breakdown,
		ValidTags:         validTags,
	}
}

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

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		http.Error(w, "GITHUB_TOKEN not configured", http.StatusInternalServerError)
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
