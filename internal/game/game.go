package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"

	"github.com/gorilla/websocket"
)

const (
	BoardSize = 16
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
	Board []string
	Conn  *websocket.Conn
	Send  chan []byte
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
		clientInfo[client.Name] = CountMarkedTiles(client.Board)
	}
	return clientInfo
}

// CheckWin checks if a board has a winning pattern
func CheckWin(board []string) bool {
	// Check rows
	for i := 0; i < BoardSize; i += 4 {
		if board[i] != "" && board[i+1] != "" && board[i+2] != "" && board[i+3] != "" {
			return true
		}
	}

	// Check columns
	for i := 0; i < 4; i++ {
		if board[i] != "" && board[i+4] != "" && board[i+8] != "" && board[i+12] != "" {
			return true
		}
	}

	// Check diagonals
	if board[0] != "" && board[5] != "" && board[10] != "" && board[15] != "" {
		return true
	}
	if board[3] != "" && board[6] != "" && board[9] != "" && board[12] != "" {
		return true
	}

	return false
}

// GetAllClients returns a slice of all connected clients
func (g *Game) GetAllClients() []*Client {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clients := make([]*Client, 0, len(g.clients))
	for client := range g.clients {
		clients = append(clients, client)
	}
	return clients
}

// CountMarkedTiles returns the number of marked tiles on a board
func CountMarkedTiles(board []string) int {
	count := 0
	for _, value := range board {
		if value != "" {
			count++
		}
	}
	return count
}

// CalculateWinProgress returns a score indicating how close a board is to winning
// Higher score means closer to winning
func CalculateWinProgress(board []string) int {
	maxMarkedInLine := 0

	// Check rows
	for i := 0; i < BoardSize; i += 4 {
		markedInRow := 0
		for j := 0; j < 4; j++ {
			if board[i+j] != "" {
				markedInRow++
			}
		}
		if markedInRow > maxMarkedInLine {
			maxMarkedInLine = markedInRow
		}
	}

	// Check columns
	for i := 0; i < 4; i++ {
		markedInCol := 0
		for j := 0; j < BoardSize; j += 4 {
			if board[i+j] != "" {
				markedInCol++
			}
		}
		if markedInCol > maxMarkedInLine {
			maxMarkedInLine = markedInCol
		}
	}

	// Check diagonals
	markedInDiag1 := 0
	markedInDiag2 := 0
	for i := 0; i < 4; i++ {
		if board[i*4+i] != "" {
			markedInDiag1++
		}
		if board[i*4+(3-i)] != "" {
			markedInDiag2++
		}
	}
	if markedInDiag1 > maxMarkedInLine {
		maxMarkedInLine = markedInDiag1
	}
	if markedInDiag2 > maxMarkedInLine {
		maxMarkedInLine = markedInDiag2
	}

	return maxMarkedInLine
}

// GetClientBoards returns a map of client names to their board states, sorted by win progress
func (g *Game) GetClientBoards() map[string][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Create a slice of clients with their win progress
	type clientProgress struct {
		name     string
		board    []string
		progress int
	}
	clients := make([]clientProgress, 0, len(g.clients))
	for client := range g.clients {
		clients = append(clients, clientProgress{
			name:     client.Name,
			board:    client.Board,
			progress: CalculateWinProgress(client.Board),
		})
	}

	// Sort by progress (descending)
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].progress > clients[j].progress
	})

	// Create the result map
	result := make(map[string][]string)
	for _, client := range clients {
		result[client.name] = client.board
	}
	return result
}

// GetWords returns a copy of the word set
func (g *Game) GetWords() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	words := make([]string, len(g.words))
	copy(words, g.words)
	return words
}
