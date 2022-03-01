# An example demonstrating perfect negotation with `pion/webrtc`

This example implements a WebSocket signaling server in `./server` and a WebRTC peer based-on `pion/webrtc` in `./client`.

## Note

This is example is currently not working as pion/webrtc does not support rollbacks.

See: https://github.com/pion/webrtc/issues/2133

## Usage

1. Start signaling server `go run ./server/`
2. Start first peer `go run ./client/`
3. Start second peer `go run ./client/`

## References


- https://w3c.github.io/webrtc-pc/#perfect-negotiation-example
- https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Perfect_negotiation
