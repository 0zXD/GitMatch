package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v50/github"
	openai "github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/oauth2"
)

func handleAnalyzeIssueRequest(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	issueStr := r.URL.Query().Get("issue")

	if owner == "" || repo == "" || issueStr == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}

	issueNumber, err := strconv.Atoi(issueStr)
	if err != nil {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}

	token := r.Header.Get("Authorization")
	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	ctx := context.Background()
	analysis, err := getIssueAnalysis(ctx, owner, repo, issueNumber, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

func getIssueAnalysis(ctx context.Context, owner, repo string, issueNumber int, githubToken string) (*IssueAnalysis, error) {
	issueKey := fmt.Sprintf("%s/%s#%d", owner, repo, issueNumber)

	// Check cache first
	if repoAnalysisCollection != nil {
		var analysis IssueAnalysis
		err := repoAnalysisCollection.FindOne(ctx, bson.M{"_id": issueKey}).Decode(&analysis)
		if err == nil {
			return &analysis, nil
		}
	}

	log.Printf("Analyzing Issue %s...", issueKey)

	// Fetch README and Issue
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	readme, _, err := client.Repositories.GetReadme(ctx, owner, repo, nil)
	var readmeContent string
	if err == nil {
		if readme.Content != nil {
			decoded, decErr := base64.StdEncoding.DecodeString(*readme.Content)
			if decErr == nil {
				readmeContent = string(decoded)
			}
		}
	}

	issue, _, err := client.Issues.Get(ctx, owner, repo, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}
	issueBody := ""
	if issue.Body != nil {
		issueBody = *issue.Body
	}

	analysis, err := analyzeIssueWithLLM(ctx, readmeContent, issueBody)
	if err != nil {
		return nil, fmt.Errorf("analyzing issue: %w", err)
	}
	analysis.RepoName = issueKey
	analysis.AnalyzedAt = time.Now()

	// Save to cache
	if repoAnalysisCollection != nil {
		_, err := repoAnalysisCollection.ReplaceOne(ctx, bson.M{"_id": issueKey}, analysis, options.Replace().SetUpsert(true))
		if err != nil {
			log.Printf("Failed to cache issue analysis: %v", err)
		}
	}

	return analysis, nil
}

func analyzeIssueWithLLM(ctx context.Context, readmeContent, issueBody string) (*IssueAnalysis, error) {
	// Look for GROQ_API_KEY first, fallback to OPENAI_API_KEY if they reused the var
	apiKeysStr := os.Getenv("GROQ_API_KEY")
	if apiKeysStr == "" {
		apiKeysStr = os.Getenv("OPENAI_API_KEY")
	}
	if apiKeysStr == "" {
		return nil, fmt.Errorf("GROQ_API_KEY or OPENAI_API_KEY is not set")
	}

	apiKeys := strings.Split(apiKeysStr, ",")

	if len(readmeContent) > 20000 {
		readmeContent = readmeContent[:20000]
	}
	if len(issueBody) > 10000 {
		issueBody = issueBody[:10000]
	}

	prompt := `You are an expert developer helping a contributor onboard to a project and tackle a specific issue.
Analyze the provided README AND the Issue body to extract the following information.
Output ONLY a valid JSON object matching this schema:
{
	"setup_complexity": number (1-5, where 1 is minimal setup e.g. 'npm install', 5 is very complex),
	"contributing_friendliness": number (1-5, where 5 means great contributing guidelines, mentions of welcoming newbies),
	"tech_stack": array of strings,
	"prerequisites": array of strings,
	"mentorship_signals": boolean,
	"issue_debrief": string (A concise 1-2 sentence plain-english summary of what the issue actually requires),
	"tackle_plan": array of strings (3-5 concrete steps or educated guesses on how the user might start tackling this issue in the codebase)
}

README CONTENT:
` + readmeContent + `

ISSUE BODY:
` + issueBody

	var lastErr error

	for _, key := range apiKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		config := openai.DefaultConfig(key)
		// Base URL for Groq API
		config.BaseURL = "https://api.groq.com/openai/v1"
		client := openai.NewClientWithConfig(config)

		resp, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model: "openai/gpt-oss-120b",
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: "You are an expert developer helping a user. You MUST output ONLY valid JSON.",
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					},
				},
				ResponseFormat: &openai.ChatCompletionResponseFormat{
					Type: openai.ChatCompletionResponseFormatTypeJSONObject,
				},
				Temperature: 0.1,
			},
		)

		if err != nil {
			log.Printf("LLM API call failed with a token (maybe quota reached): %v. Trying next...", err)
			lastErr = err
			continue // try next key
		}

		var analysis IssueAnalysis
		responseString := strings.TrimSpace(resp.Choices[0].Message.Content)
		err = json.Unmarshal([]byte(responseString), &analysis)
		if err != nil {
			log.Printf("JSON unmarshal error from LLM response: %v. Trying next...", err)
			lastErr = fmt.Errorf("invalid json from llm: %w", err)
			continue // try next key
		}

		// Success!
		return &analysis, nil
	}

	return nil, fmt.Errorf("all API keys exhausted. last error: %w", lastErr)
}
