package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
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
