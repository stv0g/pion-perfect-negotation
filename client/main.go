package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

var sessionName = flag.String("name", "session", "Name of session")
var signalingURL = flag.String("url", "ws://localhost:8080", "Signaling URL")

func main() {
	flag.Parse()

	u, _ := url.Parse(*signalingURL)
	u.Path = fmt.Sprintf("/%s", *sessionName)

	pc, err := NewPeerConnection(u)
	if err != nil {
		logrus.Errorf("%s", err)
		os.Exit(-1)
	}

	signals := make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal is received
	<-signals

	if err := pc.Close(); err != nil {
		logrus.Errorf("%s", err)
		os.Exit(-1)
	}
}
