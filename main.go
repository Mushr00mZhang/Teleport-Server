package main

import (
	"os"
	"server/websocket"
	"strings"
	"time"
	_ "time/tzdata"
)

func main() {
	// 设置timezone，默认Asia/Shanghai
	tz := os.Getenv("TZ")
	if strings.TrimSpace(tz) == "" {
		tz = "Asia/Shanghai"
	}
	time.LoadLocation(tz)
	ws := websocket.NewServer("", 8889)
	ws.Listen()
}
