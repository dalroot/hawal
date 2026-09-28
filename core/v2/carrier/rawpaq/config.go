// Package rawpaq adapts a KCP session over an injected packet backend into a
// Hawal v2 carrier. The Linux PCAP backend is intentionally isolated from this
// rootless, unit-testable package.
package rawpaq

import (
	"errors"
	"fmt"
	"time"
)

type Mode string

const (
	ModeNormal Mode = "normal"
	ModeFast   Mode = "fast"
	ModeFast2  Mode = "fast2"
	ModeFast3  Mode = "fast3"
	ModeManual Mode = "manual"
)

// Config contains only RawPaq/KCP behavior. Secrets, forwarding rules and
// database state are deliberately excluded.
type Config struct {
	InterfaceName string
	LocalAddress  string
	RouterMAC     string
	Mode          Mode
	MTU           int
	SendWindow    int
	ReceiveWindow int
	DataShards    int
	ParityShards  int
	DSCP          int
	Manual        ManualTuning
	ReadTimeout   time.Duration
}

type ManualTuning struct {
	NoDelay      int
	IntervalMS   int
	FastResend   int
	NoCongestion int
	WriteDelay   bool
	ACKNoDelay   bool
}

func DefaultConfig() Config {
	return Config{
		Mode:          ModeFast3,
		LocalAddress:  "0.0.0.0:0",
		MTU:           1350,
		SendWindow:    256,
		ReceiveWindow: 1024,
		DSCP:          46,
		ReadTimeout:   30 * time.Second,
	}
}

func (c Config) Validate() error {
	switch c.Mode {
	case ModeNormal, ModeFast, ModeFast2, ModeFast3:
	case ModeManual:
		if c.Manual.IntervalMS < 5 || c.Manual.IntervalMS > 500 || c.Manual.NoDelay < 0 || c.Manual.NoDelay > 1 || c.Manual.NoCongestion < 0 || c.Manual.NoCongestion > 1 || c.Manual.FastResend < 0 || c.Manual.FastResend > 64 {
			return errors.New("rawpaq: manual KCP tuning outside safe range")
		}
	default:
		return fmt.Errorf("rawpaq: unsupported KCP mode %q", c.Mode)
	}
	if c.MTU < 256 || c.MTU > 1500 {
		return errors.New("rawpaq: MTU must be between 256 and 1500")
	}
	if c.SendWindow < 1 || c.SendWindow > 32768 || c.ReceiveWindow < 1 || c.ReceiveWindow > 32768 {
		return errors.New("rawpaq: KCP window outside safe range")
	}
	if c.DataShards < 0 || c.ParityShards < 0 || c.DataShards > 128 || c.ParityShards > 128 || c.DataShards+c.ParityShards > 255 {
		return errors.New("rawpaq: invalid FEC shard budget")
	}
	if (c.DataShards == 0) != (c.ParityShards == 0) {
		return errors.New("rawpaq: data and parity shards must both be enabled or disabled")
	}
	if c.DSCP < 0 || c.DSCP > 63 {
		return errors.New("rawpaq: DSCP must be between 0 and 63")
	}
	if c.ReadTimeout <= 0 || c.ReadTimeout > 10*time.Minute {
		return errors.New("rawpaq: invalid read timeout")
	}
	return nil
}
