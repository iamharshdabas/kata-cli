package leetcode

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
	"kata-cli/internal/types"
)

func TestLeetCodeAPIFetcher_FetchFromCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "leetcode-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cachePath := filepath.Join(tmpDir, "cache.json")
	expected := map[int]*types.Problem{
		1: {
			Title:      "Two Sum",
			Difficulty: "Easy",
		},
	}
	data, _ := json.Marshal(expected)
	_ = os.WriteFile(cachePath, data, 0644)

	fetcher := &LeetCodeAPIFetcher{
		CachePath: cachePath,
	}

	result, err := fetcher.Fetch()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	p, ok := result[1]
	if !ok || p.Title != "Two Sum" {
		t.Errorf("Expected cached problem to be loaded, got %+v", result)
	}
}

func TestLeetCodeAPIFetcher_FetchFromAPI(t *testing.T) {
	// Setup a mock local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		// Mock payload response matching LeetCode structure
		payload := LeetCodeProblems{
			StatStatusPairs: []struct {
				Stat struct {
					QuestionTitle      string `json:"question__title"`
					FrontendQuestionId int    `json:"frontend_question_id"`
				} `json:"stat"`
				Difficulty struct {
					Level int `json:"level"`
				} `json:"difficulty"`
			}{
				{
					Stat: struct {
						QuestionTitle      string `json:"question__title"`
						FrontendQuestionId int    `json:"frontend_question_id"`
					}{
						QuestionTitle:      "Add Two Numbers",
						FrontendQuestionId: 2,
					},
					Difficulty: struct {
						Level int `json:"level"`
					}{
						Level: 2, // Medium
					},
				},
			},
		}

		_ = json.NewEncoder(rw).Encode(payload)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "leetcode-test-api")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cachePath := filepath.Join(tmpDir, "new_cache.json")

	fetcher := &LeetCodeAPIFetcher{
		CachePath: cachePath,
		APIURL:    server.URL,
		UserAgent: "TestAgent",
		Timeout:   2 * time.Second,
	}

	result, err := fetcher.Fetch()
	if err != nil {
		t.Fatalf("Unexpected error fetching from mock API: %v", err)
	}

	p, ok := result[2]
	if !ok {
		t.Fatal("Expected problem #2 to be fetched")
	}
	if p.Title != "Add Two Numbers" || p.Difficulty != "Med" {
		t.Errorf("Fetched problem title/difficulty incorrect: %+v", p)
	}

	// Verify it wrote the cache file
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Expected cache file to be written, but it does not exist")
	}
}
