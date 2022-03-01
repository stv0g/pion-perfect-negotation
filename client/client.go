package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/VILLASframework/VILLASnode/tools/ws-relay/common"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type SignalingClient struct {
	*websocket.Conn

	URL *url.URL

	done chan struct{}

	isClosing        bool
	reconnectTimeout time.Duration

	messageCallbacks []func(msg *common.SignalingMessage)
	connectCallbacks []func(msg *common.SignalingMessage)
}

func exponentialBackup(dur time.Duration) time.Duration {
	max := 1 * time.Minute

	dur = time.Duration(1.5 * float32(dur)).Round(time.Second)
	if dur > max {
		dur = max
	}

	return dur
}

func NewSignalingClient(u *url.URL) (*SignalingClient, error) {
	c := &SignalingClient{
		messageCallbacks: []func(msg *common.SignalingMessage){},
		connectCallbacks: []func(msg *common.SignalingMessage){},
		isClosing:        false,
		reconnectTimeout: time.Second,
		URL:              u,
	}

	return c, nil
}

func (c *SignalingClient) OnConnect(cb func(msg *common.SignalingMessage)) {
	c.connectCallbacks = append(c.connectCallbacks, cb)
}

func (c *SignalingClient) OnMessage(cb func(msg *common.SignalingMessage)) {
	c.messageCallbacks = append(c.messageCallbacks, cb)
}

func (c *SignalingClient) SendSignalingMessage(msg *common.SignalingMessage) error {
	logrus.Infof("Sending message: %#+v", msg)
	return c.Conn.WriteJSON(msg)
}

func (c *SignalingClient) Close() error {
	// Return immediatly if there is no open connection
	if c.Conn == nil {
		return nil
	}

	c.isClosing = true

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := c.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(5*time.Second))
	if err != nil {
		return fmt.Errorf("failed to send close message: %s", err)
	}

	select {
	case <-c.done:
		logrus.Infof("Connection closed")
	case <-time.After(3 * time.Second):
		logrus.Warn("Timed-out waiting for connection close")
	}

	return nil
}

func (c *SignalingClient) Connect() error {
	var err error

	dialer := websocket.Dialer{
		HandshakeTimeout: 1 * time.Second,
	}

	c.Conn, _, err = dialer.Dial(c.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %w", c.URL, err)
	}

	var msg common.SignalingMessage
	if err := c.Conn.ReadJSON(&msg); err != nil {
		return fmt.Errorf("failed to receive message: %w", err)
	}

	go c.read()

	for _, cb := range c.connectCallbacks {
		cb(&msg)
	}

	// Reset reconnect timer
	c.reconnectTimeout = 1 * time.Second

	c.done = make(chan struct{})

	return nil
}

func (c *SignalingClient) ConnectWithBackoff() error {
	t := time.NewTimer(c.reconnectTimeout)
	for range t.C {
		if err := c.Connect(); err != nil {
			c.reconnectTimeout = exponentialBackup(c.reconnectTimeout)
			t.Reset(c.reconnectTimeout)

			logrus.Errorf("Failed to connect: %s. Reconnecting in %s", err, c.reconnectTimeout)
		} else {
			break
		}
	}

	return nil
}

func (c *SignalingClient) read() {
	for {
		msg := &common.SignalingMessage{}
		if err := c.Conn.ReadJSON(msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.closed()
			} else {
				logrus.Errorf("Failed to read: %s", err)
			}
			return
		}

		logrus.Infof("Received message: %#+v", msg)

		for _, cb := range c.messageCallbacks {
			cb(msg)
		}
	}
}

func (c *SignalingClient) closed() {
	if err := c.Conn.Close(); err != nil {
		logrus.Errorf("Failed to close connection: %w", err)
	}

	c.Conn = nil

	close(c.done)

	if c.isClosing {
		logrus.Infof("Connection closed")
	} else {
		logrus.Warnf("Connection lost. Reconnecting in %s", c.reconnectTimeout)
		go c.ConnectWithBackoff()
	}
}
