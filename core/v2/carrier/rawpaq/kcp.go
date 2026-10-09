package rawpaq

import (
	"errors"

	"github.com/xtaci/kcp-go/v5"
)

func applyKCP(session *kcp.UDPSession, config Config) error {
	noDelay, interval, resend, noCongestion := 0, 40, 2, 1
	writeDelay, ackNoDelay := false, true
	switch config.Mode {
	case ModeNormal:
		noDelay, interval, resend, noCongestion = 0, 40, 2, 1
	case ModeFast:
		noDelay, interval, resend, noCongestion = 0, 30, 2, 1
	case ModeFast2:
		noDelay, interval, resend, noCongestion = 1, 20, 2, 1
	case ModeFast3:
		noDelay, interval, resend, noCongestion = 1, 10, 2, 1
	case ModeManual:
		noDelay = config.Manual.NoDelay
		interval = config.Manual.IntervalMS
		resend = config.Manual.FastResend
		noCongestion = config.Manual.NoCongestion
		writeDelay = config.Manual.WriteDelay
		ackNoDelay = config.Manual.ACKNoDelay
	default:
		return errors.New("rawpaq: invalid KCP mode")
	}
	session.SetNoDelay(noDelay, interval, resend, noCongestion)
	session.SetWindowSize(config.SendWindow, config.ReceiveWindow)
	if !session.SetMtu(config.MTU) {
		return errors.New("rawpaq: KCP rejected MTU")
	}
	session.SetWriteDelay(writeDelay)
	session.SetACKNoDelay(ackNoDelay)
	session.SetStreamMode(true)
	if config.DSCP > 0 {
		_ = session.SetDSCP(config.DSCP)
	}
	return nil
}
