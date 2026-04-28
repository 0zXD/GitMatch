package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var cacheCollection *mongo.Collection

func initMongo() {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Println("MONGODB_URI not set — caching disabled")
		return
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Printf("MongoDB connection failed: %v — caching disabled", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("MongoDB ping failed: %v — caching disabled", err)
		return
	}

	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		dbName = "gitmatch"
	}

	cacheCollection = client.Database(dbName).Collection("harvest_cache")

	// Create TTL index: entries expire after 24 hours
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "cached_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(86400),
	}
	if _, err := cacheCollection.Indexes().CreateOne(ctx, indexModel); err != nil {
		log.Printf("TTL index creation note: %v", err)
	}

	log.Println("MongoDB cache connected")
}

// normalizeCacheKey produces a stable key from a query string.
// It extracts language:/topic: tokens, sorts them, and joins.
func normalizeCacheKey(q string) string {
	tokens := strings.Fields(q)
	sort.Strings(tokens)
	return strings.Join(tokens, " ")
}

func main() {
	// Try to load .env from current directory, back up to parent if missing
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}

	initMongo()

	http.HandleFunc("/issues", corsMiddleware(handleIssuesRequest))
	port := "8082"
	fmt.Printf("Harvester server running on port %s...\n", port)
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
