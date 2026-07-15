package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	resp, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		log.Fatalf("failed to read from the connection: %v", err)
	}

	if resp == "OK\n" {
		log.Printf("a response was received from the server: %q", resp)
	} else {
		log.Fatalf("expected %q, received %q", "OK\n", resp)
	}
}
