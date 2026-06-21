package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"kata-cli/internal/db"
	"kata-cli/internal/types"

	"github.com/charmbracelet/lipgloss"
)

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func formatProblemLeft(key string, title string) (string, string) {
	left := fmt.Sprintf("#%-6s%s", key, title)
	padding := strings.Repeat(" ", 34-lipgloss.Width(left))
	if len(left) > 34 {
		padding = " "
	}
	return left, padding
}

func renderInputLine(stepActive bool, label, viewVal, rawVal string) string {
	cursor := "  "
	if stepActive {
		cursor = "> "
		return lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(cursor+label+viewVal) + "\n"
	}
	return cursor + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(label) + rawVal + "\n"
}

func (m Model) View() string {
	var s strings.Builder

	// 1. Boxed Header
	headerText := headerTextStyle.Render("~ / kata-cli")
	paddingLen := 57 - lipgloss.Width(headerText)
	leftPad := paddingLen / 2
	headerContent := strings.Repeat(" ", leftPad) + headerText + strings.Repeat(" ", paddingLen-leftPad)
	s.WriteString(headerBoxStyle.Render(headerContent) + "\n\n")

	// Render Top Navigation Tab Bar & Status Bar
	if m.state == types.StateDashboard || m.state == types.StateRedo {
		s.WriteString(renderTabBar(m) + "\n")
		s.WriteString(renderStatusBar(m) + "\n\n")
	}

	// 2. Main Content
	switch m.state {
	case types.StateRegister:
		s.WriteString(renderRegister(m))
	case types.StateDeleteConfirm:
		s.WriteString(renderDeleteConfirm(m))
	default:
		s.WriteString(renderDashboard(m))
	}

	// 3. Metadata, Diagnostic Info & Fortunes (right above footer)
	if m.state != types.StateRegister && m.state != types.StateDeleteConfirm {
		// Single-line selected problem metadata
		s.WriteString(renderSelectedKataMeta(m))

		if m.statusErr != nil {
			s.WriteString("  " + dangerTextStyle.Render(fmt.Sprintf("⚠️  Error: %v", m.statusErr)) + "\n")
		} else if m.statusMessage != "" {
			s.WriteString("  " + warningTextStyle.Render(fmt.Sprintf("ℹ️  %s", m.statusMessage)) + "\n")
		}
		s.WriteString("  🔮 " + mutedTextStyle.Render(m.currentFortune) + "\n\n")
	}

	// 4. Footer
	s.WriteString(renderFooter(m))

	return s.String()
}

func renderTabBar(m Model) string {
	tabs := []string{
		"Needs Redo",
		"Chill (Easy)",
		"Mid (Med)",
		"Boss (Hard)",
		"All Catalog",
	}

	var renderedTabs []string
	for idx, name := range tabs {
		if types.ViewMode(idx) == m.viewMode {
			renderedTabs = append(renderedTabs, highlightTextStyle.Render(fmt.Sprintf("● %s", name)))
		} else {
			renderedTabs = append(renderedTabs, mutedTextStyle.Render(fmt.Sprintf("○ %s", name)))
		}
	}

	return "  " + strings.Join(renderedTabs, "   ") + "\n"
}

func renderStatusBar(m Model) string {
	smashedStr := fmt.Sprintf("✓ Cooked: %s", successTextStyle.Render(strconv.Itoa(m.stats.SmashedToday)))
	streakStr := fmt.Sprintf("🔥 Streak: %s days", warningTextStyle.Render(strconv.Itoa(m.stats.CurrentStreak)))
	efStr := fmt.Sprintf("📈 Brain: %s", highlightTextStyle.Render(fmt.Sprintf("%.2f", m.getGlobalEF())))

	return fmt.Sprintf("  %s   │   %s   │   %s", streakStr, smashedStr, efStr)
}

func windowList(listLength, cursor, maxLines int) (int, int) {
	if listLength <= maxLines {
		return 0, listLength
	}
	start := cursor - maxLines/2
	if start < 0 {
		start = 0
	}
	end := start + maxLines
	if end > listLength {
		end = listLength
		start = end - maxLines
	}
	return start, end
}

func renderDashboard(m Model) string {
	var s strings.Builder
	var listTitle string
	switch m.viewMode {
	case types.ModePending:
		listTitle = "  [ N E E D S   A   R E D O ]"
	case types.ModeEasy:
		listTitle = "  [ C H I L L   K A T A S ]"
	case types.ModeMed:
		listTitle = "  [ M I D   K A T A S ]"
	case types.ModeHard:
		listTitle = "  [ B O S S   K A T A S ]"
	case types.ModeAll:
		listTitle = "  [ T H E   W H O L E   C A T A L O G ]"
	}
	s.WriteString(sectionHeaderStyle.Render(listTitle) + "\n\n")

	if m.searchActive {
		s.WriteString("  " + highlightTextStyle.Render("🔍 Filter: ") + m.searchInput.View() + "\n\n")
	}

	list := m.getActiveList()
	if len(list) == 0 {
		if m.searchActive {
			s.WriteString(warningTextStyle.Italic(true).Render(fmt.Sprintf("     🔍 No katas found matching \"%s\"", m.searchInput.Value())) + "\n\n")
		} else if m.viewMode == types.ModePending {
			s.WriteString(successTextStyle.Italic(true).Render("     🎉 All clear! Vibe check passed. absolute legend.") + "\n\n")
		} else {
			s.WriteString(warningTextStyle.Italic(true).Render("     📂 Catalog is empty. Press [n] to start grind!") + "\n\n")
		}
	} else {
		// Calculate available height for problem items
		overhead := 15
		if m.searchActive {
			overhead += 2
		}
		maxLines := m.terminalHeight - overhead
		if maxLines < 3 {
			maxLines = 3 // Always show at least 3 items to avoid bounds error
		}

		start, end := windowList(len(list), m.cursor, maxLines)

		if start > 0 {
			s.WriteString(mutedTextStyle.Render("     ▲ ... more above ...") + "\n")
		}

		for i := start; i < end; i++ {
			key := list[i]
			p, ok := m.problems[key]
			if !ok {
				continue
			}
			left, padding := formatProblemLeft(key, p.Title)
			if m.state == types.StateRedo {
				if key == m.selectedKey {
					s.WriteString(fmt.Sprintf("     %s%s%s\n\n", left, padding, formatDifficulty(p.Difficulty)))

					nextDays := 6
					if p.Interval >= 6 {
						nextDays = int(math.Round(float64(p.Interval) * p.EaseFactor))
					}

					s.WriteString(highlightTextStyle.Render("            How'd it go? did you cook?") + "\n")
					s.WriteString(successTextStyle.Render(fmt.Sprintf("            [1] Cooked (Next: %d days)", nextDays)) + "\n")
					s.WriteString(dangerTextStyle.Render("            [2] Fumbled  (Do Tomorrow)") + "\n\n")
				} else {
					tag := fmt.Sprintf("[    %s ]", p.Difficulty)
					s.WriteString(mutedTextStyle.Render(fmt.Sprintf("     %s%s%s", left, padding, tag)) + "\n")
				}
			} else {
				selector := "     "
				lineStyle := lipgloss.NewStyle()
				if i == m.cursor {
					selector = "  >  "
					lineStyle = highlightTextStyle
				}
				dueInfo := ""
				if m.viewMode == types.ModeAll || m.viewMode == types.ModeEasy || m.viewMode == types.ModeMed || m.viewMode == types.ModeHard {
					todayStr := time.Now().Format("2006-01-02")
					if p.NextReview <= todayStr {
						dueInfo = "  " + dangerTextStyle.Render("(Due)")
					} else {
						days := -db.DaysSince(p.NextReview)
						dueInfo = "  " + mutedTextStyle.Render(fmt.Sprintf("(In %dd)", days))
					}
				}
				s.WriteString(selector + lineStyle.Render(left+padding) + formatDifficulty(p.Difficulty) + dueInfo + "\n")
			}
		}

		if end < len(list) {
			s.WriteString(mutedTextStyle.Render("     ▼ ... more below ...") + "\n")
		}
		s.WriteString("\n")
	}

	return s.String()
}

func renderSelectedKataMeta(m Model) string {
	list := m.getActiveList()
	if len(list) == 0 || m.cursor >= len(list) {
		return ""
	}
	key := list[m.cursor]
	p, ok := m.problems[key]
	if !ok {
		return ""
	}

	todayStr := time.Now().Format("2006-01-02")
	var statusVal string
	if p.NextReview <= todayStr {
		diff := db.DaysSince(p.NextReview)
		if diff == 0 {
			statusVal = dangerTextStyle.Render("Due Today")
		} else {
			statusVal = dangerTextStyle.Render(fmt.Sprintf("Overdue %dd", diff))
		}
	} else {
		diff := -db.DaysSince(p.NextReview)
		statusVal = successTextStyle.Render(fmt.Sprintf("In %dd", diff))
	}

	lastReviewVal := "Never"
	if p.LastReview != "" {
		daysAgo := db.DaysSince(p.LastReview)
		if daysAgo == 0 {
			lastReviewVal = "Today"
		} else {
			lastReviewVal = fmt.Sprintf("%dd ago", daysAgo)
		}
	}

	// Minimal, clean single-line format:
	// 🎯 #1 │ Chill │ Due Today │ Last: 2d ago │ Next: 2026-06-22 │ Int: 1d │ EF: 2.50
	return fmt.Sprintf(
		"  🎯 %s │ %s │ %s │ Last: %s │ Next: %s │ Int: %dd │ EF: %.2f\n",
		highlightTextStyle.Render("#"+key),
		formatDifficulty(p.Difficulty),
		statusVal,
		lastReviewVal,
		p.NextReview,
		p.Interval,
		p.EaseFactor,
	)
}

func renderRegister(m Model) string {
	var s strings.Builder
	s.WriteString(sectionHeaderStyle.Render("  [ A D D   T O   T H E   G R I N D ]") + "\n\n")

	if m.regLoading {
		s.WriteString(highlightTextStyle.Render("  🔄 Cache miss hits different. Updating local cache from LeetCode API... let me cook! 🧑‍🍳") + "\n\n")
	} else if m.regStep == types.RegStepPreview {
		s.WriteString(renderInputLine(false, "ID/No.     : ", "", m.regNumber.Value()))
		s.WriteString(renderInputLine(false, "Name       : ", "", m.regTitle.Value()))
		diffStr := []string{"Easy", "Med", "Hard"}[m.regDiffIdx]
		s.WriteString("  " + mutedTextStyle.Render("Tier       : ") + formatDifficulty(diffStr) + "\n\n")
		// Warn if updating an existing problem
		num := strings.TrimSpace(m.regNumber.Value())
		if _, exists := m.problems[num]; exists {
			s.WriteString("  " + warningTextStyle.Render("⚠️  Problem already in catalog! Saving preserves spacing stats.") + "\n\n")
		} else {
			s.WriteString("  " + successTextStyle.Render("✨ Preview loaded successfully!") + "\n\n")
		}
	} else {
		s.WriteString(renderInputLine(m.regStep == types.RegStepNumber, "ID/No.     : ", m.regNumber.View(), m.regNumber.Value()))
		s.WriteString(renderInputLine(m.regStep == types.RegStepTitle, "Name       : ", m.regTitle.View(), m.regTitle.Value()))

		diffCursor := "  "
		if m.regStep == types.RegStepDifficulty {
			diffCursor = "> "
		}
		diffOpts := make([]string, 3)
		for idx, opt := range []string{"Chill", "Mid", "Boss"} {
			if idx == m.regDiffIdx {
				diffOpts[idx] = highlightTextStyle.Render(fmt.Sprintf("[%d] *%s", idx+1, opt))
			} else {
				diffOpts[idx] = mutedTextStyle.Render(fmt.Sprintf("[%d]  %s", idx+1, opt))
			}
		}
		s.WriteString(diffCursor + mutedTextStyle.Render("Tier       : ") + strings.Join(diffOpts, "  ") + "\n\n")

		// Warn in manual difficulty step if updating an existing problem
		num := strings.TrimSpace(m.regNumber.Value())
		if m.regStep == types.RegStepDifficulty {
			if _, exists := m.problems[num]; exists {
				s.WriteString("  " + warningTextStyle.Render("⚠️  Problem already in catalog! Saving preserves spacing stats.") + "\n\n")
			}
		}
	}
	return s.String()
}

func renderDeleteConfirm(m Model) string {
	var s strings.Builder
	s.WriteString(sectionHeaderStyle.Render("  [ D E L E T E   C O N F I R M A T I O N ]") + "\n\n")

	p, ok := m.problems[m.selectedKey]
	if !ok {
		s.WriteString("  Error: selected problem not found.\n\n")
		return s.String()
	}

	s.WriteString(dangerTextStyle.Render(fmt.Sprintf("  ⚠️  Are you sure you want to delete problem #%s?", m.selectedKey)) + "\n")
	s.WriteString("     Title: " + p.Title + "\n")
	s.WriteString("     Tier : " + formatDifficulty(p.Difficulty) + "\n\n")

	return s.String()
}

func renderFooter(m Model) string {
	var helpContent string
	switch m.state {
	case types.StateDashboard:
		if m.searchActive {
			helpContent = "  [esc] clear search   [enter] select   [↓/↑] scroll"
		} else {
			helpContent = "  [j/k] scroll   [h/l/tab] tabs   [g/G] top/bot   [enter] redo   [/] search   [o] open   [d] delete   [q] exit"
		}
	case types.StateRedo:
		helpContent = "  [1/2] select   [esc] cancel"
	case types.StateDeleteConfirm:
		helpContent = "  [y/enter] delete   [n/esc] cancel"
	case types.StateRegister:
		if m.regLoading {
			helpContent = "  [esc] cancel"
		} else {
			switch m.regStep {
			case types.RegStepNumber, types.RegStepTitle:
				helpContent = "  [enter] next   [esc] cancel"
			case types.RegStepDifficulty:
				helpContent = "  [1/2/3] select   [enter] save   [esc] cancel"
			case types.RegStepPreview:
				helpContent = "  [enter] save & proceed   [esc] cancel"
			}
		}
	}
	return dividerStyle.Render(strings.Repeat("─", 59)) + "\n" + helpStyle.Render(helpContent) + "\n"
}
