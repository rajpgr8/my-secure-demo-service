package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Simple HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Distroless!")
	})

	// Health Checks
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	http.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		// In a real app, check DB connection here
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	})

	srv := &http.Server{Addr: ":8080"}

	// Start Server
	go func() {
		fmt.Println("Starting server on :8080")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	// Simulate cleanup
	time.Sleep(2 * time.Second)
	fmt.Println("Server exited.")
}
