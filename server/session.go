package main

import (
	"sync"

	"github.com/VILLASframework/VILLASnode/tools/ws-relay/common"
	"github.com/sirupsen/logrus"
)

type Session struct {
	Name string

	Messages chan common.SignalingMessage

	Connections      map[*Connection]interface{}
	ConnectionsMutex sync.RWMutex
}

func NewSession(name string) *Session {
	logrus.Infof("New session: %s", name)
	s := &Session{
		Name:        name,
		Connections: map[*Connection]interface{}{},
		Messages:    make(chan common.SignalingMessage, 100),
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

		for c := range s.Connections {
			if msg.Sender != c {
				c.Messages <- msg
			}
		}

		s.ConnectionsMutex.RUnlock()
	}
}

func (s *Session) HasImpoliteConnection() bool {
	hasImpolite := false
	for c := range s.Connections {
		if !c.isPolite {
			hasImpolite = true
		}
	}
	return hasImpolite
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
