package main

import (
	"encoding/json"
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
	Addr   string                 `json:"-"`
	Ch     chan string            `json:"-"`
	Addrs  map[string]chan string `json:"-"`
	Conn   net.Conn               `json:"-"`
	Server *Server                `json:"-"`
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
	user.Server.Users[user.Code.String()] = user
	// 启动监听
	go user.Listen()
	return user
}
func (user *User) SendJSON(obj map[string]interface{}) {
	buf, _ := json.Marshal(obj)
	user.Conn.Write(buf)
}
func (user *User) SetCode() {
	obj := map[string]interface{}{
		"Type": "SetCode",
		"Msg":  user.Code,
	}
	user.SendJSON(obj)
}
func (user *User) SetUsers() {
	users := make([]map[string]interface{}, 0)
	for _, u := range user.Server.Users {
		if u == nil {
			continue
		}
		users = append(users, map[string]interface{}{
			"Name": u.Name,
			"Code": u.Code,
		})
	}
	obj := map[string]interface{}{
		"Type": "SetUsers",
		"Msg":  users,
	}
	user.SendJSON(obj)
}
func (user *User) Listen() {
	for {
		msg := <-user.Ch
		user.Conn.Write([]byte(fmt.Sprintln(msg)))
	}
}

func (user *User) Login() {
	// user.Server.MapLock.Lock()
	// user.Server.Users[user.Code.String()].Name = user
	// user.Server.MapLock.Unlock()
	// user.Server.SendToAll(user, "login")
}
func (user *User) Logout() {
	user.Server.MapLock.Lock()
	delete(user.Server.Users, user.Addr)
	user.Server.MapLock.Unlock()
	// user.Server.SendToAll(user, "logout")
}
func (user *User) SendToUser(to string, msg string) {
	// user.Server.SendToUser(user, uuid.MustParse(to), msg)
	send, ok := user.Server.Users[to]
	if !ok {
		fmt.Printf("User not found: %s", to)
	}
	params := map[string]interface{}{
		"Type": "Message",
		"Msg":  msg,
		"From": user.Code,
	}
	buf, _ := json.Marshal(params)
	send.Ch <- string(buf)
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
