// Package protocol contains data negotiated only after a secure handshake.
package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	CurrentVersion uint16 = 2
	maxNegotiation        = 64
)

type Capability uint64

const (
	CapabilityStreams Capability = 1 << iota
	CapabilityDatagrams
	CapabilityMigration
	CapabilityKeyUpdate
)

// Offer is encrypted inside the secure session; it is never a cleartext wire
// prefix or protocol fingerprint.
type Offer struct {
	MinVersion   uint16
	MaxVersion   uint16
	Capabilities Capability
	MaxRecord    uint32
}

type Selection struct {
	Version      uint16
	Capabilities Capability
	MaxRecord    uint32
}

func Negotiate(local, remote Offer) (Selection, error) {
	min := local.MinVersion
	if remote.MinVersion > min {
		min = remote.MinVersion
	}
	max := local.MaxVersion
	if remote.MaxVersion < max {
		max = remote.MaxVersion
	}
	if min == 0 || min > max {
		return Selection{}, errors.New("protocol: no common version")
	}
	maxRecord := local.MaxRecord
	if remote.MaxRecord < maxRecord {
		maxRecord = remote.MaxRecord
	}
	if maxRecord < 1024 {
		return Selection{}, errors.New("protocol: negotiated record limit is too small")
	}
	return Selection{Version: max, Capabilities: local.Capabilities & remote.Capabilities, MaxRecord: maxRecord}, nil
}

func (o Offer) MarshalBinary() ([]byte, error) {
	if o.MinVersion == 0 || o.MaxVersion < o.MinVersion || o.MaxRecord < 1024 {
		return nil, errors.New("protocol: invalid offer")
	}
	buf := make([]byte, 16)
	binary.BigEndian.PutUint16(buf[0:2], o.MinVersion)
	binary.BigEndian.PutUint16(buf[2:4], o.MaxVersion)
	binary.BigEndian.PutUint64(buf[4:12], uint64(o.Capabilities))
	binary.BigEndian.PutUint32(buf[12:16], o.MaxRecord)
	return buf, nil
}

func ParseOffer(data []byte) (Offer, error) {
	if len(data) != 16 || len(data) > maxNegotiation {
		return Offer{}, fmt.Errorf("protocol: invalid offer length %d", len(data))
	}
	o := Offer{
		MinVersion:   binary.BigEndian.Uint16(data[0:2]),
		MaxVersion:   binary.BigEndian.Uint16(data[2:4]),
		Capabilities: Capability(binary.BigEndian.Uint64(data[4:12])),
		MaxRecord:    binary.BigEndian.Uint32(data[12:16]),
	}
	if _, err := o.MarshalBinary(); err != nil {
		return Offer{}, err
	}
	return o, nil
}
