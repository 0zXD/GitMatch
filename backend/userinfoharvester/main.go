package main

import (
"context"
"database/sql"
"encoding/json"
"fmt"
"log"
"net/http"
"os"

"github.com/google/go-github/v50/github"
"github.com/joho/godotenv"
_ "github.com/lib/pq"
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

var db *sql.DB

func initDB() {
	var err error
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/gitmatch?sslmode=disable"
	}
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("Failed to connect to PostgreSQL: %v", err)
		return
	}
	err = db.Ping()
	if err != nil {
		log.Printf("Failed to ping PostgreSQL: %v", err)
		return
	}
	log.Println("PostgreSQL connected successfully.")

	// Create table if it doesn't exist
schema := `
CREATE TABLE IF NOT EXISTS saved_issues (
id SERIAL PRIMARY KEY,
username VARCHAR(255) NOT NULL,
issue_id BIGINT NOT NULL,
issue_data JSONB NOT NULL,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
UNIQUE (username, issue_id)
);`
_, err = db.Exec(schema)
if err != nil {
log.Printf("Failed to create table saved_issues: %v", err)
} else {
log.Println("Table saved_issues checked/created.")
}
}

func main() {
// Try loading .env from local or parent directory
if err := godotenv.Load(); err != nil {
_ = godotenv.Load("../../.env")
}

initDB()

http.HandleFunc("/user", corsMiddleware(handleUserRequest))
http.HandleFunc("/saved_issues", corsMiddleware(handleSavedIssuesRequest))

port := "8084"
fmt.Printf("User Info Harvester server running on port %s...\n", port)
log.Fatal(http.ListenAndServe(":"+port, nil))
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
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

type SaveIssueRequest struct {
Username string          `json:"username"`
Issue    json.RawMessage `json:"issue_data"`
IssueID  int64           `json:"issue_id"`
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
issues = []json.RawMessage{} // Ensure empty array rather than null
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(issues)

case http.MethodPost:
var req SaveIssueRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "Invalid request payload", http.StatusBadRequest)
return
}

if req.Username == "" || req.IssueID == 0 {
http.Error(w, "Missing required fields", http.StatusBadRequest)
return
}

_, err := db.Exec("INSERT INTO saved_issues (username, issue_id, issue_data) VALUES ($1, $2, $3) ON CONFLICT (username, issue_id) DO NOTHING",
req.Username, req.IssueID, req.Issue)
if err != nil {
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
