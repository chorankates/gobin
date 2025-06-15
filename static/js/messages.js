// Message types for WebSocket communication
const MessageType = {
    // Server to client messages
    INIT_BOARD: 'init_board',   // Initial board state with words and player info
    TILE_MARKED: 'tile_marked', // Update when a tile is marked
    PLAYER_WON: 'player_won',   // Notification when a player wins
    CLIENT_LIST: 'client_list', // List of connected clients and their tile counts

    // Client to server messages
    MARK_TILE: 'mark_tile'      // Request to mark/unmark a tile
}; 