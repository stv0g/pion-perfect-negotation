package main

import (
	"flag"
	"net/url"
)

var signalingURL = flag.String("url", "ws://localhost:8080/session", "Signalling URL")
var polite = flag.Bool("polite", false, "Is this peer polite?")

func main() {
	flag.Parse()

	u, _ := url.Parse(*signalingURL)

	NewPeerConnection(u, *polite)

	// Block forever
	select {}
}
