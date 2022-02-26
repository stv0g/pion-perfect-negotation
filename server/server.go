package main

import (
	"flag"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Message struct {
	Sender *Connection
	Data   []byte
	Type   int
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096
)

var (
	addr     = flag.String("addr", "localhost:8080", "http service address")
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	} // use default options
	sessions      = map[string]*Session{}
	sessionsMutex = sync.RWMutex{}
)

func handle(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade:", err)
		return
	}

	n := strings.TrimLeft(r.URL.Path, "/")

	sessionsMutex.RLock()
	s, ok := sessions[n]
	sessionsMutex.RUnlock()

	if !ok {
		s = NewSession(n)

		sessionsMutex.Lock()
		sessions[n] = s
		sessionsMutex.Unlock()

		logrus.Infof("Created new session: %s", n)
	} else {
		logrus.Infof("Use existing session: %s", n)
	}

	s.NewConnection(c)
}

func main() {
	flag.Parse()

	http.HandleFunc("/", handle)

	logrus.Infof("Listening on: %s", *addr)

	err := http.ListenAndServe(*addr, nil)
	logrus.Fatal(err)
}
