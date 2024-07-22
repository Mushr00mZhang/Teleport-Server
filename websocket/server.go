package websocket

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Server struct {
	Addr    string             // 地址
	Port    int                // 端口
	Clients map[string]*Client // [登录名]:客户端
	MapLock sync.RWMutex
	Chan    chan string
}

func NewServer(addr string, port int) *Server {
	return &Server{
		Addr:    addr,
		Port:    port,
		Clients: map[string]*Client{},
	}
}

func (server *Server) Listen() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		Login(w, r, server)
	})
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", server.Addr, server.Port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
