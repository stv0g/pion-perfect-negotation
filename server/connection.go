package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/VILLASframework/VILLASnode/tools/ws-relay/common"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Connection struct {
	*websocket.Conn

	Session *Session

	Messages chan common.SignalingMessage

	close chan struct{}
	done  chan struct{}

	isPolite  bool
	isClosing bool
}

func (s *Session) NewConnection(c *websocket.Conn) (*Connection, error) {
	logrus.Infof("New connection from %s for session '%s'", c.RemoteAddr(), s)

	s.ConnectionsMutex.Lock()
	d := &Connection{
		Conn:     c,
		Session:  s,
		Messages: make(chan common.SignalingMessage),
		isPolite: s.HasImpoliteConnection(),
		close:    make(chan struct{}),
		done:     make(chan struct{}),
	}
	s.Connections[d] = nil
	s.ConnectionsMutex.Unlock()

	if err := d.Conn.WriteJSON(&common.SignalingMessage{
		Control: &common.ControlMessage{
			Polite: d.isPolite,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to send control message: %w", err)
	}

	d.Conn.SetReadLimit(maxMessageSize)
	d.Conn.SetReadDeadline(time.Now().Add(pongWait))
	d.Conn.SetPongHandler(func(string) error {
		d.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go d.read()
	go d.run()

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
				if !d.isClosing {
					d.isClosing = true
					err := d.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(5*time.Second))
					if err != nil && err != websocket.ErrCloseSent {
						logrus.Errorf("Failed to send close message: %s", err)
					}
				}

				d.closed()
			} else {
				logrus.Errorf("Failed to read: %s", err)
			}
			break
		}

		logrus.Infof("Read message from %s: %+#v", d.Conn.RemoteAddr(), msg)
		d.Session.Messages <- msg
	}
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

			logrus.Infof("Sending from %s to %s: %+#v", msg.Sender, d.Conn.RemoteAddr(), msg)

			d.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := d.Conn.WriteJSON(msg); err != nil {
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
	if d.isClosing {
		return errors.New("connection is closing")
	}

	d.isClosing = true
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

	d.Session.ConnectionsMutex.Lock()
	delete(d.Session.Connections, d)
	d.Session.ConnectionsMutex.Unlock()

	logrus.Infof("Connection closed: %s", d.String())
}
