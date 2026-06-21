package types

// State representing the active screen of the CLI
type State int

const (
	StateDashboard State = iota
	StateRedo
	StateRegister
	StateDeleteConfirm
)

// RegisterStep representing the active input field during registration
type RegisterStep int

const (
	RegStepNumber RegisterStep = iota
	RegStepTitle
	RegStepDifficulty
	RegStepPreview
)

// ViewMode representing the dashboard list view mode
type ViewMode int

const (
	ModePending ViewMode = iota
	ModeEasy
	ModeMed
	ModeHard
	ModeAll
)

// Problem tracks spaced repetition data for a single kata
type Problem struct {
	Title      string  `json:"title"`
	Difficulty string  `json:"difficulty"`            // "Easy", "Med", "Hard"
	NextReview string  `json:"next_review"`            // "YYYY-MM-DD"
	Interval   int     `json:"interval"`               // in days
	EaseFactor float64 `json:"ease_factor"`            // SM-2 Ease Factor
	LastReview string  `json:"last_review,omitempty"`   // "YYYY-MM-DD"
}

// Stats tracks user practice stats
type Stats struct {
	SmashedToday     int    `json:"smashed_today"`
	CurrentStreak    int    `json:"current_streak"`
	LastActivityDate string `json:"last_activity_date"` // "YYYY-MM-DD"
}
