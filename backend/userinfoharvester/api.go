package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

func handleUserRequest(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Missing 'username' query parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var client *github.Client

	// 1. Try to get user-specific token from DB
	var encryptedToken string
	err := db.QueryRow("SELECT encrypted_token FROM github_users WHERE username = $1", username).Scan(&encryptedToken)
	if err == nil && encryptedToken != "" {
		decrypted, err := decryptToken(encryptedToken)
		if err == nil && decrypted != "" {
			ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: decrypted})
			tc := oauth2.NewClient(ctx, ts)
			client = github.NewClient(tc)
		}
	}

	// 2. Fallback to global token
	if client == nil {
		token := os.Getenv("GITHUB_TOKEN")
		if token != "" {
			ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
			tc := oauth2.NewClient(ctx, ts)
			client = github.NewClient(tc)
		} else {
			client = github.NewClient(nil)
		}
	}

	user, _, err := client.Users.Get(ctx, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user: %v", err), http.StatusInternalServerError)
		return
	}

	opt := &github.RepositoryListOptions{
		Type:        "owner",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.List(ctx, username, opt)
		if err != nil {
			log.Printf("Error fetching repositories: %v", err)
			break
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	languageMap := make(map[string]int)
	topicMap := make(map[string]int)

	for _, repo := range allRepos {
		if repo.Language != nil {
			languageMap[repo.GetLanguage()]++
		}
		for _, topic := range repo.Topics {
			topicMap[topic]++
		}
	}

	profile := UserProfile{
		Name:        user.GetName(),
		Username:    user.GetLogin(),
		Bio:         user.GetBio(),
		Location:    user.GetLocation(),
		Company:     user.GetCompany(),
		Twitter:     user.GetTwitterUsername(),
		Blog:        user.GetBlog(),
		PublicRepos: user.GetPublicRepos(),
		Followers:   user.GetFollowers(),
		Following:   user.GetFollowing(),
		Created:     user.GetCreatedAt().Format("2006-01-02"),
		Languages:   languageMap,
		Topics:      topicMap,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func handleSavedIssuesRequest(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "Missing username parameter", http.StatusBadRequest)
			return
		}

		rows, err := db.Query("SELECT issue_data FROM saved_issues WHERE username = $1 ORDER BY created_at DESC", username)
		if err != nil {
			http.Error(w, "Failed to query saved issues", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var issues []json.RawMessage
		for rows.Next() {
			var data []byte
			if err := rows.Scan(&data); err != nil {
				log.Printf("Row scan error: %v", err)
				continue
			}
			issues = append(issues, data)
		}

		if issues == nil {
			issues = []json.RawMessage{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(issues)

	case http.MethodPost:
		var req SaveIssueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.IssueID == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		_, err := db.Exec("INSERT INTO saved_issues (username, issue_id, issue_data) VALUES ($1, $2, $3) ON CONFLICT (username, issue_id) DO NOTHING",
			req.Username, req.IssueID, req.Issue)
		if err != nil {
			log.Printf("Failed to save issue: %v", err)
			http.Error(w, "Failed to save issue", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"success"}`))

	case http.MethodDelete:
		username := r.URL.Query().Get("username")
		issueID := r.URL.Query().Get("issue_id")

		if username == "" || issueID == "" {
			http.Error(w, "Missing username or issue_id parameter", http.StatusBadRequest)
			return
		}

		_, err := db.Exec("DELETE FROM saved_issues WHERE username = $1 AND issue_id = $2", username, issueID)
		if err != nil {
			http.Error(w, "Failed to delete saved issue", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"deleted"}`))

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
