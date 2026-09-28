package shaping

import (
	"testing"
	"time"
)

func TestBudgetRejectsExcess(t *testing.T) {
	budget := Budget{MaxExtraBytes: 100, MaxDelay: 10 * time.Millisecond}
	input := Input{RecordBytes: 1000}
	if err := budget.Validate(input, Decision{BucketBytes: 1200}); err == nil {
		t.Fatal("excess padding accepted")
	}
	if err := budget.Validate(input, Decision{BucketBytes: 1000, Delay: 20 * time.Millisecond}); err == nil {
		t.Fatal("excess delay accepted")
	}
}
