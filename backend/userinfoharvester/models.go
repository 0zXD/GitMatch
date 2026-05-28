package main

import "encoding/json"

type UserProfile struct {
	Name        string         `json:"name"`
	Username    string         `json:"username"`
	Bio         string         `json:"bio"`
	Location    string         `json:"location"`
	Company     string         `json:"company"`
	Twitter     string         `json:"twitter"`
	Blog        string         `json:"blog"`
	PublicRepos int            `json:"public_repos"`
	Followers   int            `json:"followers"`
	Following   int            `json:"following"`
	Created     string         `json:"created_at"`
	Languages   map[string]int `json:"languages"`
	Topics      map[string]int `json:"topics"`
}

type SaveIssueRequest struct {
	Username string          `json:"username"`
	Issue    json.RawMessage `json:"issue_data"`
	IssueID  string          `json:"issue_id"`
}
