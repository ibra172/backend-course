package main

import (
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen on port 8080: %v", err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("error on accept happened: %v\n", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()

			if _, err := c.Write([]byte("OK\n")); err != nil {
				log.Printf("failed to send response: %v", err)
				return
			}
		}(conn)
	}
}
