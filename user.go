package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
)

type Client struct {
	Addr string
	Ch   chan string
}
type User struct {
	Name   string
	Code   uuid.UUID
	Addr   string
	Ch     chan string
	Addrs  map[string]chan string
	Conn   net.Conn
	Server *Server
}

func NewUser(conn net.Conn, server *Server) *User {
	user := &User{
		Name:   conn.RemoteAddr().String(),
		Code:   uuid.New(),
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
	user.Server.SendToAll(user, "login")
}
func (user *User) Logout() {
	user.Server.MapLock.Lock()
	delete(user.Server.Users, user.Addr)
	user.Server.MapLock.Unlock()
	user.Server.SendToAll(user, "logout")
}
func (user *User) SendMsg(msg string) {
	strs := strings.Split(msg, "|")
	l := len(strs)
	if l == 0 {
		return
	}
	switch strs[0] {
	case "rename":
		if l < 2 {
			return
		}
		user.Name = strs[1]
	case "to":
		if l < 3 {
			return
		}
		code := strs[1]
		msg := strs[2]
		if code == "all" {
			user.Server.SendToAll(user, msg)
		} else {
			user.Server.SendToUser(user, uuid.MustParse(code), msg)
		}
		// case "get":
		// 	if l < 3 {
		// 		return
		// 	}
		// 	addr := strs[1]
		// 	msg := strs[2]
		// 	if addr == "broadcast" {
		// 		user.Server.SendToAll(user, msg)
		// 	} else {
		// 		user.Server.SendToUser(user, addr, msg)
		// 	}
	}
}
