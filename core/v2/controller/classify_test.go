package controller

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		in   Observation
		want FailureClass
	}{
		{"no evidence", Observation{}, FailureUnknown},
		{"local service", Observation{LocalServiceChecked: true}, FailureLocalService},
		{"mtu", Observation{LocalServiceHealthy: true, RouteReachable: true, LargePacketsFail: true, SmallPacketsPass: true}, FailureMTU},
		{"udp", Observation{LocalServiceHealthy: true, RouteReachable: true, UDPSent: true}, FailureUDPBlackhole},
		{"post handshake", Observation{LocalServiceHealthy: true, RouteReachable: true, HandshakeComplete: true, FirstPayloadSent: true}, FailurePostHandshake},
		{"healthy", Observation{LocalServiceHealthy: true, RouteReachable: true, HandshakeComplete: true, FirstPayloadSent: true, FirstPayloadSeen: true}, FailureNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.in); got.Class != tt.want {
				t.Fatalf("Classify() = %q, want %q (%#v)", got.Class, tt.want, got)
			}
		})
	}
}
