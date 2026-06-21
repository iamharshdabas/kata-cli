package leetcode

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"kata-cli/internal/types"
)

// Fetcher defines the interface for fetching coding problems (e.g. from LeetCode).
type Fetcher interface {
	Fetch() (map[int]*types.Problem, error)
}

// LeetCodeProblems is the JSON payload structure returned by LeetCode API.
type LeetCodeProblems struct {
	StatStatusPairs []struct {
		Stat struct {
			QuestionTitle      string `json:"question__title"`
			FrontendQuestionId int    `json:"frontend_question_id"`
		} `json:"stat"`
		Difficulty struct {
			Level int `json:"level"`
		} `json:"difficulty"`
	} `json:"stat_status_pairs"`
}

// LeetCodeAPIFetcher fetches, parses, and caches problems from the LeetCode REST API.
type LeetCodeAPIFetcher struct {
	CachePath string
	APIURL    string
	UserAgent string
	Timeout   time.Duration
}

// NewLeetCodeAPIFetcher initializes a default API Fetcher.
func NewLeetCodeAPIFetcher(cachePath string) *LeetCodeAPIFetcher {
	return &LeetCodeAPIFetcher{
		CachePath: cachePath,
		APIURL:    "https://leetcode.com/api/problems/algorithms/",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		Timeout:   10 * time.Second,
	}
}

// CacheBypasser is an optional interface to bypass the internal cache.
type CacheBypasser interface {
	FetchBypassingCache() (map[int]*types.Problem, error)
}

// ReadCache reads the cache from disk without querying the API if it's not expired.
func (f *LeetCodeAPIFetcher) ReadCache() (map[int]*types.Problem, error) {
	if f.CachePath == "" {
		return nil, fmt.Errorf("no cache path configured")
	}
	info, err := os.Stat(f.CachePath)
	if err != nil {
		return nil, err
	}
	if time.Since(info.ModTime()) >= 7*24*time.Hour {
		return nil, fmt.Errorf("cache expired")
	}
	data, err := os.ReadFile(f.CachePath)
	if err != nil {
		return nil, err
	}
	var cached map[int]*types.Problem
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	return cached, nil
}

// Fetch fetches the catalog from cache if valid (less than 7 days old), or falls back to querying the LeetCode API.
func (f *LeetCodeAPIFetcher) Fetch() (map[int]*types.Problem, error) {
	if cached, err := f.ReadCache(); err == nil {
		return cached, nil
	}

	return f.FetchBypassingCache()
}

// FetchBypassingCache ignores the cache, queries LeetCode API directly, and updates the local cache.
func (f *LeetCodeAPIFetcher) FetchBypassingCache() (map[int]*types.Problem, error) {
	client := &http.Client{Timeout: f.Timeout}
	req, err := http.NewRequest("GET", f.APIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", f.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response status: %s", resp.Status)
	}

	var payload LeetCodeProblems
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	mapped := make(map[int]*types.Problem)
	for _, pair := range payload.StatStatusPairs {
		diffStr := "Easy"
		if pair.Difficulty.Level == 2 {
			diffStr = "Med"
		} else if pair.Difficulty.Level == 3 {
			diffStr = "Hard"
		}

		mapped[pair.Stat.FrontendQuestionId] = &types.Problem{
			Title:      pair.Stat.QuestionTitle,
			Difficulty: diffStr,
		}
	}

	// Cache results if cachePath is set
	if f.CachePath != "" {
		if cacheData, err := json.Marshal(mapped); err == nil {
			_ = os.WriteFile(f.CachePath, cacheData, 0644)
		}
	}

	return mapped, nil
}

// FetchAndCacheLeetCode is a backward-compatible package level function.
func FetchAndCacheLeetCode() (map[int]*types.Problem, error) {
	fetcher := NewLeetCodeAPIFetcher(".leetcode_cache.json")
	return fetcher.Fetch()
}

