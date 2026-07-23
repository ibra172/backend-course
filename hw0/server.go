package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const shutdownTimeout time.Duration = 10 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen on port 8080: %v", err)
	}
	defer listener.Close()

	wg := sync.WaitGroup{}

	go func() {
		<-ctx.Done()

		log.Println("shutdown signal received")

		err := listener.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			log.Printf("failed to close listener: %v", err)
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				break
			}

			log.Printf("failed to accept connection: %v", err)
			continue
		}

		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			defer c.Close()

			if _, err := c.Write([]byte("OK\n")); err != nil {
				log.Printf("failed to send response: %v", err)
				return
			}
		}(conn)
	}

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	timer := time.NewTimer(shutdownTimeout)
	defer timer.Stop()

	select {
	case <-done:
		log.Println("server stopped gracefully")
	
	case <-timer.C:
		log.Printf("shutdown timeout of %s exceeded", shutdownTimeout)
	}
}
