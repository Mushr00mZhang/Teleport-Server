package main

import (
	"fmt"
	"net"
)

type Client struct {
	Addr string
	Ch   chan string
}
type User struct {
	Name   string
	Code   string
	Addr   string
	Ch     chan string
	Addrs  map[string]chan string
	Conn   net.Conn
	Server *Server
}

func NewUser(conn net.Conn, server *Server) *User {
	user := &User{
		Name:   conn.RemoteAddr().String(),
		Code:   "",
		Addr:   conn.RemoteAddr().String(),
		Ch:     make(chan string),
		Conn:   conn,
		Server: server,
	}
	// 启动监听
	go user.Listen()
	return user
}
func (user *User) Listen() {
	for {
		msg := <-user.Ch
		user.Conn.Write([]byte(fmt.Sprintln(msg)))
	}
}

func (user *User) Login() {
	user.Server.MapLock.Lock()
	user.Server.Users[user.Addr] = user
	user.Server.MapLock.Unlock()
	user.Server.Broadcast(user, "已上线")
}
func (user *User) Logout() {
	user.Server.MapLock.Lock()
	delete(user.Server.Users, user.Addr)
	user.Server.MapLock.Unlock()
	user.Server.Broadcast(user, "下线")
}
func (user *User) SendMsg(msg string) {
	user.Server.Broadcast(user, msg)
}
