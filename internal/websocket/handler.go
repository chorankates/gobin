package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

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
		log.Printf("[%s] Failed to upgrade connection: %v", time.Now().Format(time.RFC3339), err)
		return
	}

	// Create new client with temporary name
	client := &game.Client{
		ID:    fmt.Sprintf("client-%d", len(h.game.GetClients())+1),
		Name:  "Anonymous", // Will be updated when client sends their name
		Board: make([]bool, 16),
		Conn:  conn,
		Send:  make(chan []byte, 256),
	}
	h.game.AddClient(client)
	log.Printf("[%s] New client connected: %s", time.Now().Format(time.RFC3339), client.ID)

	// Start goroutines for reading and writing
	go h.readPump(client)
	go h.writePump(client)
}

// readPump pumps messages from the WebSocket connection to the hub.
func (h *Handler) readPump(client *game.Client) {
	defer func() {
		log.Printf("[%s] Client disconnected: %s (Name: %s)", time.Now().Format(time.RFC3339), client.ID, client.Name)
		client.Conn.Close()
		h.game.RemoveClient(client)
		h.broadcastClientList() // Update client list when someone disconnects
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[%s] WebSocket error for client %s: %v", time.Now().Format(time.RFC3339), client.ID, err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[%s] Error unmarshaling message from client %s: %v", time.Now().Format(time.RFC3339), client.ID, err)
			continue
		}

		switch msg.Type {
		case TypeSetName:
			if msg.Name != "" {
				oldName := client.Name
				client.Name = msg.Name
				log.Printf("[%s] Client %s set name: %s (was: %s)", time.Now().Format(time.RFC3339), client.ID, msg.Name, oldName)
				// Send initial board state after name is set
				response := Message{
					Type:    TypeInitBoard,
					Board:   client.Board,
					Words:   h.game.ShuffleWords(),
					Name:    client.Name,
					Clients: h.game.GetClientBoards(),
				}
				if responseMsg, err := json.Marshal(response); err == nil {
					client.Send <- responseMsg
				}
				// Broadcast updated client list
				h.broadcastClientList()
			}

		case TypeMarkTile:
			if msg.Index >= 0 && msg.Index < 16 {
				// Update this client's board
				client.Board[msg.Index] = !client.Board[msg.Index]
				log.Printf("[%s] Client %s marked tile %d (sequence: %d)", time.Now().Format(time.RFC3339), client.Name, msg.Index, msg.Sequence)

				// Check for win
				hasWon := game.CheckWin(client.Board)
				if hasWon {
					log.Printf("[%s] Client %s has won the game!", time.Now().Format(time.RFC3339), client.Name)
				}

				// Send updated board state back to this client
				response := Message{
					Type:     TypeTileMarked,
					Index:    msg.Index,
					Board:    client.Board,
					Win:      hasWon,
					Sequence: msg.Sequence,
				}
				if responseMsg, err := json.Marshal(response); err == nil {
					select {
					case client.Send <- responseMsg:
						// Message sent successfully
					default:
						log.Printf("[%s] Failed to send response to client %s - channel full", time.Now().Format(time.RFC3339), client.Name)
						client.Conn.Close()
						return
					}
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
								log.Printf("[%s] Notifying client %s about %s's win", time.Now().Format(time.RFC3339), otherClient.Name, client.Name)
								otherClient.Send <- winResponse
							}
						}
					}
				}
			}
		}
	}
}

// writePumps messages from the hub to the WebSocket connection.
func (h *Handler) writePump(client *game.Client) {
	defer func() {
		log.Printf("[%s] Write pump closed for client %s", time.Now().Format(time.RFC3339), client.Name)
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

			// Write the first message
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[%s] Error writing message to client %s: %v", time.Now().Format(time.RFC3339), client.Name, err)
				return
			}

			// Write any queued messages
			n := len(client.Send)
			for i := 0; i < n; i++ {
				message = <-client.Send
				if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Printf("[%s] Error writing queued message to client %s: %v", time.Now().Format(time.RFC3339), client.Name, err)
					return
				}
			}
		}
	}
}

// broadcastClientList sends the current client list to all connected clients
func (h *Handler) broadcastClientList() {
	clientList := h.game.GetClientBoards()
	log.Printf("[%s] Broadcasting client list update. Current clients: %v", time.Now().Format(time.RFC3339), clientList)
	msg := Message{
		Type:    TypeClientList,
		Clients: clientList,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[%s] Error marshaling client list: %v", time.Now().Format(time.RFC3339), err)
		return
	}
	for _, client := range h.game.GetAllClients() {
		select {
		case client.Send <- msgBytes:
		default:
			log.Printf("[%s] Failed to send client list to client %s - channel full or closed", time.Now().Format(time.RFC3339), client.Name)
			h.game.RemoveClient(client)
		}
	}
}
