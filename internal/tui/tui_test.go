package tui

import (
	"testing"
	"kata-cli/internal/types"

	"github.com/charmbracelet/bubbletea"
)

// MockRepository implements db.Repository
type MockRepository struct {
	Problems map[string]*types.Problem
	Stats    types.Stats
}

func (m *MockRepository) Load() (map[string]*types.Problem, types.Stats, error) {
	return m.Problems, m.Stats, nil
}

func (m *MockRepository) Save(problems map[string]*types.Problem, stats types.Stats) error {
	m.Problems = problems
	m.Stats = stats
	return nil
}

// MockFetcher implements leetcode.Fetcher
type MockFetcher struct {
	Problems map[int]*types.Problem
	Err      error
}

func (m *MockFetcher) Fetch() (map[int]*types.Problem, error) {
	return m.Problems, m.Err
}

// MockScheduler implements spacedrep.Scheduler
type MockScheduler struct {
	Interval   int
	EaseFactor float64
}

func (m *MockScheduler) SubmitReview(p *types.Problem, cooked bool) (int, float64) {
	p.Interval = m.Interval
	p.EaseFactor = m.EaseFactor
	return m.Interval, m.EaseFactor
}

// MockSyncer implements Syncer
type MockSyncer struct {
	SyncCount  int
	LastNum    string
	LastAction string
}

func (m *MockSyncer) Sync(num string, action string) error {
	m.SyncCount++
	m.LastNum = num
	m.LastAction = action
	return nil
}

func TestTUI_InitialState(t *testing.T) {
	mockRepo := &MockRepository{
		Problems: map[string]*types.Problem{
			"1": {
				Title:      "Two Sum",
				Difficulty: "Easy",
				NextReview: "2026-06-20",
				Interval:   1,
				EaseFactor: 2.5,
			},
		},
		Stats: types.Stats{CurrentStreak: 3},
	}
	mockFetcher := &MockFetcher{}
	mockScheduler := &MockScheduler{}
	mockSyncer := &MockSyncer{}

	m := NewModel(mockRepo, mockFetcher, mockScheduler, mockSyncer)

	if m.state != types.StateDashboard {
		t.Errorf("Expected initial state to be StateDashboard, got %v", m.state)
	}

	if m.stats.CurrentStreak != 3 {
		t.Errorf("Expected stats streak to be loaded as 3, got %d", m.stats.CurrentStreak)
	}

	if len(m.problems) != 1 {
		t.Errorf("Expected 1 loaded problem, got %d", len(m.problems))
	}
}

func TestTUI_ViewModeToggle(t *testing.T) {
	m := NewModel(&MockRepository{}, &MockFetcher{}, &MockScheduler{}, &MockSyncer{})

	if m.viewMode != types.ModePending {
		t.Errorf("Expected initial view mode to be ModePending, got %v", m.viewMode)
	}

	// Send Tab key to cycle view mode
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(Model)

	if m.viewMode != types.ModeEasy {
		t.Errorf("Expected view mode to cycle to ModeEasy after Tab, got %v", m.viewMode)
	}

	// Cycle through remaining tabs back to ModePending
	for i := 0; i < 4; i++ {
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updatedModel.(Model)
	}

	if m.viewMode != types.ModePending {
		t.Errorf("Expected view mode to cycle back to ModePending after 5 total tabs, got %v", m.viewMode)
	}
}

func TestTUI_StateRegisterFlow(t *testing.T) {
	mockRepo := &MockRepository{Problems: make(map[string]*types.Problem)}
	m := NewModel(mockRepo, &MockFetcher{}, &MockScheduler{}, &MockSyncer{})

	// Press 'n' to go to register state
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = updatedModel.(Model)

	if m.state != types.StateRegister {
		t.Fatalf("Expected state to transition to StateRegister, got %v", m.state)
	}

	if m.regStep != types.RegStepNumber {
		t.Errorf("Expected registration to start at RegStepNumber, got %v", m.regStep)
	}
}

func TestTUI_SearchFilter(t *testing.T) {
	mockRepo := &MockRepository{
		Problems: map[string]*types.Problem{
			"1": {
				Title:      "Two Sum",
				Difficulty: "Easy",
				NextReview: "2026-06-20",
				Interval:   1,
				EaseFactor: 2.5,
			},
			"2": {
				Title:      "Add Two Numbers",
				Difficulty: "Med",
				NextReview: "2026-06-20",
				Interval:   1,
				EaseFactor: 2.5,
			},
		},
	}
	m := NewModel(mockRepo, &MockFetcher{}, &MockScheduler{}, &MockSyncer{})
	m.viewMode = types.ModeAll
	m.updateAllList()

	// Initial check
	if len(m.getActiveList()) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(m.getActiveList()))
	}

	// Press '/' to start search
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updatedModel.(Model)

	if !m.searchActive {
		t.Fatal("Expected search to be active")
	}

	// Type "sum"
	for _, char := range "sum" {
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		m = updatedModel.(Model)
	}

	// Active list should be filtered to only "1"
	filteredList := m.getActiveList()
	if len(filteredList) != 1 {
		t.Fatalf("Expected 1 filtered item, got %d (list: %v)", len(filteredList), filteredList)
	}
	if filteredList[0] != "1" {
		t.Errorf("Expected filtered item to be '1' (Two Sum), got %s", filteredList[0])
	}

	// Press Esc to cancel search
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updatedModel.(Model)

	if m.searchActive {
		t.Fatal("Expected search to be deactivated")
	}
	if len(m.getActiveList()) != 2 {
		t.Errorf("Expected 2 items after search cleared, got %d", len(m.getActiveList()))
	}
}

func TestTUI_DeleteProblem(t *testing.T) {
	mockRepo := &MockRepository{
		Problems: map[string]*types.Problem{
			"1": {
				Title:      "Two Sum",
				Difficulty: "Easy",
				NextReview: "2026-06-20",
				Interval:   1,
				EaseFactor: 2.5,
			},
		},
	}
	mockSyncer := &MockSyncer{}
	m := NewModel(mockRepo, &MockFetcher{}, &MockScheduler{}, mockSyncer)
	m.viewMode = types.ModeAll
	m.updateAllList()

	// Hover over the item at cursor index 0 ("1") and press "d"
	m.cursor = 0
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updatedModel.(Model)

	if m.state != types.StateDeleteConfirm {
		t.Fatalf("Expected state to transition to StateDeleteConfirm, got %v", m.state)
	}
	if m.selectedKey != "1" {
		t.Errorf("Expected selectedKey to be '1', got %s", m.selectedKey)
	}

	// Confirm delete with "y"
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updatedModel.(Model)
	if cmd != nil {
		cmd()
	}

	if m.state != types.StateDashboard {
		t.Errorf("Expected state to transition back to StateDashboard, got %v", m.state)
	}
	if _, exists := m.problems["1"]; exists {
		t.Error("Expected problem #1 to be deleted from model")
	}
	if mockSyncer.SyncCount != 1 || mockSyncer.LastAction != "delete" {
		t.Errorf("Expected 1 git sync call with action 'delete', got count %d, action %s", mockSyncer.SyncCount, mockSyncer.LastAction)
	}
}

func TestTUI_VimNavigation(t *testing.T) {
	mockRepo := &MockRepository{
		Problems: map[string]*types.Problem{
			"1": {Title: "One", Difficulty: "Easy"},
			"2": {Title: "Two", Difficulty: "Med"},
			"3": {Title: "Three", Difficulty: "Hard"},
		},
	}
	m := NewModel(mockRepo, &MockFetcher{}, &MockScheduler{}, &MockSyncer{})
	m.viewMode = types.ModeAll
	m.updateAllList()

	// Initial check
	if len(m.getActiveList()) != 3 {
		t.Fatalf("Expected 3 problems, got %d", len(m.getActiveList()))
	}

	// 1. Test cycling tabs using h/l (Vim)
	// Initially ModeAll (4)
	// Press 'h' -> ModeHard (3)
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = updatedModel.(Model)
	if m.viewMode != types.ModeHard {
		t.Errorf("Expected view mode to cycle left to ModeHard, got %v", m.viewMode)
	}

	// Press 'l' -> ModeAll (4)
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = updatedModel.(Model)
	if m.viewMode != types.ModeAll {
		t.Errorf("Expected view mode to cycle right to ModeAll, got %v", m.viewMode)
	}

	// 2. Test list jumps using g/G (Vim)
	m.cursor = 0
	// Press 'G' -> jump to bottom (index 2)
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = updatedModel.(Model)
	if m.cursor != 2 {
		t.Errorf("Expected cursor at bottom (2), got %d", m.cursor)
	}

	// Press 'g' -> jump to top (index 0)
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = updatedModel.(Model)
	if m.cursor != 0 {
		t.Errorf("Expected cursor at top (0), got %d", m.cursor)
	}
}

func TestTUI_RegisterPreviewFlow(t *testing.T) {
	mockRepo := &MockRepository{Problems: make(map[string]*types.Problem)}
	mockSyncer := &MockSyncer{}
	m := NewModel(mockRepo, &MockFetcher{}, &MockScheduler{}, mockSyncer)

	// Transition to StateRegister
	m.state = types.StateRegister
	m.regStep = types.RegStepNumber
	m.regNumber.SetValue("146")

	// Mock successful leetcodeResultMsg
	msg := leetcodeResultMsg{
		num:        146,
		found:      true,
		title:      "LRU Cache",
		difficulty: "Hard",
	}

	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(Model)

	// Verify we are in RegStepPreview state
	if m.state != types.StateRegister {
		t.Errorf("Expected state to remain StateRegister, got %v", m.state)
	}
	if m.regStep != types.RegStepPreview {
		t.Errorf("Expected regStep to be RegStepPreview, got %v", m.regStep)
	}
	if m.regTitle.Value() != "LRU Cache" {
		t.Errorf("Expected regTitle to be 'LRU Cache', got %s", m.regTitle.Value())
	}
	if m.regDiffIdx != 2 { // 2 = Hard (Boss)
		t.Errorf("Expected regDiffIdx to be 2, got %d", m.regDiffIdx)
	}
	if cmd != nil {
		t.Error("Expected no command to be returned on transition to preview")
	}

	// Press Enter to confirm saving
	updatedModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedModel.(Model)
	if cmd != nil {
		cmd()
	}

	// Verify it saved and returned to dashboard
	if m.state != types.StateDashboard {
		t.Errorf("Expected state to transition back to StateDashboard, got %v", m.state)
	}
	p, ok := m.problems["146"]
	if !ok {
		t.Fatal("Expected problem #146 to be registered and saved")
	}
	if p.Title != "LRU Cache" || p.Difficulty != "Hard" {
		t.Errorf("Saved problem title/difficulty incorrect: %+v", p)
	}
	if mockSyncer.SyncCount != 1 || mockSyncer.LastAction != "add" {
		t.Errorf("Expected 1 git sync call with action 'add', got count %d, action %s", mockSyncer.SyncCount, mockSyncer.LastAction)
	}
}

func TestTUI_RegisterDuplicatePreservesStats(t *testing.T) {
	mockRepo := &MockRepository{
		Problems: map[string]*types.Problem{
			"146": {
				Title:      "Old LRU Cache",
				Difficulty: "Med",
				NextReview: "2026-06-30",
				Interval:   10,
				EaseFactor: 2.8,
				LastReview: "2026-06-20",
			},
		},
	}
	mockSyncer := &MockSyncer{}
	m := NewModel(mockRepo, &MockFetcher{}, &MockScheduler{}, mockSyncer)

	// Transition to StateRegister
	m.state = types.StateRegister
	m.regStep = types.RegStepNumber
	m.regNumber.SetValue("146")

	// Trigger preview loading for the existing problem with new title / difficulty
	msg := leetcodeResultMsg{
		num:        146,
		found:      true,
		title:      "New LRU Cache",
		difficulty: "Hard",
	}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	// Confirm that the title has changed in inputs
	if m.regTitle.Value() != "New LRU Cache" {
		t.Errorf("Expected preview title to be 'New LRU Cache', got '%s'", m.regTitle.Value())
	}

	// Press Enter to save and overwrite details
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedModel.(Model)
	if cmd != nil {
		cmd()
	}

	// Verify details were updated
	p, ok := m.problems["146"]
	if !ok {
		t.Fatal("Expected problem #146 to exist in map")
	}
	if p.Title != "New LRU Cache" || p.Difficulty != "Hard" {
		t.Errorf("Expected updated title and difficulty, got title='%s', diff='%s'", p.Title, p.Difficulty)
	}

	// Verify spaced repetition stats were PRESERVED!
	if p.NextReview != "2026-06-30" || p.Interval != 10 || p.EaseFactor != 2.8 || p.LastReview != "2026-06-20" {
		t.Errorf("Expected spaced repetition stats to be preserved, got NextReview='%s', Interval=%d, EaseFactor=%.2f, LastReview='%s'",
			p.NextReview, p.Interval, p.EaseFactor, p.LastReview)
	}

	// Git sync action should be "update"
	if mockSyncer.LastAction != "update" {
		t.Errorf("Expected git sync action to be 'update', got '%s'", mockSyncer.LastAction)
	}
}
