package controller

import (
	"testing"
	"time"
)

func TestSelectPathUsesHysteresisAndDwell(t *testing.T) {
	now := time.Now()
	policy := SelectionPolicy{MinDwell: time.Minute, MaxSampleAge: 10 * time.Second, FailureThreshold: 3, RequiredImprovement: .2}
	samples := []PathSample{
		{ID: "a", Usable: true, RTT: 100 * time.Millisecond, ObservedAt: now},
		{ID: "b", Usable: true, RTT: 50 * time.Millisecond, ObservedAt: now},
	}
	decision, err := SelectPath("a", now.Add(-10*time.Second), now, samples, policy)
	if err != nil || decision.Change {
		t.Fatalf("early selection = %#v, %v", decision, err)
	}
	decision, err = SelectPath("a", now.Add(-2*time.Minute), now, samples, policy)
	if err != nil || !decision.Change || decision.Selected != "b" {
		t.Fatalf("mature selection = %#v, %v", decision, err)
	}
}

func TestSelectPathFailsOverFailedCurrent(t *testing.T) {
	now := time.Now()
	policy := SelectionPolicy{MinDwell: time.Hour, MaxSampleAge: time.Minute, FailureThreshold: 3, RequiredImprovement: .2}
	samples := []PathSample{
		{ID: "a", Usable: false, RTT: time.Second, ConsecutiveFailures: 3, ObservedAt: now},
		{ID: "b", Usable: true, RTT: 80 * time.Millisecond, ObservedAt: now},
	}
	decision, err := SelectPath("a", now, now, samples, policy)
	if err != nil || !decision.Change || decision.Selected != "b" {
		t.Fatalf("failover = %#v, %v", decision, err)
	}
}
