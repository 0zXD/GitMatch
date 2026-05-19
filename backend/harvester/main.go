package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	loadEnvFiles()

	initMongo()

	http.HandleFunc("/issues", corsMiddleware(handleIssuesRequest))
	http.HandleFunc("/analyze-issue", corsMiddleware(handleAnalyzeIssueRequest))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	fmt.Printf("Harvester serving on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
