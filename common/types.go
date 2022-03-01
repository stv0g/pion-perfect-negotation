package common

import "github.com/pion/webrtc/v3"

type ControlMessage struct {
	Polite bool `json:"polite"`
}

type SignalingMessage struct {
	Description *webrtc.SessionDescription `json:"description,omitempty"`
	Candidate   *webrtc.ICECandidate       `json:"candidate,omitempty"`
	Control     *ControlMessage            `json:"control"`

	Sender interface{} `json:"-"`
}
