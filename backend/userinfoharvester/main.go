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

func main() {
	http.HandleFunc("/user", corsMiddleware(handleUserRequest))
	port := "8084"
	fmt.Printf("User Info Harvester server running on port %s...\n", port)
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

func handleUserRequest(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Missing 'username' query parameter", http.StatusBadRequest)
		return
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

	// Fetch User Data
	user, _, err := client.Users.Get(ctx, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user: %v", err), http.StatusInternalServerError)
		return
	}

	// Fetch Repositories
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
			log.Printf("Error fetching repositories: %v", err) // Log error but don't fail completely
			break
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// Aggregate Languages and Topics
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
