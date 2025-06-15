package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for now (we can restrict this later)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Message represents a WebSocket message
type Message struct {
	Type    string         `json:"type"`
	Index   int            `json:"index,omitempty"`
	Board   []bool         `json:"board,omitempty"`
	Words   []string       `json:"words,omitempty"`
	Win     bool           `json:"win,omitempty"`
	Name    string         `json:"name,omitempty"`
	Clients map[string]int `json:"clients,omitempty"` // Changed to map[string]int to store marked count
}

// Client represents a connected WebSocket client
type Client struct {
	conn  *websocket.Conn
	send  chan []byte
	board []bool
	name  string
}

// Game represents the bingo game state
type Game struct {
	clients map[*Client]bool
	words   []string
}

var game = &Game{
	clients: make(map[*Client]bool),
}

func loadWordSet() error {
	data, err := os.ReadFile("static/sets/corporate.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &game.words)
}

func generateName() string {
	adjectives := []string{"Happy", "Clever", "Swift", "Brave", "Witty", "Calm", "Eager", "Fierce", "Gentle", "Jolly"}
	nouns := []string{"Panda", "Tiger", "Eagle", "Dolphin", "Fox", "Lion", "Bear", "Wolf", "Hawk", "Owl"}
	return fmt.Sprintf("%s %s", adjectives[rand.Intn(len(adjectives))], nouns[rand.Intn(len(nouns))])
}

func shuffleWords(words []string) []string {
	// Create a copy of the words slice
	shuffled := make([]string, len(words))
	copy(shuffled, words)

	// Shuffle the copy
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled
}

func countMarkedTiles(board []bool) int {
	count := 0
	for _, marked := range board {
		if marked {
			count++
		}
	}
	return count
}

func broadcastClientList() {
	// Create a map of client names and their marked tile counts
	clientInfo := make(map[string]int)
	for client := range game.clients {
		clientInfo[client.name] = countMarkedTiles(client.board)
	}

	// Create the message
	msg := Message{
		Type:    "client_list",
		Clients: clientInfo,
	}

	// Convert to JSON
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling client list: %v", err)
		return
	}

	// Broadcast to all clients
	for client := range game.clients {
		select {
		case client.send <- jsonMsg:
		default:
			close(client.send)
			delete(game.clients, client)
		}
	}
}

func checkWin(board []bool) bool {
	// Check rows
	for i := 0; i < 16; i += 4 {
		if board[i] && board[i+1] && board[i+2] && board[i+3] {
			return true
		}
	}

	// Check columns
	for i := 0; i < 4; i++ {
		if board[i] && board[i+4] && board[i+8] && board[i+12] {
			return true
		}
	}

	// Check diagonals
	if board[0] && board[5] && board[10] && board[15] {
		return true
	}
	if board[3] && board[6] && board[9] && board[12] {
		return true
	}

	return false
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Initialize client with a new board and name
	client := &Client{
		conn:  conn,
		send:  make(chan []byte, 256),
		board: make([]bool, 16), // 4x4 board
		name:  generateName(),
	}
	game.clients[client] = true

	// Send initial board state and randomized words to the client
	initialMsg := Message{
		Type:  "init_board",
		Board: client.board,
		Words: shuffleWords(game.words),
		Name:  client.name,
	}
	if msg, err := json.Marshal(initialMsg); err == nil {
		client.send <- msg
	}

	// Broadcast updated client list
	broadcastClientList()

	// Start goroutines for reading and writing
	go client.readPump()
	go client.writePump()
}

func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
		delete(game.clients, c)
		broadcastClientList() // Update client list when someone disconnects
	}()

	for {
		_, message, err := c.conn.ReadMessage()
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

		if msg.Type == "mark_tile" && msg.Index >= 0 && msg.Index < 16 {
			// Update this client's board
			c.board[msg.Index] = !c.board[msg.Index]

			// Check for win
			hasWon := checkWin(c.board)

			// Send updated board state back to this client
			response := Message{
				Type:  "tile_marked",
				Index: msg.Index,
				Board: c.board,
				Win:   hasWon,
			}
			if responseMsg, err := json.Marshal(response); err == nil {
				c.send <- responseMsg
			}

			// Broadcast updated client list with new marked count
			broadcastClientList()

			// If there's a win, notify all other clients
			if hasWon {
				winMsg := Message{
					Type: "player_won",
					Name: c.name,
				}
				if winResponse, err := json.Marshal(winMsg); err == nil {
					for client := range game.clients {
						if client != c {
							client.send <- winResponse
						}
					}
				}
			}
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Load word set
	if err := loadWordSet(); err != nil {
		log.Fatalf("Failed to load word set: %v", err)
	}

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// Handle WebSocket connections
	http.HandleFunc("/ws", handleWebSocket)

	// Start the server
	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
