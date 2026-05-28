package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

func loadEnvFiles() {
	for _, file := range []string{".env", "../.env", "../../.env", "backend/.env", "../backend/.env"} {
		if err := godotenv.Load(file); err == nil {
			return
		}
	}
}

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
	loadEnvFiles()

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
