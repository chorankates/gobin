package websocket

// Message types for WebSocket communication
const (
	// Server to client messages
	TypeInitBoard  = "init_board"  // Initial board state with words and player info
	TypeTileMarked = "tile_marked" // Update when a tile is marked
	TypePlayerWon  = "player_won"  // Notification when a player wins
	TypeClientList = "client_list" // List of connected clients and their tile counts

	// Client to server messages
	TypeMarkTile = "mark_tile" // Request to mark/unmark a tile
	TypeSetName  = "set_name"  // Set player name
)

type Message struct {
	Type     string              `json:"type"`
	Name     string              `json:"name,omitempty"`
	Index    int                 `json:"index,omitempty"`
	Board    []string            `json:"board,omitempty"`
	Words    []string            `json:"words,omitempty"`
	Win      bool                `json:"win,omitempty"`
	Sequence int                 `json:"sequence,omitempty"`
	Clients  map[string][]string `json:"clients,omitempty"`
}
