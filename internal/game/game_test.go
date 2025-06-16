package game

import (
	"os"
	"testing"
)

func TestCheckWin(t *testing.T) {
	tests := []struct {
		name     string
		board    []string
		expected bool
	}{
		{
			name: "No win",
			board: []string{
				"", "", "", "",
				"", "", "", "",
				"", "", "", "",
				"", "", "", "",
			},
			expected: false,
		},
		{
			name: "Row win",
			board: []string{
				"word1", "word2", "word3", "word4",
				"", "", "", "",
				"", "", "", "",
				"", "", "", "",
			},
			expected: true,
		},
		{
			name: "Column win",
			board: []string{
				"word1", "", "", "",
				"word2", "", "", "",
				"word3", "", "", "",
				"word4", "", "", "",
			},
			expected: true,
		},
		{
			name: "Diagonal win 1",
			board: []string{
				"word1", "", "", "",
				"", "word2", "", "",
				"", "", "word3", "",
				"", "", "", "word4",
			},
			expected: true,
		},
		{
			name: "Diagonal win 2",
			board: []string{
				"", "", "", "word1",
				"", "", "word2", "",
				"", "word3", "", "",
				"word4", "", "", "",
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
		board    []string
		expected int
	}{
		{
			name: "Empty board",
			board: []string{
				"", "", "", "",
				"", "", "", "",
				"", "", "", "",
				"", "", "", "",
			},
			expected: 0,
		},
		{
			name: "Partially filled board",
			board: []string{
				"word1", "word2", "", "",
				"", "word3", "", "",
				"", "", "word4", "",
				"", "", "", "",
			},
			expected: 4,
		},
		{
			name: "Full board",
			board: []string{
				"word1", "word2", "word3", "word4",
				"word5", "word6", "word7", "word8",
				"word9", "word10", "word11", "word12",
				"word13", "word14", "word15", "word16",
			},
			expected: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CountMarkedTiles(tt.board); got != tt.expected {
				t.Errorf("CountMarkedTiles() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculateWinProgress(t *testing.T) {
	tests := []struct {
		name     string
		board    []string
		expected int
	}{
		{
			name: "Empty board",
			board: []string{
				"", "", "", "",
				"", "", "", "",
				"", "", "", "",
				"", "", "", "",
			},
			expected: 0,
		},
		{
			name: "Row with 3 marked",
			board: []string{
				"word1", "word2", "word3", "",
				"", "", "", "",
				"", "", "", "",
				"", "", "", "",
			},
			expected: 3,
		},
		{
			name: "Column with 3 marked",
			board: []string{
				"word1", "", "", "",
				"word2", "", "", "",
				"word3", "", "", "",
				"", "", "", "",
			},
			expected: 3,
		},
		{
			name: "Diagonal with 3 marked",
			board: []string{
				"word1", "", "", "",
				"", "word2", "", "",
				"", "", "word3", "",
				"", "", "", "",
			},
			expected: 3,
		},
		{
			name: "Multiple lines with different progress",
			board: []string{
				"word1", "word2", "word3", "",
				"word4", "", "", "",
				"word5", "", "", "",
				"word6", "", "", "",
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateWinProgress(tt.board); got != tt.expected {
				t.Errorf("CalculateWinProgress() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetClientBoards(t *testing.T) {
	game := New()

	// Create test clients with different board states
	client1 := &Client{
		Name: "Player1",
		Board: []string{
			"word1", "word2", "word3", "",
			"", "", "", "",
			"", "", "", "",
			"", "", "", "",
		},
	}

	client2 := &Client{
		Name: "Player2",
		Board: []string{
			"word1", "", "", "",
			"word2", "", "", "",
			"word3", "", "", "",
			"", "", "", "",
		},
	}

	client3 := &Client{
		Name: "Player3",
		Board: []string{
			"", "", "", "",
			"", "", "", "",
			"", "", "", "",
			"", "", "", "",
		},
	}

	game.AddClient(client1)
	game.AddClient(client2)
	game.AddClient(client3)

	boards := game.GetClientBoards()

	// Verify the boards are returned in the correct order (by progress)
	expectedOrder := []string{"Player1", "Player2", "Player3"}
	i := 0
	for name := range boards {
		if name != expectedOrder[i] {
			t.Errorf("Expected client %s at position %d, got %s", expectedOrder[i], i, name)
		}
		i++
	}

	// Verify the board contents
	if len(boards["Player1"]) != 16 || boards["Player1"][0] != "word1" {
		t.Error("Player1's board not correctly stored")
	}
	if len(boards["Player2"]) != 16 || boards["Player2"][0] != "word1" {
		t.Error("Player2's board not correctly stored")
	}
	if len(boards["Player3"]) != 16 || boards["Player3"][0] != "" {
		t.Error("Player3's board not correctly stored")
	}
}

func TestGameClientManagement(t *testing.T) {
	game := New()

	// Create test client
	client := &Client{
		ID:   "test-client",
		Name: "Test Player",
		Board: []string{
			"", "", "", "",
			"", "", "", "",
			"", "", "", "",
			"", "", "", "",
		},
	}

	// Test adding client
	game.AddClient(client)
	if len(game.GetAllClients()) != 1 {
		t.Error("Expected 1 client after adding")
	}

	// Test removing client
	game.RemoveClient(client)
	if len(game.GetAllClients()) != 0 {
		t.Error("Expected 0 clients after removing")
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
