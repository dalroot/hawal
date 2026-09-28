package protocol

import "testing"

func TestOfferRoundTripAndNegotiation(t *testing.T) {
	local := Offer{MinVersion: 2, MaxVersion: 3, Capabilities: CapabilityStreams | CapabilityMigration, MaxRecord: 64 << 10}
	wire, err := local.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseOffer(wire)
	if err != nil || parsed != local {
		t.Fatalf("ParseOffer() = %#v, %v", parsed, err)
	}
	remote := Offer{MinVersion: 2, MaxVersion: 2, Capabilities: CapabilityStreams | CapabilityDatagrams, MaxRecord: 32 << 10}
	selected, err := Negotiate(local, remote)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Version != 2 || selected.Capabilities != CapabilityStreams || selected.MaxRecord != 32<<10 {
		t.Fatalf("unexpected selection %#v", selected)
	}
}

func TestNegotiationRejectsNoCommonVersion(t *testing.T) {
	_, err := Negotiate(
		Offer{MinVersion: 2, MaxVersion: 2, MaxRecord: 4096},
		Offer{MinVersion: 3, MaxVersion: 3, MaxRecord: 4096},
	)
	if err == nil {
		t.Fatal("incompatible versions accepted")
	}
}
