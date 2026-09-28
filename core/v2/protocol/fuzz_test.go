package protocol

import "testing"

func FuzzParseOffer(f *testing.F) {
	valid, _ := (Offer{MinVersion: 2, MaxVersion: 2, Capabilities: CapabilityStreams, MaxRecord: 4096}).MarshalBinary()
	f.Add(valid)
	f.Add([]byte{})
	f.Add([]byte("HWL1"))
	f.Fuzz(func(t *testing.T, data []byte) {
		offer, err := ParseOffer(data)
		if err != nil {
			return
		}
		roundTrip, err := offer.MarshalBinary()
		if err != nil || len(roundTrip) != 16 {
			t.Fatalf("accepted offer did not round-trip: %#v, %v", offer, err)
		}
	})
}
