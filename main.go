package main

import (
	"fmt"
	"io"
	"net"
)

func main() {
	listener, err := net.Listen("tcp4", "127.0.0.1:8888")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("connected")
		go func(conn *net.Conn) {
			buffer, err := io.ReadAll(*conn)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(string(buffer))
		}(&conn)
	}
}
