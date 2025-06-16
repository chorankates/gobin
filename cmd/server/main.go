package main

import (
	"flag"
	"math/rand"
	"net/http"
	"time"

	"gobin/internal/game"
	"gobin/internal/websocket"

	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	// Parse command line flags
	port := flag.String("port", "8080", "Port to listen on")
	wordSetPath := flag.String("wordset", "static/sets/corporate.json", "Path to word set JSON file")
	flag.Parse()

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Create game instance
	g := game.New()
	if err := g.LoadWordSet(*wordSetPath); err != nil {
		sugar.Fatalw("Failed to load word set",
			"error", err,
			"path", *wordSetPath,
		)
	}

	// Create WebSocket handler
	wsHandler := websocket.New(g)

	// Set up routes
	http.HandleFunc("/ws", wsHandler.HandleWebSocket)
	http.Handle("/", http.FileServer(http.Dir("static")))

	// Start server
	addr := ":" + *port
	sugar.Infow("Starting server",
		"address", addr,
		"wordSetPath", *wordSetPath,
	)
	if err := http.ListenAndServe(addr, nil); err != nil {
		sugar.Fatalw("Failed to start server",
			"error", err,
			"address", addr,
		)
	}
}
