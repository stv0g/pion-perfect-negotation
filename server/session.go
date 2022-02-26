package main

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Session struct {
	Name string

	Messages chan Message

	Connections      []*Connection
	ConnectionsMutex sync.RWMutex
}

func NewSession(name string) *Session {
	logrus.Infof("New session: %s", name)
	s := &Session{
		Name:        name,
		Connections: []*Connection{},
		Messages:    make(chan Message, 100),
	}

	go s.run()

	return s
}

func (s *Session) String() string {
	return s.Name
}

func (s *Session) run() {
	for msg := range s.Messages {
		s.ConnectionsMutex.RLock()

		for _, c := range s.Connections {

			if msg.Sender != c {
				c.Messages <- msg
			}
		}

		s.ConnectionsMutex.RUnlock()
	}
}

func (s *Session) NewConnection(c *websocket.Conn) *Connection {
	logrus.Infof("New connection from %s for session %s", c.RemoteAddr(), s)

	d := &Connection{
		Conn:     c,
		Session:  s,
		Messages: make(chan Message, 100),
	}

	s.ConnectionsMutex.Lock()
	s.Connections = append(s.Connections, d)
	s.ConnectionsMutex.Unlock()

	d.run()

	return d
}
