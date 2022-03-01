package main

import (
	"flag"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

var signalingURL = flag.String("url", "ws://localhost:8080/session", "Signalling URL")

func main() {
	flag.Parse()

	u, _ := url.Parse(*signalingURL)

	pc, err := NewPeerConnection(u)
	if err != nil {
		logrus.Panicf("%s", err)
	}
	defer pc.Close()

	signals := make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal is received
	<-signals
}
