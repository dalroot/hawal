// Package controller classifies observed link failures conservatively.
package controller

// FailureClass is a diagnosis category, not a claim about who caused it.
type FailureClass string

const (
	FailureNone                  FailureClass = "none"
	FailureLocalService          FailureClass = "local-service"
	FailureNoRoute               FailureClass = "no-route"
	FailureConnectPath           FailureClass = "connect-path"
	FailurePostHandshake         FailureClass = "post-handshake-interference-suspected"
	FailureReset                 FailureClass = "reset-observed"
	FailureUDPBlackhole          FailureClass = "udp-blackhole-suspected"
	FailureMTU                   FailureClass = "mtu-or-fragmentation-suspected"
	FailureThroughputDegradation FailureClass = "throughput-degradation"
	FailureUnknown               FailureClass = "unknown"
)

// Observation is assembled from measurements at endpoints owned by the user.
// Unknown fields must remain false instead of being inferred.
type Observation struct {
	LocalServiceChecked bool
	LocalServiceHealthy bool
	RouteChecked        bool
	RouteReachable      bool
	SYNSeenAtRemote     bool
	SYNACKSeenAtLocal   bool
	HandshakeComplete   bool
	FirstPayloadSent    bool
	FirstPayloadSeen    bool
	ResetObserved       bool
	UDPSent             bool
	UDPReceived         bool
	LargePacketsFail    bool
	SmallPacketsPass    bool
	ThroughputCollapsed bool
}

// Diagnosis records both a label and the observations supporting it.
type Diagnosis struct {
	Class      FailureClass
	Confidence float64
	Evidence   []string
}

// Classify maps endpoint observations to the narrowest justified diagnosis.
// It deliberately never returns "DPI detected": packet loss and resets have
// multiple possible causes and require controlled comparison.
func Classify(o Observation) Diagnosis {
	if o.LocalServiceChecked && !o.LocalServiceHealthy {
		return Diagnosis{FailureLocalService, 0.95, []string{"local_service_unhealthy"}}
	}
	if o.RouteChecked && !o.RouteReachable && !o.SYNSeenAtRemote {
		return Diagnosis{FailureNoRoute, 0.75, []string{"route_unreachable", "syn_not_seen_remote"}}
	}
	if o.LargePacketsFail && o.SmallPacketsPass {
		return Diagnosis{FailureMTU, 0.8, []string{"small_packets_pass", "large_packets_fail"}}
	}
	if o.UDPSent && !o.UDPReceived {
		return Diagnosis{FailureUDPBlackhole, 0.65, []string{"udp_sent", "udp_not_seen_remote"}}
	}
	if o.ResetObserved {
		return Diagnosis{FailureReset, 0.7, []string{"reset_observed"}}
	}
	if o.HandshakeComplete && o.FirstPayloadSent && !o.FirstPayloadSeen {
		return Diagnosis{FailurePostHandshake, 0.7, []string{"handshake_complete", "payload_sent", "payload_not_seen_remote"}}
	}
	if o.SYNSeenAtRemote && !o.SYNACKSeenAtLocal {
		return Diagnosis{FailureConnectPath, 0.65, []string{"syn_seen_remote", "synack_not_seen_local"}}
	}
	if o.ThroughputCollapsed {
		return Diagnosis{FailureThroughputDegradation, 0.55, []string{"throughput_collapsed"}}
	}
	if o.HandshakeComplete && (!o.FirstPayloadSent || o.FirstPayloadSeen) {
		return Diagnosis{FailureNone, 0.8, []string{"handshake_complete"}}
	}
	return Diagnosis{FailureUnknown, 0.2, []string{"insufficient_observations"}}
}
