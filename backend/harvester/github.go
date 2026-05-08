package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

var issuesApiCallCount uint64

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
	req.Header.Set("User-Agent", "gitmatch-harvester")

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
