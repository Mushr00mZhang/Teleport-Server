package websocket

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024 * 2
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	Id        *uuid.UUID                 `json:"-"`          // Id
	LoginName string                     ``                  // 登录名
	Password  string                     `json:",omitempty"` // 密码
	NickName  string                     ``                  // 昵称
	Conns     map[string]*websocket.Conn `json:"-"`          // [监听地址]:连接池
	Chans     map[string]chan []byte     `json:"-"`          // [监听地址]:通道
	Status    int                        ``                  // 状态
}
type Message struct {
	Type    string
	Content interface{}
	From    string
	To      string
	Time    time.Time
}
type FileContent struct {
	Id    string
	Buf   []int8
	Index int
	Count int
	Size  int
	MD5   string
}
type FileMsg struct {
	Type    string
	Content FileContent
	From    string
	To      string
	Time    time.Time
}

func (client *Client) Read(remote string, server *Server) {
	conn, ch := client.Conns[remote], client.Chans[remote]
	defer func() {
		conn.Close()
		close(ch)
		delete(client.Conns, remote)
		delete(client.Chans, remote)
		if len(client.Chans) == 0 {
			client.Status = 0
			server.Logoff(client)
		}
	}()
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		var msg Message
		_, buf, err := conn.ReadMessage()
		if err != nil {
			// if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			// }
			log.Printf("error: %v", err)
			break
		}
		if string(buf[0:10]) == "file-chunk" {
			id := string(buf[10:46])
			index, err := strconv.ParseInt(strings.TrimSpace(string(buf[46:54])), 10, 32)
			if err != nil {
				continue
			}
			count, err := strconv.ParseInt(strings.TrimSpace(string(buf[54:62])), 10, 32)
			if err != nil {
				continue
			}
			md5 := string(buf[62:94])
			to := strings.TrimSpace(string(buf[94:128]))
			fmt.Printf("File chunk to: %s id: %s[%s/%d] md5: %s len: %d\n", to, id, strings.Repeat("0", len(strconv.FormatInt(count, 10))-len(strconv.FormatInt(index+1, 10)))+strconv.FormatInt(index+1, 10), count, md5, len(buf)-128)
			if to != client.LoginName {
				if to, ok := server.Clients[to]; ok {
					for _, ch := range to.Chans {
						ch <- buf
					}
				}
			} else {
				for addr, ch := range client.Chans {
					if addr != remote {
						ch <- buf
					}
				}
			}
			continue
		}
		err = json.Unmarshal(buf, &msg)
		if err != nil {
			log.Printf("Unmarshal error: %v", err)
		}
		switch msg.Type {
		case "rename":
			nickName, ok := msg.Content.(string)
			if ok {
				client.NickName = nickName
			}
		case "getUsers":
			users := make([]*Client, len(server.Clients))
			i := 0
			for _, cl := range server.Clients {
				users[i] = cl
				i++
			}
			msg := Message{
				Type:    "users",
				Content: users,
				From:    "server",
				To:      client.LoginName,
			}
			buf, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Marshal error: %v", err)
			}
			ch <- buf
		case "text":
			msg.Time = time.Now()
			buf, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Marshal error: %v", err)
			}
			if msg.To != client.LoginName {
				if to, ok := server.Clients[msg.To]; ok {
					for _, ch := range to.Chans {
						ch <- buf
					}
				}
			} else {
				for addr, ch := range client.Chans {
					if addr != remote {
						ch <- buf
					}
				}
			}
		case "file":
			msg.Time = time.Now()
			buf, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Marshal error: %v", err)
			}
			if msg.To != client.LoginName {
				if to, ok := server.Clients[msg.To]; ok {
					for _, ch := range to.Chans {
						ch <- buf
					}
				}
			} else {
				for addr, ch := range client.Chans {
					if addr != remote {
						ch <- buf
					}
				}
			}
			// case "file-chunk":
			// 	msg.Time = time.Now()
			// 	buf, err := json.Marshal(msg)
			// 	if err != nil {
			// 		log.Printf("Marshal error: %v", err)
			// 	}
			// 	if msg.To != client.LoginName {
			// 		if to, ok := server.Clients[msg.To]; ok {
			// 			for _, ch := range to.Chans {
			// 				ch <- buf
			// 			}
			// 		}
			// 	} else {
			// 		for addr, ch := range client.Chans {
			// 			if addr != remote {
			// 				ch <- buf
			// 			}
			// 		}
			// 	}
		}
	}
}
func (client *Client) Write(remote string, server *Server) {
	conn, ch := client.Conns[remote], client.Chans[remote]
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case buf, ok := <-ch:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			var w io.WriteCloser
			var err error
			if string(buf[0:10]) == "file-chunk" {
				w, err = conn.NextWriter(websocket.BinaryMessage)
			} else {
				w, err = conn.NextWriter(websocket.TextMessage)
			}
			if err != nil {
				return
			}
			w.Write(buf)

			// Add queued chat messages to the current websocket message.
			// n := len(ch)
			// for i := 0; i < n; i++ {
			// 	w.Write(newline)
			// 	w.Write(<-ch)
			// }

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func NewClient(loginName string) *Client {
	return &Client{
		LoginName: loginName,
		NickName:  loginName,
		Conns:     map[string]*websocket.Conn{},
		Chans:     map[string]chan []byte{},
	}
}

func Login(w http.ResponseWriter, r *http.Request, server *Server) {
	query := r.URL.Query()
	loginName := query.Get("LoginName")
	nickName := query.Get("NickName")
	// buf, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	w.WriteHeader(500)
	// 	return
	// }
	var client *Client
	// err = json.Unmarshal(buf, &client)
	// if err != nil {
	// 	w.WriteHeader(500)
	// 	return
	// }
	if c, ok := server.Clients[loginName]; ok {
		client = c
	} else {
		client = NewClient(loginName)
		if nickName != "" {
			client.NickName = nickName
		}
		// client.Conns = map[string]*websocket.Conn{}
		// client.Chans = map[string]chan []byte{}
		server.Clients[client.LoginName] = client
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	client.Conns[r.RemoteAddr] = conn
	client.Chans[r.RemoteAddr] = make(chan []byte, 256)
	client.Status = 1
	go client.Read(r.RemoteAddr, server)
	go client.Write(r.RemoteAddr, server)
	if len(client.Chans) == 1 {
		server.Login(client)
	}
}
