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
)
