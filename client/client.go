package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type SignalingMessage struct {
	Description *webrtc.SessionDescription `json:"description,omitempty"`
	Candidate   *webrtc.ICECandidate       `json:"candidate,omitempty"`
}

type SignalingClient struct {
	*websocket.Conn

	done  chan struct{}
	close chan struct{}

	callbacks []func(msg *SignalingMessage)
}

func NewSignalingClient(u *url.URL) (*SignalingClient, error) {
	c := &SignalingClient{
		done:      make(chan struct{}),
		callbacks: []func(msg *SignalingMessage){},
	}

	var err error

	c.Conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", u, err)
	}

	go c.read()
	go c.run()

	return c, nil
}

func (c *SignalingClient) OnSignalingMessage(cb func(msg *SignalingMessage)) {
	c.callbacks = append(c.callbacks, cb)
}

func (c *SignalingClient) SendSignalingMessage(msg *SignalingMessage) error {
	logrus.Infof("Sending message: %#+v", msg)
	return c.Conn.WriteJSON(msg)
}

func (c *SignalingClient) Close() {
	close(c.close)
}

func (c *SignalingClient) read() {
	defer close(c.done)

	for {
		msg := &SignalingMessage{}

		if err := c.Conn.ReadJSON(msg); err != nil {
			logrus.Errorf("Failed to read: %s", err)
			return
		}

		logrus.Infof("Received message: %#+v", msg)

		for _, cb := range c.callbacks {
			cb(msg)
		}
	}
}

func (c *SignalingClient) run() {
	for {
		select {
		case <-c.done:
			return

		case <-c.close:
			logrus.Info("Closing")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logrus.Errorf("Write close: %s", err)
				return
			}

			select {
			case <-c.done:
			case <-time.After(time.Second):
			}

			return
		}
	}
}
