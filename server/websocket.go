package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

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

// Heartbeat configuration
const (
	PingPeriod  = 30 * time.Second // Gửi ping mỗi 30 giây
	PongWait    = 10 * time.Second // Đợi pong tối đa 10 giây
	WriteWait   = 5 * time.Second  // Timeout cho write
	ReadTimeout = 60 * time.Second // Read timeout tổng thể
)

// WebSocketClient đại diện cho một kết nối client
type WebSocketClient struct {
	conn         *websocket.Conn
	send         chan []byte
	lastPongTime time.Time // Để track lần nhận pong cuối
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
		conn:         conn,
		send:         make(chan []byte, 256),
		lastPongTime: time.Now(),
	}

	h.register <- client

	// Khởi chạy goroutines cho read và write
	go client.writePump(h)
	go client.readPump(h)
	// Khởi chạy goroutine monitor để gửi ping định kỳ
	go client.pingPump(h)
}

// readPump đọc messages từ client
func (c *WebSocketClient) readPump(h *WebSocketHub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline
	c.conn.SetReadDeadline(time.Now().Add(ReadTimeout))
	// Set pong handler để reset deadline khi nhận pong
	c.conn.SetPongHandler(func(string) error {
		c.lastPongTime = time.Now()
		c.conn.SetReadDeadline(time.Now().Add(ReadTimeout))
		return nil
	})

	for {
		messageType, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// Xử lý pong message (nếu có)
		if messageType == websocket.PongMessage {
			c.lastPongTime = time.Now()
			continue
		}
		// Frontend không gửi message, bỏ qua
	}
}

// writePump gửi messages đến client
func (c *WebSocketClient) writePump(_ *WebSocketHub) {
	defer c.conn.Close()

	// Set write deadline
	c.conn.SetWriteDeadline(time.Now().Add(WriteWait))

	for {
		message, ok := <-c.send
		if !ok {
			// Channel đã đóng
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// Reset write deadline trước mỗi lần write
		c.conn.SetWriteDeadline(time.Now().Add(WriteWait))

		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error writing message: %v", err)
			return
		}
	}
}

// pingPump gửi ping định kỳ để kiểm tra kết nối
func (c *WebSocketClient) pingPump(_ *WebSocketHub) {
	ticker := time.NewTicker(PingPeriod)
	defer ticker.Stop()

	for range ticker.C {
		// Kiểm tra nếu client không phản hồi pong trong PongWait
		if time.Since(c.lastPongTime) > PongWait {
			log.Printf("Client ping timeout, closing connection")
			c.conn.Close()
			return
		}

		// Gửi ping message
		c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
		if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Printf("Error sending ping: %v", err)
			return
		}
	}
}
