package websocket

import (
	"encoding/json"
	"log"
	"net/http"
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
	maxMessageSize = 512
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
	Id        *uuid.UUID                 `json:"-"` // Id
	LoginName string                     ``         // 登录名
	Password  string                     ``         // 密码
	NickName  string                     ``         // 昵称
	Conns     map[string]*websocket.Conn `json:"-"` // [监听地址]:连接池
	Chans     map[string]chan []byte     `json:"-"` // [监听地址]:通道
}
type Message struct {
	Type    string
	Content string
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
	}()
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		var msg Message
		_, buf, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		err = json.Unmarshal(buf, &msg)
		if err != nil {
			log.Printf("Parse error: %v", err)
		}
		msg.Time = time.Now()
		switch msg.Type {
		case "rename":
			client.NickName = msg.Content
		case "text":
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

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(buf)

			// Add queued chat messages to the current websocket message.
			n := len(ch)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-ch)
			}

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
	go client.Read(r.RemoteAddr, server)
	go client.Write(r.RemoteAddr, server)
}
