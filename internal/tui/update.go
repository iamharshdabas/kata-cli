package tui

import (
	"fmt"
	"strconv"
	"strings"

	"kata-cli/internal/types"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalHeight = msg.Height
		return m, nil

	case leetcodeResultMsg:
		m.regLoading = false
		if msg.err != nil || !msg.found {
			// Fallback to manual entry
			m.regStep = types.RegStepTitle
			m.regNumber.Blur()
			m.regTitle.Focus()
			if msg.err != nil {
				m.statusErr = fmt.Errorf("LeetCode query failed: %w", msg.err)
			} else {
				m.statusMessage = "Problem not found; entering manually"
			}
			return m, textinput.Blink
		}
		// Found! Transition to preview state for user confirmation
		m.regStep = types.RegStepPreview
		m.regTitle.SetValue(msg.title)
		switch msg.difficulty {
		case "Easy":
			m.regDiffIdx = 0
		case "Med":
			m.regDiffIdx = 1
		case "Hard":
			m.regDiffIdx = 2
		default:
			m.regDiffIdx = 2
		}
		return m, nil

	case gitSyncResultMsg:
		if msg.err != nil {
			m.statusErr = fmt.Errorf("Git sync failed: %w", msg.err)
			m.statusMessage = ""
		} else {
			m.statusMessage = "Git sync complete!"
			m.statusErr = nil
		}
		return m, nil

	case tea.KeyMsg:
		// Always allow quitting from any state via Ctrl+C
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.state {
		case types.StateDashboard:
			if m.searchActive {
				switch msg.String() {
				case "esc":
					m.searchActive = false
					m.searchInput.Blur()
					m.searchInput.SetValue("")
					m.cursor = 0
					m.clampCursor()
					return m, nil

				case "enter":
					list := m.getActiveList()
					if len(list) > 0 {
						m.state = types.StateRedo
						m.selectedKey = list[m.cursor]
						m.searchActive = false
						m.searchInput.Blur()
					}
					return m, nil

				case "down", "j":
					list := m.getActiveList()
					if len(list) > 0 {
						m.cursor = (m.cursor + 1) % len(list)
					}
					m.updateFortune()
					return m, nil

				case "up", "k":
					list := m.getActiveList()
					if len(list) > 0 {
						m.cursor = (m.cursor - 1 + len(list)) % len(list)
					}
					m.updateFortune()
					return m, nil

				default:
					var searchCmd tea.Cmd
					m.searchInput, searchCmd = m.searchInput.Update(msg)
					m.cursor = 0 // Reset cursor on filter text change
					m.clampCursor()
					return m, searchCmd
				}
			}

			// Normal Dashboard Key Handling (when search is inactive)
			switch msg.String() {
			case "q":
				return m, tea.Quit

			case "j", "down":
				list := m.getActiveList()
				if len(list) > 0 {
					m.cursor = (m.cursor + 1) % len(list)
				}
				m.updateFortune()
				return m, nil

			case "k", "up":
				list := m.getActiveList()
				if len(list) > 0 {
					m.cursor = (m.cursor - 1 + len(list)) % len(list)
				}
				m.updateFortune()
				return m, nil

			case "enter":
				list := m.getActiveList()
				if len(list) > 0 {
					m.state = types.StateRedo
					m.selectedKey = list[m.cursor]
				}
				return m, nil

			case "tab", "right", "l":
				m.viewMode = types.ViewMode((int(m.viewMode) + 1) % 5)
				m.cursor = 0
				m.clampCursor()
				m.updateFortune()
				return m, nil

			case "shift+tab", "left", "h":
				m.viewMode = types.ViewMode((int(m.viewMode) - 1 + 5) % 5)
				m.cursor = 0
				m.clampCursor()
				m.updateFortune()
				return m, nil

			case "g", "home":
				m.cursor = 0
				return m, nil

			case "G", "end":
				list := m.getActiveList()
				if len(list) > 0 {
					m.cursor = len(list) - 1
				}
				return m, nil



			case "o":
				list := m.getActiveList()
				if len(list) > 0 && m.cursor < len(list) {
					key := list[m.cursor]
					p, ok := m.problems[key]
					if ok {
						url := getLeetCodeURL(p.Title)
						if err := openBrowser(url); err == nil {
							m.statusMessage = "🌐 Opened LeetCode URL in browser!"
							m.statusErr = nil
						} else {
							m.statusErr = fmt.Errorf("failed to open browser: %w", err)
						}
					}
				}
				return m, nil

			case "n":
				m.state = types.StateRegister
				m.regStep = types.RegStepNumber
				m.regNumber.Reset()
				m.regTitle.Reset()
				m.regNumber.Focus()
				m.regDiffIdx = 2 // Default to Hard (Boss)
				return m, textinput.Blink

			case "/":
				m.searchActive = true
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				m.cursor = 0
				return m, textinput.Blink

			case "d", "x":
				list := m.getActiveList()
				if len(list) > 0 {
					m.state = types.StateDeleteConfirm
					m.selectedKey = list[m.cursor]
				}
				return m, nil
			}

		case types.StateDeleteConfirm:
			switch msg.String() {
			case "y", "enter":
				// Perform deletion
				delete(m.problems, m.selectedKey)
				m.saveDatabase()
				m.updatePendingList()
				m.updateAllList()
				m.state = types.StateDashboard
				m.statusMessage = fmt.Sprintf("Deleted problem #%s", m.selectedKey)
				m.statusErr = nil
				m.updateFortune()
				return m, triggerGitSyncCmd(m.syncer, m.selectedKey, "delete")

			case "n", "esc":
				m.state = types.StateDashboard
				return m, nil
			}

		case types.StateRedo:
			switch msg.String() {
			case "esc":
				m.state = types.StateDashboard
				return m, nil

			case "1":
				cmd = m.submitRedo(true)
				m.state = types.StateDashboard
				m.updateFortune()
				return m, cmd

			case "2":
				cmd = m.submitRedo(false)
				m.state = types.StateDashboard
				m.updateFortune()
				return m, cmd
			}

		case types.StateRegister:
			if m.regLoading {
				switch msg.String() {
				case "esc":
					m.state = types.StateDashboard
					return m, nil
				}
				return m, nil
			}

			switch msg.String() {
			case "esc":
				m.state = types.StateDashboard
				return m, nil
			}

			switch m.regStep {
			case types.RegStepNumber:
				if msg.String() == "enter" {
					val := strings.TrimSpace(m.regNumber.Value())
					if val != "" && isDigitsOnly(val) {
						num, _ := strconv.Atoi(val)
						type cacheReader interface {
							ReadCache() (map[int]*types.Problem, error)
						}
						var cachedProb *types.Problem
						var foundInCache bool
						if cr, ok := m.fetcher.(cacheReader); ok {
							if cachedMap, err := cr.ReadCache(); err == nil {
								if prob, ok := cachedMap[num]; ok {
									cachedProb = prob
									foundInCache = true
								}
							}
						}
						if foundInCache {
							m.regStep = types.RegStepPreview
							m.regTitle.SetValue(cachedProb.Title)
							switch cachedProb.Difficulty {
							case "Easy":
								m.regDiffIdx = 0
							case "Med":
								m.regDiffIdx = 1
							case "Hard":
								m.regDiffIdx = 2
							default:
								m.regDiffIdx = 2
							}
							return m, nil
						}
						m.regLoading = true
						return m, queryLeetCodeCmd(m.fetcher, num)
					}
					return m, nil
				}
				m.regNumber, cmd = m.regNumber.Update(msg)
				return m, cmd

			case types.RegStepTitle:
				if msg.String() == "enter" {
					val := strings.TrimSpace(m.regTitle.Value())
					if val != "" {
						m.regStep = types.RegStepDifficulty
						m.regTitle.Blur()
					}
					return m, nil
				}
				m.regTitle, cmd = m.regTitle.Update(msg)
				return m, cmd

			case types.RegStepDifficulty:
				switch msg.String() {
				case "enter":
					cmd = m.saveNewKata()
					m.state = types.StateDashboard
					m.updateFortune()
					return m, cmd
				case "1", "2", "3":
					m.regDiffIdx = int(msg.String()[0] - '1')
					cmd = m.saveNewKata()
					m.state = types.StateDashboard
					m.updateFortune()
					return m, cmd
				case "left", "h":
					m.regDiffIdx = (m.regDiffIdx - 1 + 3) % 3
					return m, nil
				case "right", "l":
					m.regDiffIdx = (m.regDiffIdx + 1) % 3
					return m, nil
				}

			case types.RegStepPreview:
				if msg.String() == "enter" {
					cmd = m.saveNewKata()
					m.state = types.StateDashboard
					m.updateFortune()
					return m, cmd
				}
				return m, nil
			}
		}
	}

	// Fallback/Default focus text input update
	if m.state == types.StateRegister && !m.regLoading {
		if m.regStep == types.RegStepNumber {
			m.regNumber, cmd = m.regNumber.Update(msg)
		} else if m.regStep == types.RegStepTitle {
			m.regTitle, cmd = m.regTitle.Update(msg)
		}
	}

	return m, cmd
}

// getLeetCodeURL converts a problem title to a standard LeetCode URL slug.
func getLeetCodeURL(title string) string {
	title = strings.ToLower(title)
	var sb strings.Builder
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' {
			sb.WriteRune(r)
		}
	}
	cleanStr := sb.String()
	cleanStr = strings.ReplaceAll(cleanStr, " ", "-")
	for strings.Contains(cleanStr, "--") {
		cleanStr = strings.ReplaceAll(cleanStr, "--", "-")
	}
	cleanStr = strings.Trim(cleanStr, "-")
	return fmt.Sprintf("https://leetcode.com/problems/%s/", cleanStr)
}
