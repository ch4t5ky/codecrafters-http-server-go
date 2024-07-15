package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running test
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go HandleConnection(conn)
	}
}

func HandleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("Error reading: %#v\n", err)
		return
	}
	packet := strings.Fields(string(buf))

	switch path := packet[1]; {
	case path == "/":
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	case path == "/user-agent":
		msg := ""
		packet = strings.Split(string(buf), "\r\n")
		for i := 0; i < len(packet); i++ {
			fmt.Println(packet[i])
			dict := strings.Split(packet[i], ":")
			if len(dict) != 2 {
				continue
			}
			header, value := dict[0], dict[1]
			header = strings.ToLower(header)
			if header == "user-agent" {
				msg = strings.ReplaceAll(value, " ", "")
				break
			}
		}
		conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(msg), msg)))
	case strings.HasPrefix(path, "/echo/"):
		msg := strings.Split(path, "/")[2]
		conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(msg), msg)))
	case strings.HasPrefix(path, "/files/"):
		fileName := strings.Split(path, "/")[2]
		data, _ := os.ReadFile(fileName)
		msg := string(data)
		conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(msg), msg)))
	default:
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}
