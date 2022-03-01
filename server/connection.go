package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/VILLASframework/VILLASnode/tools/ws-relay/common"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096
)

type Connection struct {
	common.Connection

	*websocket.Conn

	Session *Session

	Messages chan SignalingMessage

	close chan struct{}
	done  chan struct{}

	Closing bool
}

func (s *Session) NewConnection(c *websocket.Conn, r *http.Request) (*Connection, error) {
	logrus.Infof("New connection from %s for session '%s'", c.RemoteAddr(), s)

	d := &Connection{
		Connection: common.Connection{
			Created:   time.Now(),
			Remote:    r.RemoteAddr,
			UserAgent: r.UserAgent(),
		},
		Conn:     c,
		Session:  s,
		Messages: make(chan SignalingMessage),
		close:    make(chan struct{}),
		done:     make(chan struct{}),
	}

	s.AddConnection(d)

	d.Conn.SetReadLimit(maxMessageSize)
	d.Conn.SetReadDeadline(time.Now().Add(pongWait))
	d.Conn.SetPongHandler(func(string) error {
		d.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go d.read()
	go d.run()

	metricConnectionsCreated.Inc()

	return d, nil
}

func (d *Connection) String() string {
	return d.Conn.RemoteAddr().String()
}

func (d *Connection) read() {
	for {
		var msg common.SignalingMessage
		if err := d.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				if !d.Closing {
					d.Closing = true
					err := d.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(5*time.Second))
					if err != nil && err != websocket.ErrCloseSent {
						logrus.Errorf("Failed to send close message: %s", err)
					}
				}
			} else {
				logrus.Errorf("Failed to read: %s", err)
			}
			break
		}

		logrus.Infof("Read signaling message from %s: %s", d.Conn.RemoteAddr(), msg)
		d.Session.Messages <- SignalingMessage{
			SignalingMessage: msg,
			Sender:           d,
		}
	}

	d.closed()
}

func (d *Connection) run() {
	ticker := time.NewTicker(pingPeriod)

loop:
	for {
		select {

		case <-d.done:
			break loop

		case msg, ok := <-d.Messages:
			if !ok {
				d.Close()
				break loop
			}

			logrus.Infof("Sending from %s to %s: %s", msg.Sender, d.Conn.RemoteAddr(), msg)

			d.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := d.Conn.WriteJSON(msg.SignalingMessage); err != nil {
				logrus.Errorf("Failed to send message: %s", err)
			}

		case <-ticker.C:
			logrus.Debug("Send ping message")

			d.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := d.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				logrus.Errorf("Failed to ping: %s", err)
			}
		}
	}
}

func (d *Connection) Close() error {
	if d.Closing {
		return errors.New("connection is closing")
	}

	d.Closing = true
	logrus.Infof("Connection closing: %s", d.Conn.RemoteAddr())

	if err := d.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		return fmt.Errorf("failed to send close message: %w", err)
	}

	select {
	case <-d.done:
	case <-time.After(time.Second):
		logrus.Warn("Timed-out waiting for connection close")
	}

	return nil
}

func (d *Connection) closed() {
	close(d.done)

	if err := d.Conn.Close(); err != nil {
		logrus.Errorf("Failed to close connection: %w", err)
	}

	logrus.Infof("Connection closed: %s", d.String())

	d.Session.RemoveConnection(d)
}
