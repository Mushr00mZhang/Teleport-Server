package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/google/uuid"
)

type Server struct {
	Addr    string
	Port    int
	Users   map[string]*User
	MapLock sync.RWMutex
	Ch      chan string
}

func NewServer(addr string, port int) *Server {
	return &Server{
		Addr:  addr,
		Port:  port,
		Users: make(map[string]*User),
		Ch:    make(chan string),
	}
}

// 监听server channel
func (server *Server) ListenCh() {
	for {
		msg := <-server.Ch
		server.MapLock.Lock()
		for _, user := range server.Users {
			user.Ch <- msg
		}
		server.MapLock.Unlock()
	}
}
func (server *Server) SendToAll(user *User, msg string) {
	server.Ch <- fmt.Sprintf("[%s|all]msg:%s", user.Code, msg)
}
func (server *Server) SendToUser(sender *User, code uuid.UUID, msg string) {
	for _, user := range server.Users {
		if code == user.Code {
			user.Ch <- fmt.Sprintf("[%s|%s]msg:%s", sender.Code, user.Code, msg)
			for _, ch := range user.Addrs {
				ch <- fmt.Sprintf("[%s|%s]msg:%s", sender.Code, user.Code, msg)
			}
			break
		}
	}
}

func (server *Server) Handler(conn net.Conn) {
	user := NewUser(conn, server)
	fmt.Printf("User address %s connected\n", user.Addr)
	// user.SetUsers()
	// user.SetCode()
	// user.Login()
	go func() {
		buf := make([]byte, 4096)
		for {
			fmt.Println("Receive msg")
			l, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {

				} else if err, ok := err.(*net.OpError); ok && err.Op == "read" {
					server.MapLock.Lock()
					delete(server.Users, user.Code.String())
					server.MapLock.Unlock()
				}
				fmt.Println("Conn read err:", err)
				return
			}
			fmt.Printf("Message received: %s size: %dB\n", string(buf[:l]), l)
			var res map[string]interface{}
			err = json.Unmarshal(buf[:l], &res)
			if err != nil {
				fmt.Println("Unmarshal res err:", err)
				return
			}
			switch res["Type"] {
			// case "":
			// 	fmt.Printf("User address %s connected\n", user.Addr)
			case "Login":
				user.Name = res["Name"].(string)
				for _, user := range user.Server.Users {
					user.SetUsers()
				}
				// user := NewUser(conn, server)
				// user.Logout()
				// user.Login()
			case "Logout":
				// user.Logout()
			case "Send":
				user.SendToUser(res["To"].(string), res["Msg"].(string))
			}
		}
	}()
}
func (server *Server) Start() {

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", server.Addr, server.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	defer listener.Close()
	go server.ListenCh()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			continue
		}
		go server.Handler(conn)
	}
}
