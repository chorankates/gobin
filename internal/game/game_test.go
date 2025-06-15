package game

import (
	"os"
	"testing"
)

func TestCheckWin(t *testing.T) {
	tests := []struct {
		name     string
		board    []bool
		expected bool
	}{
		{
			name:     "No win",
			board:    make([]bool, 16),
			expected: false,
		},
		{
			name: "Row win",
			board: []bool{
				true, true, true, true,
				false, false, false, false,
				false, false, false, false,
				false, false, false, false,
			},
			expected: true,
		},
		{
			name: "Column win",
			board: []bool{
				true, false, false, false,
				true, false, false, false,
				true, false, false, false,
				true, false, false, false,
			},
			expected: true,
		},
		{
			name: "Diagonal win (top-left to bottom-right)",
			board: []bool{
				true, false, false, false,
				false, true, false, false,
				false, false, true, false,
				false, false, false, true,
			},
			expected: true,
		},
		{
			name: "Diagonal win (top-right to bottom-left)",
			board: []bool{
				false, false, false, true,
				false, false, true, false,
				false, true, false, false,
				true, false, false, false,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckWin(tt.board); got != tt.expected {
				t.Errorf("CheckWin() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCountMarkedTiles(t *testing.T) {
	tests := []struct {
		name     string
		board    []bool
		expected int
	}{
		{
			name:     "No marked tiles",
			board:    make([]bool, 16),
			expected: 0,
		},
		{
			name: "All marked tiles",
			board: []bool{
				true, true, true, true,
				true, true, true, true,
				true, true, true, true,
				true, true, true, true,
			},
			expected: 16,
		},
		{
			name: "Some marked tiles",
			board: []bool{
				true, false, true, false,
				false, true, false, true,
				true, false, true, false,
				false, true, false, true,
			},
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countMarkedTiles(tt.board); got != tt.expected {
				t.Errorf("countMarkedTiles() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGameClientManagement(t *testing.T) {
	game := New()

	// Test adding a client
	client := &Client{
		ID:    "test-client",
		Name:  "Test Player",
		Board: make([]bool, 16),
	}
	game.AddClient(client)

	// Test getting clients
	clients := game.GetClients()
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}
	if count, exists := clients["Test Player"]; !exists || count != 0 {
		t.Errorf("Expected client 'Test Player' with 0 marked tiles, got %v", clients)
	}

	// Test removing a client
	game.RemoveClient(client)
	clients = game.GetClients()
	if len(clients) != 0 {
		t.Errorf("Expected 0 clients after removal, got %d", len(clients))
	}
}

func TestLoadWordSet(t *testing.T) {
	// Create a temporary word set file
	content := `["word1", "word2", "word3"]`
	tmpfile, err := os.CreateTemp("", "wordset-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test loading the word set
	game := New()
	if err := game.LoadWordSet(tmpfile.Name()); err != nil {
		t.Errorf("LoadWordSet() error = %v", err)
	}

	// Test shuffling words
	shuffled := game.ShuffleWords()
	if len(shuffled) != 3 {
		t.Errorf("Expected 3 words, got %d", len(shuffled))
	}
}
