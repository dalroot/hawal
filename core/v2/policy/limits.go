// Package policy validates operational safety limits independently of config IO.
package policy

import (
	"errors"
	"time"
)

type Limits struct {
	MaxRecordBytes       int
	MaxQueuedBytes       int
	MaxStreams           uint32
	StreamWindowBytes    uint64
	SessionWindowBytes   uint64
	HandshakeTimeout     time.Duration
	IdleTimeout          time.Duration
	MinMigrationDwell    time.Duration
	ReconnectMinimum     time.Duration
	ReconnectMaximum     time.Duration
	MaxReconnectAttempts int
	MaxPaddingPercent    uint8
}

func Defaults() Limits {
	return Limits{
		MaxRecordBytes:       64 << 10,
		MaxQueuedBytes:       16 << 20,
		MaxStreams:           4096,
		StreamWindowBytes:    1 << 20,
		SessionWindowBytes:   16 << 20,
		HandshakeTimeout:     10 * time.Second,
		IdleTimeout:          90 * time.Second,
		MinMigrationDwell:    30 * time.Second,
		ReconnectMinimum:     500 * time.Millisecond,
		ReconnectMaximum:     30 * time.Second,
		MaxReconnectAttempts: 8,
		MaxPaddingPercent:    25,
	}
}

func (l Limits) Validate() error {
	if l.MaxRecordBytes < 1024 || l.MaxRecordBytes > 1<<20 {
		return errors.New("policy: max record bytes outside safe range")
	}
	if l.MaxQueuedBytes < l.MaxRecordBytes || l.MaxQueuedBytes > 1<<30 {
		return errors.New("policy: invalid queue byte limit")
	}
	if l.MaxStreams == 0 || l.StreamWindowBytes == 0 || l.SessionWindowBytes < l.StreamWindowBytes {
		return errors.New("policy: invalid flow-control limits")
	}
	if l.HandshakeTimeout <= 0 || l.IdleTimeout <= l.HandshakeTimeout {
		return errors.New("policy: invalid session timeouts")
	}
	if l.ReconnectMinimum <= 0 || l.ReconnectMaximum < l.ReconnectMinimum || l.MaxReconnectAttempts < 1 {
		return errors.New("policy: invalid reconnect policy")
	}
	if l.MinMigrationDwell < 0 || l.MaxPaddingPercent > 100 {
		return errors.New("policy: invalid migration or padding policy")
	}
	return nil
}
