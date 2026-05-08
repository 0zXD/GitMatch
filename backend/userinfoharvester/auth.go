package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

var oauthConfig *oauth2.Config

func initOAuthConfig() {
	loadEnvFiles()

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
