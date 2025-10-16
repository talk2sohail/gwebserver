package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("method=%s uri=%s remote_addr=%s user_agent=%s", r.Method, r.RequestURI, r.RemoteAddr, r.UserAgent())
		next(w, r)
	}
}

type Joke struct {
	Setup     string `json:"setup"`
	Punchline string `json:"punchline"`
}

func jokeHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("https://official-joke-api.appspot.com/random_joke")
	if err != nil {
		http.Error(w, "Failed to get a joke", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var joke Joke
	if err := json.NewDecoder(resp.Body).Decode(&joke); err != nil {
		http.Error(w, "Failed to decode the joke", http.StatusInternalServerError)
		return
	}

	_, err = fmt.Fprintf(w, "%s\n%s", joke.Setup, joke.Punchline)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func main() {
	// Configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Handlers
	http.HandleFunc("/", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hello, World!")
		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}))

	http.HandleFunc("/health", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, "OK")
		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}))

	http.HandleFunc("/joke", loggingMiddleware(jokeHandler))

	// Server setup
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", port, err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	log.Println("Server gracefully stopped")
}
