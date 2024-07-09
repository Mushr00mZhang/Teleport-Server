package main

import (
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
	// user.Login()
	go func() {
		buf := make([]byte, 4096)
		for {
			l, err := conn.Read(buf)
			if err != nil && err != io.EOF {
				fmt.Println("Conn read err:", err)
				return
			}
			msg := ""
			if l > 0 {
				msg = string(buf[:l-1])
			} else {
				user.Logout()
				return
			}
			fmt.Println(user.Addr, msg)
			switch msg {
			case "":
				fmt.Printf("User address %s connected\n", user.Addr)
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
