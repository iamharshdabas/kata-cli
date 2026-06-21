package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"kata-cli/internal/types"
)

func TestDaysSince(t *testing.T) {
	todayStr := time.Now().Format("2006-01-02")
	if diff := DaysSince(todayStr); diff != 0 {
		t.Errorf("Expected 0 days since today, got %d", diff)
	}

	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if diff := DaysSince(yesterdayStr); diff != 1 {
		t.Errorf("Expected 1 day since yesterday, got %d", diff)
	}

	invalidStr := "not-a-date"
	if diff := DaysSince(invalidStr); diff != 999 {
		t.Errorf("Expected invalid date to return 999, got %d", diff)
	}
}

func TestRecordRedoActivity(t *testing.T) {
	stats := &types.Stats{
		SmashedToday:     0,
		CurrentStreak:    0,
		LastActivityDate: "",
	}

	// First activity ever
	RecordRedoActivity(stats)
	if stats.SmashedToday != 1 {
		t.Errorf("Expected SmashedToday to be 1, got %d", stats.SmashedToday)
	}
	if stats.CurrentStreak != 1 {
		t.Errorf("Expected CurrentStreak to be 1, got %d", stats.CurrentStreak)
	}
	todayStr := time.Now().Format("2006-01-02")
	if stats.LastActivityDate != todayStr {
		t.Errorf("Expected LastActivityDate to be today (%s), got %s", todayStr, stats.LastActivityDate)
	}

	// Second activity today
	RecordRedoActivity(stats)
	if stats.SmashedToday != 2 {
		t.Errorf("Expected SmashedToday to increment to 2, got %d", stats.SmashedToday)
	}
	if stats.CurrentStreak != 1 { // Streak shouldn't increase twice on the same day
		t.Errorf("Expected CurrentStreak to remain 1, got %d", stats.CurrentStreak)
	}
}

func TestJSONFileRepository_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kata-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_db.json")
	repo := NewJSONFileRepository(dbPath)

	// Load non-existent database (should initialize clean empty)
	probs, stats, err := repo.Load()
	if err != nil {
		t.Fatalf("Unexpected error loading non-existent db: %v", err)
	}
	if len(probs) != 0 {
		t.Errorf("Expected 0 problems, got %d", len(probs))
	}
	if stats.CurrentStreak != 0 {
		t.Errorf("Expected streak to be 0, got %d", stats.CurrentStreak)
	}

	// Save some data
	testProbs := map[string]*types.Problem{
		"1": {
			Title:      "Two Sum",
			Difficulty: "Easy",
			NextReview: "2026-06-22",
			Interval:   1,
			EaseFactor: 2.5,
		},
	}
	testStats := types.Stats{
		SmashedToday:     1,
		CurrentStreak:    5,
		LastActivityDate: time.Now().Format("2006-01-02"),
	}

	err = repo.Save(testProbs, testStats)
	if err != nil {
		t.Fatalf("Failed to save database: %v", err)
	}

	// Load back
	loadedProbs, loadedStats, err := repo.Load()
	if err != nil {
		t.Fatalf("Failed to load database: %v", err)
	}

	p, ok := loadedProbs["1"]
	if !ok {
		t.Fatal("Expected problem #1 to exist")
	}
	if p.Title != "Two Sum" || p.Difficulty != "Easy" {
		t.Errorf("Problem title/difficulty incorrect, got %s / %s", p.Title, p.Difficulty)
	}
	if loadedStats.CurrentStreak != 5 {
		t.Errorf("Expected streak to be 5, got %d", loadedStats.CurrentStreak)
	}
}
