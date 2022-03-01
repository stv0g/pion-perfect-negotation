package main

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stv0g/pion-perfect-negotation/pkg"
)

type Session struct {
	Name    string
	Created time.Time

	Messages chan SignalingMessage

	Connections      map[*Connection]interface{}
	ConnectionsMutex sync.RWMutex

	LastConnectionID int
}

func NewSession(name string) *Session {
	logrus.Infof("Session opened: %s", name)

	s := &Session{
		Name:             name,
		Created:          time.Now(),
		Connections:      map[*Connection]interface{}{},
		Messages:         make(chan SignalingMessage, 100),
		LastConnectionID: 0,
	}

	go s.run()

	metricSessionsCreated.Inc()

	return s
}

func (s *Session) RemoveConnection(c *Connection) error {
	s.ConnectionsMutex.Lock()
	defer s.ConnectionsMutex.Unlock()

	delete(s.Connections, c)

	if len(s.Connections) == 0 {
		sessionsMutex.Lock()
		delete(sessions, s.Name)
		sessionsMutex.Unlock()

		logrus.Infof("Session closed: %s", s.Name)

		return nil
	} else {
		return s.SendControlMessages()
	}
}

func (s *Session) AddConnection(c *Connection) error {
	s.ConnectionsMutex.Lock()
	defer s.ConnectionsMutex.Unlock()

	c.ID = s.LastConnectionID
	s.LastConnectionID++

	s.Connections[c] = nil

	return s.SendControlMessages()
}

func (s *Session) SendControlMessages() error {
	conns := []pkg.Connection{}
	for c := range s.Connections {
		conns = append(conns, c.Connection)
	}

	cmsg := &pkg.SignalingMessage{
		Control: &pkg.ControlMessage{
			Connections: conns,
		},
	}

	for c := range s.Connections {
		cmsg.Control.ConnectionID = c.ID

		if err := c.Conn.WriteJSON(cmsg); err != nil {
			return err
		} else {
			logrus.Infof("Send control message: %s", cmsg)
		}
	}

	return nil
}

func (s *Session) String() string {
	return s.Name
}

func (s *Session) run() {
	for msg := range s.Messages {
		msg.CollectMetrics()

		s.ConnectionsMutex.RLock()

		for c := range s.Connections {
			if msg.Sender != c {
				c.Messages <- msg
			}
		}

		s.ConnectionsMutex.RUnlock()
	}
}

func (s *Session) Close() error {
	s.ConnectionsMutex.Lock()
	defer s.ConnectionsMutex.Unlock()

	for c := range s.Connections {
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}
