package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
)

// Game represents the bingo game state
type Game struct {
	clients map[*Client]bool
	words   []string
	mu      sync.RWMutex
}

// Client represents a connected game client
type Client struct {
	ID    string
	Name  string
	Board []bool
}

// New creates a new game instance
func New() *Game {
	return &Game{
		clients: make(map[*Client]bool),
	}
}

// LoadWordSet loads the word set from a JSON file
func (g *Game) LoadWordSet(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read word set: %w", err)
	}
	return json.Unmarshal(data, &g.words)
}

// GenerateName creates a random name for a client
func GenerateName() string {
	adjectives := []string{"Happy", "Clever", "Swift", "Brave", "Witty", "Calm", "Eager", "Fierce", "Gentle", "Jolly"}
	nouns := []string{"Panda", "Tiger", "Eagle", "Dolphin", "Fox", "Lion", "Bear", "Wolf", "Hawk", "Owl"}
	return fmt.Sprintf("%s %s", adjectives[rand.Intn(len(adjectives))], nouns[rand.Intn(len(nouns))])
}

// ShuffleWords returns a shuffled copy of the word set
func (g *Game) ShuffleWords() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	shuffled := make([]string, len(g.words))
	copy(shuffled, g.words)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

// AddClient adds a new client to the game
func (g *Game) AddClient(client *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.clients[client] = true
}

// RemoveClient removes a client from the game
func (g *Game) RemoveClient(client *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.clients, client)
}

// GetClients returns a map of client names to their marked tile counts
func (g *Game) GetClients() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clientInfo := make(map[string]int)
	for client := range g.clients {
		clientInfo[client.Name] = countMarkedTiles(client.Board)
	}
	return clientInfo
}

// CheckWin checks if a board has a winning pattern
func CheckWin(board []bool) bool {
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

// countMarkedTiles returns the number of marked tiles in a board
func countMarkedTiles(board []bool) int {
	count := 0
	for _, marked := range board {
		if marked {
			count++
		}
	}
	return count
}
