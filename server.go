package main

import (
	"fmt"
	"io"
	"net"
	"sync"
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
func (server *Server) Broadcast(user *User, msg string) {
	server.Ch <- fmt.Sprintf("[%s]%s:%s", user.Addr, user.Name, msg)
}

func (server *Server) SendToUser(sender *User, userCode string, msg string) {
	for code, user := range server.Users {
		if userCode == code {
			for _, ch := range user.Addrs {
				ch <- fmt.Sprintf("[%s]%s:%s", user.Addr, sender.Name, msg)
			}
			break
		}
	}
}

func (server *Server) Handler(conn net.Conn) {
	fmt.Println("连接建立成功")
	user := NewUser(conn, server)
	user.Login()
	go func() {
		buf := make([]byte, 4096)
		for {
			l, err := conn.Read(buf)
			if l == 0 {
				user.Logout()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn read err:", err)
				return
			}
			msg := string(buf)
			fmt.Println(user.Addr, msg)
			switch msg {
			case "login":
				// user := NewUser(conn, server)
				// user.Logout()
				user.Login()
			case "logout":
				user.Logout()
			default:
				user.SendMsg(msg)
			}
		}
	}()
	// 当前handler阻塞
	// select {}
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
