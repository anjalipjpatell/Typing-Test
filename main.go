package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const port = "3333"

func main() {
	// Serve static files from frontend/dist directory
	fs := http.FileServer(http.Dir("./frontend/dist"))
	http.Handle("/", fs)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
	}

	// Start the server in a goroutine so it doesn't block the main thread
	go func() {
		fmt.Printf("Server starting on http://localhost:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Create a channel to listen for OS signals (like Ctrl+C), and block until a signal is recieved
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	fmt.Println("\nShutting down server...")

	// Give the server 3 seconds to finish any ongoing requests
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exited gracefully")
}
