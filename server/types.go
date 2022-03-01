package main

import "github.com/VILLASframework/VILLASnode/tools/ws-relay/common"

type SignalingMessage struct {
	common.SignalingMessage

	Sender *Connection
}

func (msg *SignalingMessage) CollectMetrics() {
	if msg.Candidate != nil {
		metricMessagesReceived.WithLabelValues("candidate").Inc()

		c := msg.Candidate

		metricCandidateTypes.WithLabelValues(c.Typ.String(), c.Protocol.String(), c.TCPType).Inc()

	}
	if msg.Description != nil {
		metricMessagesReceived.WithLabelValues("description").Inc()
	}
}
