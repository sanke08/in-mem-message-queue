package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sanke08/in-mem-message-queue/internal/auth"
	"github.com/sanke08/in-mem-message-queue/internal/broker"
	"github.com/sanke08/in-mem-message-queue/internal/server"
)

func main() {
	ctx := context.Background()

	am := auth.NewAuthManager()
	b := broker.NewBroker(5*time.Second, 3)
	ab := broker.NewAuthenticatedBroker(b, am)
	server := server.NewServer(ctx, ab)

	mux := http.NewServeMux()

	mux.HandleFunc("/create_key", server.HandleCreateKey)
	mux.HandleFunc("/publish", server.HandlePublish)
	mux.HandleFunc("/claim", server.HandleClaim)
	mux.HandleFunc("/ack", server.HandleAck)
	mux.HandleFunc("/stats", server.HandleStats)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	serverErrCh := make(chan error, 1)

	go func() {
		log.Printf("Server is running on %s", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// wait for signal or server error
	select {
	case sig := <-sigCh:
		log.Printf("[main] received signal %v, shutting down...", sig)
	case err := <-serverErrCh:
		// if err != nil {
		log.Printf("[main] http server error: %v", err)
		// }
	}

	// Graceful shutdown
	log.Println("[main] shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	err := httpServer.Shutdown(shutdownCtx)
	if err != nil {
		log.Printf("[main] server Shutdown error: %v", err)
	}

	log.Println("[main] server exited gracefully")
}
