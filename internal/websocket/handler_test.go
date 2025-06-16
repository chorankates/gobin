package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func setupTestServer(t *testing.T) (*game.Game, *Handler, *httptest.Server) {
	g := game.New()
	h := New(g)
	server := httptest.NewServer(http.HandlerFunc(h.HandleWebSocket))
	return g, h, server
}

func connectClient(t *testing.T, server *httptest.Server) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	return conn
}

func TestHandleWebSocket(t *testing.T) {
	_, _, server := setupTestServer(t)
	defer server.Close()

	// Test connection
	conn := connectClient(t, server)
	defer conn.Close()

	// Verify that a client was added
	time.Sleep(100 * time.Millisecond) // Give time for goroutines to start
}

func TestSetName(t *testing.T) {
	g, _, server := setupTestServer(t)
	defer server.Close()

	conn := connectClient(t, server)
	defer conn.Close()

	// Send set name message
	setNameMsg := Message{
		Type: TypeSetName,
		Name: "Test Player",
	}
	msgBytes, _ := json.Marshal(setNameMsg)
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		t.Fatalf("Failed to send set name message: %v", err)
	}

	// Wait for response
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var responseMsg Message
	if err := json.Unmarshal(response, &responseMsg); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response
	if responseMsg.Type != TypeInitBoard {
		t.Errorf("Expected response type %s, got %s", TypeInitBoard, responseMsg.Type)
	}
	if responseMsg.Name != "Test Player" {
		t.Errorf("Expected name %s, got %s", "Test Player", responseMsg.Name)
	}

	// Verify client was added with correct name
	clients := g.GetClients()
	if count, exists := clients["Test Player"]; !exists || count != 0 {
		t.Errorf("Expected client 'Test Player' with 0 marked tiles, got %v", clients)
	}
}

func TestMarkTile(t *testing.T) {
	g, _, server := setupTestServer(t)
	defer server.Close()

	conn := connectClient(t, server)
	defer conn.Close()

	// First set the name
	setNameMsg := Message{
		Type: TypeSetName,
		Name: "Test Player",
	}
	msgBytes, _ := json.Marshal(setNameMsg)
	conn.WriteMessage(websocket.TextMessage, msgBytes)
	time.Sleep(100 * time.Millisecond) // Wait for name to be set

	// Discard the initial init_board message
	conn.ReadMessage()

	// Send mark tile message
	markTileMsg := Message{
		Type:     TypeMarkTile,
		Index:    0,
		Sequence: 1,
	}
	msgBytes, _ = json.Marshal(markTileMsg)
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		t.Fatalf("Failed to send mark tile message: %v", err)
	}

	// Wait for tile_marked response, skipping unrelated messages
	var responseMsg Message
	for {
		_, response, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		if err := json.Unmarshal(response, &responseMsg); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if responseMsg.Type == TypeTileMarked {
			break
		}
	}

	// Verify response
	if responseMsg.Type != TypeTileMarked {
		t.Errorf("Expected response type %s, got %s", TypeTileMarked, responseMsg.Type)
	}
	if responseMsg.Index != 0 {
		t.Errorf("Expected index 0, got %d", responseMsg.Index)
	}
	if len(responseMsg.Board) == 0 || !responseMsg.Board[0] {
		t.Error("Expected tile to be marked")
	}

	// Verify client list was updated
	clients := g.GetClients()
	if count, exists := clients["Test Player"]; !exists || count != 1 {
		t.Errorf("Expected client 'Test Player' with 1 marked tile, got %v", clients)
	}
}

func TestWinCondition(t *testing.T) {
	_, _, server := setupTestServer(t)
	defer server.Close()

	conn := connectClient(t, server)
	defer conn.Close()

	// First set the name
	setNameMsg := Message{
		Type: TypeSetName,
		Name: "Test Player",
	}
	msgBytes, _ := json.Marshal(setNameMsg)
	conn.WriteMessage(websocket.TextMessage, msgBytes)
	time.Sleep(100 * time.Millisecond) // Wait for name to be set

	// Discard the initial init_board message
	conn.ReadMessage()

	// Mark tiles to create a winning row
	var responseMsg Message
	for i := 0; i < 4; i++ {
		markTileMsg := Message{
			Type:     TypeMarkTile,
			Index:    i,
			Sequence: i + 1,
		}
		msgBytes, _ = json.Marshal(markTileMsg)
		conn.WriteMessage(websocket.TextMessage, msgBytes)
		time.Sleep(50 * time.Millisecond) // Wait for each mark to be processed
	}

	// Wait for tile_marked response with win=true, skipping unrelated messages
	for {
		_, response, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		if err := json.Unmarshal(response, &responseMsg); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if responseMsg.Type == TypeTileMarked && responseMsg.Win {
			break
		}
	}

	// Verify win message
	if responseMsg.Type != TypeTileMarked {
		t.Errorf("Expected response type %s, got %s", TypeTileMarked, responseMsg.Type)
	}
	if !responseMsg.Win {
		t.Error("Expected win condition to be true")
	}
}

func TestMultipleClients(t *testing.T) {
	g, _, server := setupTestServer(t)
	defer server.Close()

	// Connect first client
	conn1 := connectClient(t, server)
	defer conn1.Close()

	// Connect second client
	conn2 := connectClient(t, server)
	defer conn2.Close()

	// Set names for both clients
	setNameMsg1 := Message{
		Type: TypeSetName,
		Name: "Player 1",
	}
	setNameMsg2 := Message{
		Type: TypeSetName,
		Name: "Player 2",
	}

	msgBytes1, _ := json.Marshal(setNameMsg1)
	msgBytes2, _ := json.Marshal(setNameMsg2)

	conn1.WriteMessage(websocket.TextMessage, msgBytes1)
	conn2.WriteMessage(websocket.TextMessage, msgBytes2)
	time.Sleep(100 * time.Millisecond) // Wait for names to be set

	// Verify both clients are in the list
	clients := g.GetClients()
	if len(clients) != 2 {
		t.Errorf("Expected 2 clients, got %d", len(clients))
	}
	if _, exists := clients["Player 1"]; !exists {
		t.Error("Expected Player 1 to be in client list")
	}
	if _, exists := clients["Player 2"]; !exists {
		t.Error("Expected Player 2 to be in client list")
	}
}

func TestClientDisconnect(t *testing.T) {
	g, _, server := setupTestServer(t)
	defer server.Close()

	// Connect client
	conn := connectClient(t, server)

	// Set name
	setNameMsg := Message{
		Type: TypeSetName,
		Name: "Test Player",
	}
	msgBytes, _ := json.Marshal(setNameMsg)
	conn.WriteMessage(websocket.TextMessage, msgBytes)
	time.Sleep(100 * time.Millisecond) // Wait for name to be set

	// Verify client was added
	clients := g.GetClients()
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}

	// Close connection
	conn.Close()
	time.Sleep(100 * time.Millisecond) // Wait for cleanup

	// Verify client was removed
	clients = g.GetClients()
	if len(clients) != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", len(clients))
	}
}

func TestBroadcastClientList_RemovesFullChannelClient(t *testing.T) {
	g, h, _ := setupTestServer(t)

	// Create a fake client with a full channel
	client := &game.Client{
		ID:    "test-client",
		Name:  "FullChannel",
		Board: make([]bool, 16),
		Send:  make(chan []byte, 1), // Small buffer
	}
	// Fill the channel to simulate a full channel
	client.Send <- []byte("dummy")

	// Add client to the game
	g.AddClient(client)

	// Add a normal client
	normalClient := &game.Client{
		ID:    "test-client2",
		Name:  "Normal",
		Board: make([]bool, 16),
		Send:  make(chan []byte, 1),
	}
	g.AddClient(normalClient)

	// Call broadcastClientList
	h.broadcastClientList()

	// The full channel client should be removed
	clients := g.GetClients()
	if _, exists := clients["FullChannel"]; exists {
		t.Errorf("Expected full channel client to be removed, but it still exists")
	}
	if _, exists := clients["Normal"]; !exists {
		t.Errorf("Expected normal client to remain, but it was removed")
	}
}
