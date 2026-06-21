package spacedrep

import (
	"math"
	"testing"
	"kata-cli/internal/types"
)

func TestSM2Scheduler_SubmitReview_Cooked(t *testing.T) {
	scheduler := SM2Scheduler{}

	// Test case 1: Initial review (Interval <= 1)
	p1 := &types.Problem{
		Interval:   1,
		EaseFactor: 2.5,
	}
	interval, ef := scheduler.SubmitReview(p1, true)
	if interval != 6 {
		t.Errorf("Expected interval to be 6, got %d", interval)
	}
	if ef != 2.6 {
		t.Errorf("Expected ease factor to increase to 2.6, got %f", ef)
	}

	// Test case 2: Subsequent review
	p2 := &types.Problem{
		Interval:   6,
		EaseFactor: 2.6,
	}
	expectedInterval := int(math.Round(6.0 * 2.6))
	interval, ef = scheduler.SubmitReview(p2, true)
	if interval != expectedInterval {
		t.Errorf("Expected interval to be %d, got %d", expectedInterval, interval)
	}
	if ef != 2.7 {
		t.Errorf("Expected ease factor to increase to 2.7, got %f", ef)
	}
}

func TestSM2Scheduler_SubmitReview_Fumbled(t *testing.T) {
	scheduler := SM2Scheduler{}

	p := &types.Problem{
		Interval:   10,
		EaseFactor: 1.4,
	}

	interval, ef := scheduler.SubmitReview(p, false)
	if interval != 1 {
		t.Errorf("Expected interval to reset to 1, got %d", interval)
	}
	if ef != 1.3 { // 1.4 - 0.2 capped at 1.3
		t.Errorf("Expected ease factor to be capped at 1.3, got %f", ef)
	}
}
