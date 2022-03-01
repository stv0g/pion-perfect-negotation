package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

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
	sessionsMutex = sync.Mutex{}
	server        *http.Server
)

func handle(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade:", err)
		return
	}

	n := strings.TrimLeft(r.URL.Path, "/")

	sessionsMutex.Lock()
	if s, ok := sessions[n]; !ok {
		s = NewSession(n)

		sessions[n] = s

		s.NewConnection(c)
		logrus.Infof("Created new session: %s", n)
	} else {
		s.NewConnection(c)
		logrus.Infof("Used existing session: %s", n)
	}
	sessionsMutex.Unlock()
}

func handleSignals(signals chan os.Signal) {
	for range signals {
		sessionsMutex.Lock()
		for _, s := range sessions {
			if err := s.Close(); err != nil {
				logrus.Panicf("Failed to close session: %s", err)
			}
		}
		sessionsMutex.Unlock()

		if err := server.Shutdown(context.Background()); err != nil {
			logrus.Panicf("Failed to shutdown HTTP server: %s", err)
		}
	}
}

func main() {
	flag.Parse()

	signals := make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal is received
	go handleSignals(signals)

	server = &http.Server{
		Addr: *addr,
	}

	http.HandleFunc("/", handle)

	logrus.Infof("Listening on: %s", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Errorf("Failed to listen and serve: %s", err)
	}
}
