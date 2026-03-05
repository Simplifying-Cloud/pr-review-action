package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	GitHubToken string
	RepoOwner   string
	RepoName    string
	PRNumber    int
	APIURL      string

	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string
	MaxTokens  int

	ReviewFocus string
	ExtraPrompt string
}

func Load() (*Config, error) {
	cfg := &Config{
		GitHubToken: getEnv("INPUT_GITHUB_TOKEN", os.Getenv("GITHUB_TOKEN")),
		LLMBaseURL:  os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:   os.Getenv("LLM_API_KEY"),
		LLMModel:    os.Getenv("LLM_MODEL"),
		ReviewFocus: os.Getenv("REVIEW_FOCUS"),
		ExtraPrompt: os.Getenv("EXTRA_PROMPT"),
		APIURL:      getEnv("GITHUB_API_URL", "https://api.github.com"),
	}

	maxTokens := os.Getenv("LLM_MAX_TOKENS")
	if maxTokens == "" {
		cfg.MaxTokens = 4096
	} else {
		n, err := strconv.Atoi(maxTokens)
		if err != nil {
			return nil, fmt.Errorf("invalid LLM_MAX_TOKENS: %w", err)
		}
		cfg.MaxTokens = n
	}

	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required")
	}
	if cfg.LLMBaseURL == "" {
		return nil, fmt.Errorf("LLM_BASE_URL is required")
	}
	if cfg.LLMModel == "" {
		return nil, fmt.Errorf("LLM_MODEL is required")
	}

	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY is required")
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GITHUB_REPOSITORY: %s", repo)
	}
	cfg.RepoOwner = parts[0]
	cfg.RepoName = parts[1]

	prNum, err := extractPRNumber()
	if err != nil {
		return nil, err
	}
	cfg.PRNumber = prNum

	return cfg, nil
}

func extractPRNumber() (int, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return 0, fmt.Errorf("GITHUB_EVENT_PATH is required")
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return 0, fmt.Errorf("reading event file: %w", err)
	}

	var event struct {
		PullRequest struct {
			Number int `json:"number"`
		} `json:"pull_request"`
		Number int `json:"number"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return 0, fmt.Errorf("parsing event JSON: %w", err)
	}

	if event.PullRequest.Number != 0 {
		return event.PullRequest.Number, nil
	}
	if event.Number != 0 {
		return event.Number, nil
	}
	return 0, fmt.Errorf("could not extract PR number from event payload")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
