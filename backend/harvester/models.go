package main

import "time"

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

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLIssueResponse struct {
	Data   graphQLIssueData `json:"data"`
	Errors []graphQLError   `json:"errors"`
}

type graphQLIssueData struct {
	Search graphQLIssueSearch `json:"search"`
}

type graphQLPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
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
