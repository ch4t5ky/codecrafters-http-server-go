package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

func main() {
	directory := flag.String("directory", "", "")
	flag.Parse()

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

		go HandleConnection(conn, *directory)
	}
}

func HandleConnection(conn net.Conn, dir string) {
	defer conn.Close()
	buf := make([]byte, 1024)
	bytesRead, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("Error reading: %#v\n", err)
		return
	}

	request := parseRequest(string(buf[:bytesRead]))
	fmt.Print(request)
	switch path := request.Path; {
	case path == "/":
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	case path == "/user-agent":
		userAgent := request.Headers["user-agent"]
		conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)))
	case strings.HasPrefix(path, "/echo/"):
		msg := strings.Split(path, "/")[2]
		compressionScheme, _ := request.Headers["accept-encoding"]
		switch compressionScheme {
		case "gzip":
			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			_, _ = gz.Write([]byte(msg))
			_ = gz.Close()
			msg = string(b.Bytes())
			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nContent-Encoding: gzip\r\n\r\n%s", len(msg), msg)))
		default:
			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(msg), msg)))
		}
		conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(msg), msg)))
	case strings.HasPrefix(path, "/files/"):
		switch request.Method {
		case "GET":
			fileName := strings.Split(path, "/")[2]
			file := fmt.Sprintf("%s/%s", dir, fileName)
			if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			}
			data, _ := os.ReadFile(file)
			msg := string(data)
			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(msg), msg)))
		case "POST":
			fileName := strings.Split(path, "/")[2]
			name := fmt.Sprintf("%s/%s", dir, fileName)
			file, _ := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0644)
			defer file.Close()
			_, err = file.Write([]byte(request.Body))
			conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
		}
	default:
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

func parseRequest(data string) Request {
	lines := strings.Split(data, "\r\n")

	requestLine := strings.Split(lines[0], " ")
	method := requestLine[0]
	path := requestLine[1]

	headers := make(map[string]string)
	headerIndex := 1
	header := lines[1]

	for header != "" {
		headerSplit := strings.Split(header, ":")
		name, value := headerSplit[0], headerSplit[1]
		name = strings.ToLower(name)
		value = strings.ReplaceAll(value, " ", "")
		headers[name] = value
		headerIndex++
		header = lines[headerIndex]
	}

	body := lines[headerIndex+1]
	return Request{
		Method:  method,
		Path:    path,
		Headers: headers,
		Body:    body,
	}
}
