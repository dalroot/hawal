package controller

import (
	"errors"
	"math"
	"sort"
	"time"
)

type PathSample struct {
	ID                  string
	Usable              bool
	RTT                 time.Duration
	LossPercent         float64
	ConsecutiveFailures int
	ObservedAt          time.Time
}

type SelectionPolicy struct {
	MinDwell            time.Duration
	MaxSampleAge        time.Duration
	FailureThreshold    int
	RequiredImprovement float64
}

type PathDecision struct {
	Selected   string
	Change     bool
	Reason     string
	Confidence float64
}

// SelectPath applies dwell time and hysteresis. It changes one variable (the
// primary path) and never interprets poor measurements as proof of censorship.
func SelectPath(current string, currentSince, now time.Time, samples []PathSample, policy SelectionPolicy) (PathDecision, error) {
	if policy.MinDwell < 0 || policy.MaxSampleAge <= 0 || policy.FailureThreshold < 1 || policy.RequiredImprovement <= 0 || policy.RequiredImprovement >= 1 {
		return PathDecision{}, errors.New("controller: invalid selection policy")
	}
	usable := make([]PathSample, 0, len(samples))
	var active *PathSample
	for i := range samples {
		sample := samples[i]
		if sample.ID == "" || sample.ObservedAt.IsZero() || now.Sub(sample.ObservedAt) > policy.MaxSampleAge || sample.RTT <= 0 || sample.LossPercent < 0 || sample.LossPercent > 100 {
			continue
		}
		if sample.ID == current {
			copyOfSample := sample
			active = &copyOfSample
		}
		if sample.Usable && sample.ConsecutiveFailures < policy.FailureThreshold {
			usable = append(usable, sample)
		}
	}
	if len(usable) == 0 {
		return PathDecision{Selected: current, Reason: "no_fresh_usable_alternative", Confidence: 0.3}, nil
	}
	sort.SliceStable(usable, func(i, j int) bool { return pathScore(usable[i]) < pathScore(usable[j]) })
	best := usable[0]
	if active == nil || !active.Usable || active.ConsecutiveFailures >= policy.FailureThreshold {
		return PathDecision{Selected: best.ID, Change: best.ID != current, Reason: "current_path_failed", Confidence: 0.85}, nil
	}
	if best.ID == current {
		return PathDecision{Selected: current, Reason: "current_path_best", Confidence: 0.8}, nil
	}
	if now.Sub(currentSince) < policy.MinDwell {
		return PathDecision{Selected: current, Reason: "minimum_dwell_not_reached", Confidence: 0.8}, nil
	}
	currentScore := pathScore(*active)
	if pathScore(best) > currentScore*(1-policy.RequiredImprovement) {
		return PathDecision{Selected: current, Reason: "hysteresis_kept_current", Confidence: 0.75}, nil
	}
	return PathDecision{Selected: best.ID, Change: true, Reason: "measured_path_improvement", Confidence: 0.7}, nil
}

func pathScore(sample PathSample) float64 {
	lossPenalty := 1 + math.Pow(sample.LossPercent/100, 1.5)*8
	return float64(sample.RTT) * lossPenalty
}
