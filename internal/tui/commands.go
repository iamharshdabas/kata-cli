package tui

import (
	"kata-cli/internal/leetcode"

	"github.com/charmbracelet/bubbletea"
)

type leetcodeResultMsg struct {
	num        int
	found      bool
	title      string
	difficulty string
	err        error
}

type gitSyncResultMsg struct {
	err error
}

// queryLeetCodeCmd fetches problem metadata asynchronously using the injected fetcher,
// bypassing the cache on a cache miss.
func queryLeetCodeCmd(fetcher leetcode.Fetcher, num int) tea.Cmd {
	return func() tea.Msg {
		mapped, err := fetcher.Fetch()
		if err != nil {
			return leetcodeResultMsg{num: num, err: err}
		}
		prob, found := mapped[num]
		if !found {
			// If not found in the cache, attempt to bypass cache and fetch fresh from API
			if bypasser, ok := fetcher.(leetcode.CacheBypasser); ok {
				if freshMapped, freshErr := bypasser.FetchBypassingCache(); freshErr == nil {
					mapped = freshMapped
					prob, found = mapped[num]
				}
			}
		}

		if !found {
			return leetcodeResultMsg{num: num, found: false, err: err}
		}

		return leetcodeResultMsg{
			num:        num,
			found:      true,
			title:      prob.Title,
			difficulty: prob.Difficulty,
		}
	}
}

// triggerGitSyncCmd initiates a background Git sync task via bubbletea commands.
func triggerGitSyncCmd(syncer Syncer, num string, action string) tea.Cmd {
	return func() tea.Msg {
		if syncer == nil {
			return gitSyncResultMsg{err: nil}
		}
		err := syncer.Sync(num, action)
		return gitSyncResultMsg{err: err}
	}
}
