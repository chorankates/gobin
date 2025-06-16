package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

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
	game   *game.Game
	mu     sync.RWMutex
	logger *zap.Logger
}

// New creates a new WebSocket handler
func New(g *game.Game) *Handler {
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	return &Handler{
		game:   g,
		logger: logger,
	}
}

// HandleWebSocket upgrades HTTP connections to WebSocket and manages the game
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection",
			zap.Error(err),
		)
		return
	}

	// Create new client with temporary name
	client := &game.Client{
		ID:    fmt.Sprintf("client-%d", len(h.game.GetClients())+1),
		Name:  "Anonymous",        // Will be updated when client sends their name
		Board: make([]string, 16), // Initialize with empty strings
		Conn:  conn,
		Send:  make(chan []byte, 256),
	}
	h.game.AddClient(client)
	h.logger.Info("New client connected",
		zap.String("clientID", client.ID),
	)

	// Start goroutines for reading and writing
	go h.readPump(client)
	go h.writePump(client)
}

// readPump pumps messages from the WebSocket connection to the hub.
func (h *Handler) readPump(client *game.Client) {
	defer func() {
		h.logger.Info("Client disconnected",
			zap.String("clientID", client.ID),
			zap.String("clientName", client.Name),
		)
		client.Conn.Close()
		h.game.RemoveClient(client)
		h.broadcastClientList() // Update client list when someone disconnects
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket error",
					zap.String("clientID", client.ID),
					zap.Error(err),
				)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			h.logger.Error("Error unmarshaling message",
				zap.String("clientID", client.ID),
				zap.Error(err),
			)
			continue
		}

		switch msg.Type {
		case TypeSetName:
			if msg.Name != "" {
				oldName := client.Name
				client.Name = msg.Name
				h.logger.Info("Client set name",
					zap.String("clientID", client.ID),
					zap.String("oldName", oldName),
					zap.String("newName", msg.Name),
				)
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
				if client.Board[msg.Index] == "" {
					words := h.game.GetWords()
					if len(words) > msg.Index {
						client.Board[msg.Index] = words[msg.Index] // Mark with the word
					} else {
						h.logger.Error("Word index out of range",
							zap.String("clientName", client.Name),
							zap.Int("index", msg.Index),
							zap.Int("wordCount", len(words)),
						)
						continue
					}
				} else {
					client.Board[msg.Index] = "" // Unmark by setting to empty string
				}
				h.logger.Info("Client marked tile",
					zap.String("clientName", client.Name),
					zap.Int("tileIndex", msg.Index),
					zap.Int("sequence", msg.Sequence),
				)

				// Check for win
				hasWon := game.CheckWin(client.Board)
				if hasWon {
					h.logger.Info("Client won the game",
						zap.String("clientName", client.Name),
					)
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
						h.logger.Error("Failed to send response - channel full",
							zap.String("clientName", client.Name),
						)
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
								h.logger.Info("Notifying client about win",
									zap.String("notifiedClient", otherClient.Name),
									zap.String("winner", client.Name),
								)
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
		h.logger.Info("Write pump closed",
			zap.String("clientName", client.Name),
		)
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
				h.logger.Error("Error writing message",
					zap.String("clientName", client.Name),
					zap.Error(err),
				)
				return
			}

			// Write any queued messages
			n := len(client.Send)
			for i := 0; i < n; i++ {
				message = <-client.Send
				if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
					h.logger.Error("Error writing queued message",
						zap.String("clientName", client.Name),
						zap.Error(err),
					)
					return
				}
			}
		}
	}
}

// broadcastClientList sends the current client list to all connected clients
func (h *Handler) broadcastClientList() {
	clientList := h.game.GetClientBoards()
	h.logger.Info("Broadcasting client list update",
		zap.Any("clients", clientList),
	)
	msg := Message{
		Type:    TypeClientList,
		Clients: clientList,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Error marshaling client list",
			zap.Error(err),
		)
		return
	}
	for _, client := range h.game.GetAllClients() {
		select {
		case client.Send <- msgBytes:
		default:
			h.logger.Error("Failed to send client list - channel full or closed",
				zap.String("clientName", client.Name),
			)
			h.game.RemoveClient(client)
		}
	}
}
