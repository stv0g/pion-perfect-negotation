package main

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Connection struct {
	*websocket.Conn

	Session *Session

	Messages chan Message
}

func (d *Connection) String() string {
	return d.Conn.RemoteAddr().String()
}

func (d *Connection) run() {
	go d.read()
	go d.write()
}

func (d *Connection) read() {
	d.Conn.SetReadLimit(maxMessageSize)
	d.Conn.SetReadDeadline(time.Now().Add(pongWait))
	d.Conn.SetPongHandler(func(string) error {
		d.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		mt, data, err := d.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("Error: %v", err)
			}
			break
		}

		logrus.Infof("Read message from %s: %s", d.Conn.RemoteAddr(), data)

		d.Session.Messages <- Message{
			Sender: d,
			Data:   data,
			Type:   mt,
		}
	}
}

func (d *Connection) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		logrus.Infof("Connection closing: %s", d.Conn.RemoteAddr())

		ticker.Stop()
		d.Conn.Close()
	}()

loop:
	for {
		select {
		case msg, ok := <-d.Messages:
			logrus.Infof("Sending from %s to %s: %s", msg.Sender, d.Conn.RemoteAddr(), msg.Data)

			d.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				break loop
			}

			if err := d.Conn.WriteMessage(msg.Type, msg.Data); err != nil {
				logrus.Errorf("Failed to send message")
				break loop
			}

		case <-ticker.C:
			d.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			logrus.Debug("Ping")
			if err := d.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logrus.Errorf("Failed to ping: %s", err)
			}
		}
	}

	// Close
	if err := d.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
		logrus.Errorf("Failed to send message")
	}
}
