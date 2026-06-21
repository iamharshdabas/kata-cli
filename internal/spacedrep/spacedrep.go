package spacedrep

import (
	"math"
	"kata-cli/internal/types"
)

// Scheduler defines the behavior for any spaced repetition scheduling algorithm.
type Scheduler interface {
	// SubmitReview updates the problem state and returns the new review interval (in days) and ease factor.
	SubmitReview(p *types.Problem, cooked bool) (interval int, easeFactor float64)
}

// SM2Scheduler implements a variant of the SuperMemo-2 (SM-2) algorithm.
type SM2Scheduler struct{}

// SubmitReview calculates the new spaced repetition interval and ease factor under SM-2.
func (s SM2Scheduler) SubmitReview(p *types.Problem, cooked bool) (int, float64) {
	interval := p.Interval
	easeFactor := p.EaseFactor

	if cooked {
		if interval <= 1 {
			interval = 6
		} else {
			interval = int(math.Round(float64(interval) * easeFactor))
		}
		easeFactor += 0.1
	} else {
		interval = 1
		easeFactor -= 0.2
	}

	if easeFactor < 1.3 {
		easeFactor = 1.3
	}

	p.Interval = interval
	p.EaseFactor = easeFactor

	return interval, easeFactor
}

// SubmitRedo is a backward-compatible wrapper that uses the default SM2Scheduler.
func SubmitRedo(p *types.Problem, cooked bool) int {
	scheduler := SM2Scheduler{}
	interval, _ := scheduler.SubmitReview(p, cooked)
	return interval
}

