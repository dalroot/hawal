// Package shaping defines a budgeted policy boundary. It intentionally ships
// without a traffic-mimicry profile until owned capture data exists.
package shaping

import (
	"errors"
	"time"
)

type Direction uint8

const (
	DirectionOutbound Direction = iota + 1
	DirectionInbound
)

type Input struct {
	Direction         Direction
	RecordBytes       int
	StreamClass       uint8
	SinceHandshake    time.Duration
	RTT               time.Duration
	LossPercent       float64
	RemainingOverhead int64
}

type Decision struct {
	BucketBytes int
	Delay       time.Duration
	Coalesce    bool
}

type Policy interface {
	Decide(Input) Decision
}

// Disabled preserves benchmark behavior and adds no delay or padding.
type Disabled struct{}

func (Disabled) Decide(input Input) Decision { return Decision{BucketBytes: input.RecordBytes} }

type Budget struct {
	MaxExtraBytes int
	MaxDelay      time.Duration
}

func (b Budget) Validate(input Input, decision Decision) error {
	if decision.BucketBytes < input.RecordBytes {
		return errors.New("shaping: bucket cannot truncate a record")
	}
	if decision.BucketBytes-input.RecordBytes > b.MaxExtraBytes {
		return errors.New("shaping: overhead budget exceeded")
	}
	if decision.Delay < 0 || decision.Delay > b.MaxDelay {
		return errors.New("shaping: delay budget exceeded")
	}
	return nil
}
