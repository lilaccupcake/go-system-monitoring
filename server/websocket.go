package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketUpgrader nâng cấp HTTP lên WebSocket
var WebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Cho phép tất cả origin (chỉ dùng cho development)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketClient đại diện cho một kết nối client
type WebSocketClient struct {
	conn *websocket.Conn
	send chan []byte
}

// WebSocketHub quản lý tất cả client connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex // Mutex để tránh data race
}

// NewWebSocketHub tạo hub mới
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run khởi chạy hub main loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Nếu buffer đầy, đóng client
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast gửi message đến tất cả clients
func (h *WebSocketHub) Broadcast(data interface{}) {
	message, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}
	h.broadcast <- message
}

// WebSocketHandler xử lý kết nối WebSocket mới
func (h *WebSocketHub) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Nâng cấp kết nối
	conn, err := WebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &WebSocketClient{
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	// Khởi chạy goroutines cho read và write
	go client.writePump(h)
	go client.readPump(h)
}

// readPump đọc messages từ client
func (c *WebSocketClient) readPump(h *WebSocketHub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// Agent không gửi message qua WebSocket, nên có thể bỏ qua
	}
}

// writePump gửi messages đến client
func (c *WebSocketClient) writePump(h *WebSocketHub) {
	defer c.conn.Close()

	for {
		message, ok := <-c.send
		if !ok {
			// Channel đã đóng
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error writing message: %v", err)
			return
		}
	}
}
