package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"kata-cli/internal/db"
	"kata-cli/internal/fortune"
	"kata-cli/internal/leetcode"
	"kata-cli/internal/spacedrep"
	"kata-cli/internal/types"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
)

// Syncer defines standard git sync behavior.
type Syncer interface {
	Sync(num string, action string) error
}

// DefaultGitSyncer runs actual git CLI commands to sync changes.
type DefaultGitSyncer struct{}

func (g *DefaultGitSyncer) Sync(num string, action string) error {
	// 1. Check if we are inside a git repository
	if err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		// Silently skip if not a git repository
		return nil
	}

	// 2. Check if a remote named 'origin' is configured
	outUrl, err := exec.Command("git", "config", "remote.origin.url").Output()
	if err != nil {
		// Remote origin is not configured (local-only repository).
		// We still commit changes locally for the user, but skip the push.
		if err := exec.Command("git", "add", "db.json").Run(); err != nil {
			return err
		}
		_ = exec.Command("git", "commit", "-m", fmt.Sprintf("chore(kata): %s #%s", action, num)).Run()
		return nil
	}

	url := strings.TrimSpace(string(outUrl))

	// 3. Prevent non-creators from pushing to the creator's upstream repo
	if strings.Contains(url, "iamharshdabas/kata-cli") {
		// Check if the current user is the creator (iamharshdabas)
		outName, _ := exec.Command("git", "config", "user.name").Output()
		outEmail, _ := exec.Command("git", "config", "user.email").Output()
		name := strings.ToLower(strings.TrimSpace(string(outName)))
		email := strings.ToLower(strings.TrimSpace(string(outEmail)))

		isCreator := strings.Contains(name, "harsh") || strings.Contains(name, "dabas") || strings.Contains(email, "harsh")
		if !isCreator {
			// Commit locally to preserve user data, but skip push to protect upstream and avoid write failures
			if err := exec.Command("git", "add", "db.json").Run(); err != nil {
				return err
			}
			_ = exec.Command("git", "commit", "-m", fmt.Sprintf("chore(kata): %s #%s", action, num)).Run()
			return fmt.Errorf("local commit only; origin points to creator's repo (see README to configure origin)")
		}
	}

	// 4. Remote origin exists, do full add, commit, and push
	if err := exec.Command("git", "add", "db.json").Run(); err != nil {
		return err
	}

	// Try to commit (this might return code 1 if there's nothing to commit, though db.json should have changed)
	commitMsg := fmt.Sprintf("chore(kata): %s #%s", action, num)
	_ = exec.Command("git", "commit", "-m", commitMsg).Run()

	// Push changes to upstream main branch
	return exec.Command("git", "push", "origin", "main").Run()
}

// openBrowser opens the specified URL in the system default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // "linux", "freebsd", "netbsd", "openbsd"
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// Model represents the bubbletea application state
type Model struct {
	state          types.State
	problems       map[string]*types.Problem
	stats          types.Stats
	pendingList    []string // Keys of problems due today
	allList        []string // Keys of all registered problems
	viewMode       types.ViewMode
	cursor         int // Dashboard navigation index
	currentFortune string

	// Redo state
	selectedKey string

	// Register state
	regStep    types.RegisterStep
	regNumber  textinput.Model
	regTitle   textinput.Model
	regDiffIdx int  // 0: Easy, 1: Med, 2: Hard
	regLoading bool // True when querying LeetCode

	// Search / Filtering state
	searchInput  textinput.Model
	searchActive bool

	// Terminal height to prevent overflow
	terminalHeight int

	// Diagnostics & Feedback
	statusMessage string
	statusErr     error

	// Dependency Injection
	repo      db.Repository
	fetcher   leetcode.Fetcher
	scheduler spacedrep.Scheduler
	syncer    Syncer
}

// Check if a string is digits only
func isDigitsOnly(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// NewModel creates a Model instance with explicit dependencies (perfect for testing and reuse)
func NewModel(repo db.Repository, fetcher leetcode.Fetcher, scheduler spacedrep.Scheduler, syncer Syncer) Model {
	numInput := textinput.New()
	numInput.Placeholder, numInput.CharLimit, numInput.Width = "146", 10, 10
	titleInput := textinput.New()
	titleInput.Placeholder, titleInput.CharLimit, titleInput.Width = "LRU Cache", 50, 30

	searchInput := textinput.New()
	searchInput.Placeholder = "Type to search..."
	searchInput.CharLimit = 30
	searchInput.Width = 30

	m := Model{
		state:        types.StateDashboard,
		viewMode:     types.ModePending,
		problems:     make(map[string]*types.Problem),
		regNumber:    numInput,
		regTitle:     titleInput,
		regDiffIdx:     2, // Default to Hard (Boss)
		searchInput:    searchInput,
		searchActive:   false,
		terminalHeight: 24, // Default sensible height
		repo:           repo,
		fetcher:        fetcher,
		scheduler:      scheduler,
		syncer:         syncer,
	}

	m.loadDatabase()
	m.updatePendingList()
	m.updateAllList()
	m.updateFortune()

	return m
}

// InitialModel returns a Model with default production configurations (retains compatibility)
func InitialModel() Model {
	return NewModel(
		db.NewJSONFileRepository("db.json"),
		leetcode.NewLeetCodeAPIFetcher(".leetcode_cache.json"),
		spacedrep.SM2Scheduler{},
		&DefaultGitSyncer{},
	)
}

// updateFortune picks a random fortune from the list
func (m *Model) updateFortune() {
	m.currentFortune = fortune.GetRandomFortune()
}

// loadDatabase reads db.json and recovers problems and stats
func (m *Model) loadDatabase() {
	var err error
	m.problems, m.stats, err = m.repo.Load()
	if err != nil {
		m.statusErr = fmt.Errorf("load db failed: %w", err)
	}
}

// saveDatabase writes data atomically to db.json
func (m *Model) saveDatabase() {
	if err := m.repo.Save(m.problems, m.stats); err != nil {
		m.statusErr = fmt.Errorf("save db failed: %w", err)
	}
}

// getActiveList returns the correct list based on active ViewMode and applies search queries if active
func (m *Model) getActiveList() []string {
	var baseList []string
	switch m.viewMode {
	case types.ModePending:
		baseList = m.pendingList
	case types.ModeEasy:
		baseList = m.getFilteredListByDifficulty("Easy")
	case types.ModeMed:
		baseList = m.getFilteredListByDifficulty("Med")
	case types.ModeHard:
		baseList = m.getFilteredListByDifficulty("Hard")
	case types.ModeAll:
		baseList = m.allList
	}

	if m.searchActive {
		query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
		if query == "" {
			return baseList
		}
		var filtered []string
		for _, key := range baseList {
			p, ok := m.problems[key]
			if !ok {
				continue
			}
			// Search matches problem ID or title
			if strings.Contains(strings.ToLower(key), query) || strings.Contains(strings.ToLower(p.Title), query) {
				filtered = append(filtered, key)
			}
		}
		return filtered
	}

	return baseList
}

// getFilteredListByDifficulty returns problems of a specific difficulty, sorted chronologically by due date.
func (m *Model) getFilteredListByDifficulty(diff string) []string {
	var list []string
	for k, p := range m.problems {
		if p.Difficulty == diff {
			list = append(list, k)
		}
	}
	sort.Slice(list, func(i, j int) bool { return m.compareProblems(list[i], list[j], true) })
	return list
}

func (m *Model) compareProblems(iKey, jKey string, compareDate bool) bool {
	pI, pJ := m.problems[iKey], m.problems[jKey]
	if compareDate && pI.NextReview != pJ.NextReview {
		return pI.NextReview < pJ.NextReview
	}
	diffPriority := map[string]int{"Hard": 1, "Med": 2, "Easy": 3}
	if diffPriority[pI.Difficulty] != diffPriority[pJ.Difficulty] {
		return diffPriority[pI.Difficulty] < diffPriority[pJ.Difficulty]
	}
	numI, _ := strconv.Atoi(iKey)
	numJ, _ := strconv.Atoi(jKey)
	return numI > numJ
}

// updatePendingList fetches and sorts due/overdue redos
func (m *Model) updatePendingList() {
	today := time.Now().Format("2006-01-02")
	var list []string
	for k, p := range m.problems {
		if p.NextReview == "" || p.NextReview <= today {
			list = append(list, k)
		}
	}
	sort.Slice(list, func(i, j int) bool { return m.compareProblems(list[i], list[j], false) })
	m.pendingList = list
	m.clampCursor()
}

// updateAllList fetches and sorts all problems by due date chronological, then difficulty, then ID
func (m *Model) updateAllList() {
	var list []string
	for k := range m.problems {
		list = append(list, k)
	}
	sort.Slice(list, func(i, j int) bool { return m.compareProblems(list[i], list[j], true) })
	m.allList = list
	m.clampCursor()
}

// clampCursor ensures dashboard cursor is within bounds
func (m *Model) clampCursor() {
	l := len(m.getActiveList())
	if m.cursor >= l {
		m.cursor = l - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// getGlobalEF returns average ease factor across all questions
func (m *Model) getGlobalEF() float64 {
	if len(m.problems) == 0 {
		return 0.0
	}
	var sum float64
	for _, p := range m.problems {
		sum += p.EaseFactor
	}
	return sum / float64(len(m.problems))
}

// submitRedo executes spaced repetition logic for the selected kata
func (m *Model) submitRedo(cooked bool) tea.Cmd {
	p, ok := m.problems[m.selectedKey]
	if !ok {
		return nil
	}

	p.LastReview = time.Now().Format("2006-01-02")
	interval, easeFactor := m.scheduler.SubmitReview(p, cooked)
	p.Interval = interval
	p.EaseFactor = easeFactor
	p.NextReview = time.Now().AddDate(0, 0, p.Interval).Format("2006-01-02")

	db.RecordRedoActivity(&m.stats)
	m.saveDatabase()
	m.updatePendingList()
	m.updateAllList()

	m.statusMessage = "Syncing changes..."
	m.statusErr = nil
	return triggerGitSyncCmd(m.syncer, m.selectedKey, "redo")
}

// saveNewKataDirect automatically saves details fetched from LeetCode
func (m *Model) saveNewKataDirect(num, title, diff string) tea.Cmd {
	var action string
	if existing, ok := m.problems[num]; ok {
		existing.Title = title
		existing.Difficulty = diff
		m.statusMessage = fmt.Sprintf("Updated problem #%s", num)
		action = "update"
	} else {
		m.problems[num] = &types.Problem{
			Title:      title,
			Difficulty: diff,
			NextReview: time.Now().AddDate(0, 0, 1).Format("2006-01-02"), // due tomorrow
			Interval:   1,                                                // First redo interval is 1 day
			EaseFactor: 2.5,
		}
		m.statusMessage = fmt.Sprintf("Added problem #%s", num)
		action = "add"
	}

	db.RecordRedoActivity(&m.stats)
	m.saveDatabase()
	m.updatePendingList()
	m.updateAllList()

	m.statusMessage = m.statusMessage + " & syncing changes..."
	m.statusErr = nil
	return triggerGitSyncCmd(m.syncer, num, action)
}

// saveNewKata creates a new kata entry manually (fallback path)
func (m *Model) saveNewKata() tea.Cmd {
	num, title := strings.TrimSpace(m.regNumber.Value()), strings.TrimSpace(m.regTitle.Value())
	diff := []string{"Easy", "Med", "Hard"}[m.regDiffIdx]

	var action string
	if existing, ok := m.problems[num]; ok {
		existing.Title = title
		existing.Difficulty = diff
		m.statusMessage = fmt.Sprintf("Updated problem #%s", num)
		action = "update"
	} else {
		m.problems[num] = &types.Problem{
			Title:      title,
			Difficulty: diff,
			NextReview: time.Now().AddDate(0, 0, 1).Format("2006-01-02"), // due tomorrow
			Interval:   1,
			EaseFactor: 2.5,
		}
		m.statusMessage = fmt.Sprintf("Added problem #%s", num)
		action = "add"
	}

	db.RecordRedoActivity(&m.stats)
	m.saveDatabase()
	m.updatePendingList()
	m.updateAllList()

	m.statusMessage = m.statusMessage + " & syncing changes..."
	m.statusErr = nil
	return triggerGitSyncCmd(m.syncer, num, action)
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
