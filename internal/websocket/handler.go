package websocket

import (
	"encoding/json"
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

	// Create new client
	client := &game.Client{
		ID:    fmt.Sprintf("client-%d", len(h.game.GetClients())+1),
		Name:  game.GenerateName(),
		Board: make([]bool, 16),
		Conn:  conn,
		Send:  make(chan []byte, 256),
	}
	h.game.AddClient(client)

	// Send initial board state
	msg := Message{
		Type:    TypeInitBoard,
		Board:   client.Board,
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

	// Start goroutines for reading and writing
	go h.readPump(client)
	go h.writePump(client)
}

// readPump pumps messages from the WebSocket connection to the hub.
func (h *Handler) readPump(client *game.Client) {
	defer func() {
		client.Conn.Close()
		h.game.RemoveClient(client)
		h.broadcastClientList() // Update client list when someone disconnects
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error unmarshaling message: %v", err)
			continue
		}

		if msg.Type == TypeMarkTile && msg.Index >= 0 && msg.Index < 16 {
			// Update this client's board
			client.Board[msg.Index] = !client.Board[msg.Index]

			// Check for win
			hasWon := game.CheckWin(client.Board)

			// Send updated board state back to this client
			response := Message{
				Type:  TypeTileMarked,
				Index: msg.Index,
				Board: client.Board,
				Win:   hasWon,
			}
			if responseMsg, err := json.Marshal(response); err == nil {
				client.Send <- responseMsg
			}

			// Broadcast updated client list with new marked count
			h.broadcastClientList()

			// If there's a win, notify all other clients
			if hasWon {
				winMsg := Message{
					Type: TypePlayerWon,
					Name: client.Name,
				}
				if winResponse, err := json.Marshal(winMsg); err == nil {
					for _, otherClient := range h.game.GetAllClients() {
						if otherClient != client {
							otherClient.Send <- winResponse
						}
					}
				}
			}
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (h *Handler) writePump(client *game.Client) {
	defer func() {
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// The hub closed the channel.
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message.
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

// broadcastClientList sends the current client list to all connected clients
func (h *Handler) broadcastClientList() {
	// Create a map of client names and their marked tile counts
	clientInfo := make(map[string]int)
	for _, client := range h.game.GetAllClients() {
		clientInfo[client.Name] = game.CountMarkedTiles(client.Board)
	}

	log.Printf("Broadcasting client list update. Current clients: %v", clientInfo)

	// Create the message
	msg := Message{
		Type:    TypeClientList,
		Clients: clientInfo,
	}

	// Convert to JSON
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling client list: %v", err)
		return
	}

	// Broadcast to all clients
	clients := h.game.GetAllClients()
	log.Printf("Sending client list to %d connected clients", len(clients))
	for _, client := range clients {
		select {
		case client.Send <- jsonMsg:
			log.Printf("Sent client list to client %s", client.Name)
		default:
			log.Printf("Failed to send client list to client %s - channel full or closed", client.Name)
			close(client.Send)
			h.game.RemoveClient(client)
		}
	}
}
