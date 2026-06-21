package db

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"kata-cli/internal/types"
)

// Repository defines the interface for loading and saving katas and stats.
type Repository interface {
	Load() (map[string]*types.Problem, types.Stats, error)
	Save(problems map[string]*types.Problem, stats types.Stats) error
}

// JSONFileRepository implements the Repository interface using a local JSON file.
type JSONFileRepository struct {
	FilePath string
}

// NewJSONFileRepository creates a new JSONFileRepository with the given file path.
func NewJSONFileRepository(filePath string) *JSONFileRepository {
	return &JSONFileRepository{FilePath: filePath}
}

func DaysSince(dateStr string) int {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 999
	}
	today, _ := time.Parse("2006-01-02", time.Now().Format("2006-01-02"))
	return int(today.Sub(t).Hours() / 24)
}

// Load reads the JSON database from the file path.
func (r *JSONFileRepository) Load() (map[string]*types.Problem, types.Stats, error) {
	probs := make(map[string]*types.Problem)
	var stats types.Stats

	data, err := os.ReadFile(r.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			stats = types.Stats{
				SmashedToday:     0,
				CurrentStreak:    0,
				LastActivityDate: "",
			}
			return probs, stats, nil
		}
		return probs, stats, fmt.Errorf("failed to read database file: %w", err)
	}

	// 1. Try to unmarshal into the new structured format
	var dbObj struct {
		Stats    types.Stats               `json:"stats"`
		Problems map[string]*types.Problem `json:"problems"`
	}
	if err := json.Unmarshal(data, &dbObj); err == nil && dbObj.Problems != nil {
		probs = dbObj.Problems
		stats = dbObj.Stats
	} else {
		// 2. Fallback to old flat format with "__stats" key
		var rawMap map[string]json.RawMessage
		if err := json.Unmarshal(data, &rawMap); err != nil {
			return probs, stats, fmt.Errorf("failed to parse database JSON: %w", err)
		}

		for k, v := range rawMap {
			if k == "__stats" {
				_ = json.Unmarshal(v, &stats)
			} else {
				var p types.Problem
				if err := json.Unmarshal(v, &p); err == nil {
					probs[k] = &p
				}
			}
		}
	}

	// Validate streak
	if stats.LastActivityDate != "" {
		diff := DaysSince(stats.LastActivityDate)
		if diff > 1 {
			stats.CurrentStreak = 0
			stats.SmashedToday = 0
		} else if diff == 1 {
			stats.SmashedToday = 0
		}
	}

	return probs, stats, nil
}

// Save writes the problems and stats to the file path atomically in structured format.
func (r *JSONFileRepository) Save(problems map[string]*types.Problem, stats types.Stats) error {
	dbObj := struct {
		Stats    types.Stats               `json:"stats"`
		Problems map[string]*types.Problem `json:"problems"`
	}{
		Stats:    stats,
		Problems: problems,
	}

	data, err := json.MarshalIndent(dbObj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal database: %w", err)
	}

	tmpFile := r.FilePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tmpFile, r.FilePath); err != nil {
		_ = os.Remove(tmpFile) // clean up temp file
		return fmt.Errorf("failed to replace database file: %w", err)
	}

	return nil
}

// LoadDatabase is a backward-compatible helper function.
func LoadDatabase() (map[string]*types.Problem, types.Stats) {
	repo := NewJSONFileRepository("db.json")
	probs, stats, err := repo.Load()
	if err != nil {
		// Log error and return empty maps instead of panicking
		fmt.Fprintf(os.Stderr, "Error loading database: %v\n", err)
	}
	return probs, stats
}

// SaveDatabase is a backward-compatible helper function.
func SaveDatabase(problems map[string]*types.Problem, stats types.Stats) {
	repo := NewJSONFileRepository("db.json")
	if err := repo.Save(problems, stats); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving database: %v\n", err)
	}
}

func RecordRedoActivity(stats *types.Stats) {
	diff := DaysSince(stats.LastActivityDate)
	if diff == 0 {
		stats.SmashedToday++
	} else {
		if diff == 1 {
			stats.CurrentStreak++
		} else {
			stats.CurrentStreak = 1
		}
		stats.SmashedToday = 1
	}
	stats.LastActivityDate = time.Now().Format("2006-01-02")
}

