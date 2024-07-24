package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
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
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		Login(w, r, server)
	})
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", server.Addr, server.Port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
func (server *Server) Login(client *Client) {
	msg := Message{
		Type:    "login",
		Content: client.NickName,
		From:    client.LoginName,
		To:      "all",
		Time:    time.Now(),
	}
	buf, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Marshal error:", err)
	}
	for _, cl := range server.Clients {
		if cl == client {
			continue
		}
		for _, ch := range cl.Chans {
			ch <- buf
		}
	}
}
func (server *Server) Logoff(client *Client) {
	msg := Message{
		Type:    "logoff",
		Content: client.NickName,
		From:    client.LoginName,
		To:      "all",
		Time:    time.Now(),
	}
	buf, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Marshal error:", err)
	}
	for _, cl := range server.Clients {
		if cl == client {
			continue
		}
		for _, ch := range cl.Chans {
			ch <- buf
		}
	}

}
