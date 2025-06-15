package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gobin/internal/game"

	"github.com/gorilla/websocket"
)

func TestNewHandler(t *testing.T) {
	g := game.New()
	h := New(g)
	if h.game != g {
		t.Errorf("New() game = %v, want %v", h.game, g)
	}
}

func TestHandleWebSocket(t *testing.T) {
	// Create a test game
	g := game.New()
	h := New(g)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(h.HandleWebSocket))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Test connection
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("Failed to connect to WebSocket server: %v", err)
	}

	// Verify that a client was added
	clients := g.GetClients()
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}
}
