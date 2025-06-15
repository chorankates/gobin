package websocket

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"gobin/internal/game"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Handler manages WebSocket connections and game state
type Handler struct {
	game *game.Game
	mu   sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	Type    string         `json:"type"`
	Index   int            `json:"index,omitempty"`
	Board   []bool         `json:"board,omitempty"`
	Words   []string       `json:"words,omitempty"`
	Win     bool           `json:"win,omitempty"`
	Name    string         `json:"name,omitempty"`
	Clients map[string]int `json:"clients,omitempty"`
}

// New creates a new WebSocket handler
func New(g *game.Game) *Handler {
	return &Handler{
		game: g,
	}
}

// HandleWebSocket upgrades HTTP connections to WebSocket and manages the game
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Create new client
	client := &game.Client{
		ID:    fmt.Sprintf("client-%d", len(h.game.GetClients())+1),
		Name:  game.GenerateName(),
		Board: make([]bool, 16),
	}
	h.game.AddClient(client)
	defer h.game.RemoveClient(client)

	// Send initial board state
	msg := Message{
		Type:    "init",
		Words:   h.game.ShuffleWords(),
		Name:    client.Name,
		Clients: h.game.GetClients(),
	}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Failed to send initial state: %v", err)
		return
	}

	// Broadcast updated client list
	h.broadcastClientList()

	// Handle messages
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Failed to read message: %v", err)
			break
		}

		switch msg.Type {
		case "mark":
			if msg.Index >= 0 && msg.Index < 16 {
				client.Board[msg.Index] = true

				// Check for win
				if game.CheckWin(client.Board) {
					// Notify winner
					winMsg := Message{
						Type:  "win",
						Win:   true,
						Name:  client.Name,
						Board: client.Board,
					}
					if err := conn.WriteJSON(winMsg); err != nil {
						log.Printf("Failed to send win message: %v", err)
					}

					// Notify other clients
					h.broadcastClientList()
				} else {
					// Send updated board
					updateMsg := Message{
						Type:  "update",
						Board: client.Board,
					}
					if err := conn.WriteJSON(updateMsg); err != nil {
						log.Printf("Failed to send update: %v", err)
					}
				}
			}
		}
	}
}

// broadcastClientList sends the current client list to all connected clients
func (h *Handler) broadcastClientList() {
	// In a real implementation, we would need to maintain a list of active connections
	// and broadcast to all of them. For now, this is a placeholder.
	log.Printf("Current clients: %v", h.game.GetClients())
}
