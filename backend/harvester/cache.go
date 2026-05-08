package main

import (
	"context"
	"log"
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

func loadEnvFiles() {
	for _, file := range []string{".env", "../.env", "../../.env", "backend/.env", "../backend/.env"} {
		if err := godotenv.Load(file); err == nil {
			return
		}
	}
}

func initMongo() {
	loadEnvFiles()

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
