package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/VILLASframework/VILLASnode/tools/ws-relay/common"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type PeerConnection struct {
	*webrtc.PeerConnection
	*SignalingClient

	makingOffer                  bool
	ignoreOffer                  bool
	polite                       bool
	isSettingRemoteAnswerPending bool
}

func NewPeerConnection(u *url.URL) (*PeerConnection, error) {
	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	sc, err := NewSignalingClient(u)
	if err != nil {
		return nil, fmt.Errorf("Failed to create signaling client: %w", err)
	}

	ppc := &PeerConnection{
		SignalingClient: sc,
		PeerConnection:  pc,

		makingOffer:                  false,
		ignoreOffer:                  false,
		isSettingRemoteAnswerPending: false,
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	ppc.PeerConnection.OnICEConnectionStateChange(ppc.OnICEConnectionStateChangeHandler)
	ppc.PeerConnection.OnConnectionStateChange(ppc.OnConnectionStateChangeHandler)
	ppc.PeerConnection.OnSignalingStateChange(ppc.OnSignalingStateChangeHandler)
	ppc.PeerConnection.OnICECandidate(ppc.OnICECandidateHandler)
	ppc.PeerConnection.OnNegotiationNeeded(ppc.OnNegotiationNeededHandler)
	ppc.PeerConnection.OnDataChannel(ppc.OnDataChannelHandler)

	ppc.SignalingClient.OnMessage(ppc.OnSignalingMessageHandler)
	ppc.SignalingClient.OnConnect(func(msg *common.SignalingMessage) {
		ppc.polite = msg.Control != nil && msg.Control.Polite

		logrus.Infof("Connected: polite=%v", ppc.polite)
	})

	if err := ppc.SignalingClient.ConnectWithBackoff(); err != nil {
		return nil, fmt.Errorf("failed to connect signaling client: %w", err)
	}

	_, err = ppc.CreateDataChannel("test", nil)
	if err != nil {
		logrus.Panicf("Failed to create datachannel: %s", err)
	}

	return ppc, nil
}

func (pc *PeerConnection) OnICECandidateHandler(c *webrtc.ICECandidate) {
	if c == nil {
		logrus.Info("Candidate gathering concluded")
		return
	} else {
		logrus.Infof("Found new candidate: %s", c)
	}

	pc.SendSignalingMessage(&common.SignalingMessage{
		Candidate: c,
	})
}

func (pc *PeerConnection) OnNegotiationNeededHandler() {
	logrus.Info("Negotation needed!")

	pc.makingOffer = true
	defer func() { pc.makingOffer = false }()

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		logrus.Panicf("Failed to create offer: %s", err)
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		logrus.Panicf("Failed to set local description: %s", err)
	}

	if err := pc.SendSignalingMessage(&common.SignalingMessage{
		Description: &offer,
	}); err != nil {
		logrus.Panicf("Failed to send offer: %s", err)
	}
}

func (pc *PeerConnection) OnSignalingStateChangeHandler(ss webrtc.SignalingState) {
	logrus.Infof("Signaling State has changed: %s", ss.String())
}

func (pc *PeerConnection) OnConnectionStateChangeHandler(pcs webrtc.PeerConnectionState) {
	logrus.Infof("Connection State has changed: %s", pcs.String())
}

func (pc *PeerConnection) OnICEConnectionStateChangeHandler(connectionState webrtc.ICEConnectionState) {
	logrus.Infof("ICE Connection State has changed: %s", connectionState.String())
}

func (pc *PeerConnection) OnSignalingMessageHandler(msg *common.SignalingMessage) {
	if msg.Description != nil {
		// An offer may come in while we are busy processing SRD(answer).
		// In this case, we will be in "stable" by the time the offer is processed
		// so it is safe to chain it on our Operations Chain now.
		readyForOffer := !pc.makingOffer &&
			(pc.SignalingState() == webrtc.SignalingStateStable || pc.isSettingRemoteAnswerPending)
		offerCollision := msg.Description.Type == webrtc.SDPTypeOffer && !readyForOffer

		pc.ignoreOffer = !pc.polite && offerCollision
		if pc.ignoreOffer {
			return
		}

		pc.isSettingRemoteAnswerPending = msg.Description.Type == webrtc.SDPTypeAnswer
		pc.SetRemoteDescription(*msg.Description) // SRD rolls back as needed
		pc.isSettingRemoteAnswerPending = false

		if msg.Description.Type == webrtc.SDPTypeOffer {
			// Rollback!!!
			if err := pc.SetLocalDescription(webrtc.SessionDescription{}); err != nil {
				logrus.Panicf("Failed to rollback signaling state: %s", err)
			}

			pc.SendSignalingMessage(&common.SignalingMessage{
				Description: pc.LocalDescription(),
			})
		}
	} else if msg.Candidate != nil {
		if err := pc.AddICECandidate(msg.Candidate.ToJSON()); err != nil {
			if !pc.ignoreOffer {
				logrus.Panicf("Failed to add new ICE candidate: %s", err)
			}

		}
	}
}

func (pc *PeerConnection) OnDataChannelHandler(dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		logrus.Info("Datachannel opened")
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		logrus.Infof("Received: %s", msg.Data)
	})

	i := 0
	for {
		msg := fmt.Sprintf("Hello %d", i)

		logrus.Infof("Send: %s", msg)

		dc.SendText(msg)
		time.Sleep(1 * time.Second)

		i++
	}
}

func (pc *PeerConnection) Close() {
	pc.SignalingClient.Close()
	pc.PeerConnection.Close()
}
