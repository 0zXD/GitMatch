package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/google/go-github/v50/github"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
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
var oauthConfig *oauth2.Config

func getEncryptionKey() []byte {
	key := []byte(os.Getenv("ENCRYPTION_KEY"))
	if len(key) == 0 {
		key = []byte("default-secret-key-must-be-32-bt") // 32 bytes fallback
	}
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	}
	return key
}

func encryptToken(token string) (string, error) {
	block, err := aes.NewCipher(getEncryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptToken(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(getEncryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

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

	schema := `
		CREATE TABLE IF NOT EXISTS saved_issues (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			issue_id VARCHAR(255) NOT NULL,
			issue_data JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (username, issue_id)
		);
	`
	_, err = db.Exec(schema)
	if err != nil {
		log.Printf("Failed to create table saved_issues: %v", err)
	} else {
		log.Println("Table saved_issues checked/created.")
	}

	authSchema := `
		CREATE TABLE IF NOT EXISTS github_users (
			username VARCHAR(255) PRIMARY KEY,
			encrypted_token TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err = db.Exec(authSchema)
	if err != nil {
		log.Printf("Failed to create table github_users: %v", err)
	} else {
		log.Println("Table github_users checked/created.")
	}
}

func initOAuthConfig() {
	redirectURL := os.Getenv("GITHUB_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:8084/auth/github/callback"
	}
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  redirectURL,
		Scopes:       []string{"read:user", "public_repo"},
		Endpoint:     githuboauth.Endpoint,
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../../.env")
	}

	initDB()
	initOAuthConfig()

	http.HandleFunc("/auth/github/login", corsMiddleware(handleGitHubLogin))
	http.HandleFunc("/auth/github/callback", corsMiddleware(handleGitHubCallback))
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

func handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	state := "github-oauth-state-random" // In production, use a secure state and verify it
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing 'code' parameter", http.StatusBadRequest)
		return
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange token: %v", err), http.StatusInternalServerError)
		return
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token.AccessToken})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user with token: %v", err), http.StatusInternalServerError)
		return
	}
	username := user.GetLogin()

	encrypted, err := encryptToken(token.AccessToken)
	if err != nil {
		http.Error(w, "Failed to encrypt token", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`
		INSERT INTO github_users (username, encrypted_token)
		VALUES ($1, $2)
		ON CONFLICT (username) DO UPDATE SET encrypted_token = EXCLUDED.encrypted_token, updated_at = CURRENT_TIMESTAMP
	`, username, encrypted)
	if err != nil {
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	// Note: You could use a JWT or session cookie here. For now, redirecting with username
	http.Redirect(w, r, frontendURL+"?username="+url.QueryEscape(username), http.StatusFound)
}

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

type SaveIssueRequest struct {
	Username string          `json:"username"`
	Issue    json.RawMessage `json:"issue_data"`
	IssueID  string          `json:"issue_id"`
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
