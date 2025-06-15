package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"time"

	"gobin/internal/game"
	"gobin/internal/websocket"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "Port to listen on")
	wordSetPath := flag.String("wordset", "static/sets/corporate.json", "Path to word set JSON file")
	flag.Parse()

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Create game instance
	g := game.New()
	if err := g.LoadWordSet(*wordSetPath); err != nil {
		log.Fatalf("Failed to load word set: %v", err)
	}

	// Create WebSocket handler
	wsHandler := websocket.New(g)

	// Set up routes
	http.HandleFunc("/ws", wsHandler.HandleWebSocket)
	http.Handle("/", http.FileServer(http.Dir("static")))

	// Start server
	addr := ":" + *port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
