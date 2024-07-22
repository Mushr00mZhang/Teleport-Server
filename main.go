package main

import (
	"server/websocket"
)

func main() {
	// server := NewServer("", 8888)
	// go server.Start()
	ws := websocket.NewServer("", 8889)
	ws.Listen()
}
