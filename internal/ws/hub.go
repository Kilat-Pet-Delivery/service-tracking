package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// pingPeriod is the interval for sending pings to peer. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum message size allowed from peer.
	maxMessageSize = 512
)

// TrackingUpdate represents a real-time GPS position update sent to WebSocket clients.
type TrackingUpdate struct {
	BookingID uuid.UUID `json:"booking_id"`
	RunnerID  uuid.UUID `json:"runner_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Speed     float64   `json:"speed_kmh"`
	Heading   float64   `json:"heading_degrees"`
	Timestamp time.Time `json:"timestamp"`
}

// Client represents a single WebSocket connection subscribed to a booking's tracking.
type Client struct {
	Conn      *websocket.Conn
	BookingID uuid.UUID
	Send      chan []byte
}

// ChatMessage represents a chat message sent via WebSocket.
type ChatMessage struct {
	Type       string    `json:"type"` // always "chat_message"
	BookingID  uuid.UUID `json:"booking_id"`
	MessageID  uuid.UUID `json:"message_id"`
	SenderID   uuid.UUID `json:"sender_id"`
	SenderRole string    `json:"sender_role"`
	MsgType    string    `json:"message_type"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// Hub manages WebSocket connections organized by booking rooms.
type Hub struct {
	rooms      map[uuid.UUID]map[*Client]bool // bookingID -> set of clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan *TrackingUpdate
	chatBcast  chan *ChatMessage
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		rooms:      make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *TrackingUpdate, 256),
		chatBcast:  make(chan *ChatMessage, 256),
		logger:     logger,
	}
}

// Run starts the hub's event loop. Should be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.rooms[client.BookingID]; !ok {
				h.rooms[client.BookingID] = make(map[*Client]bool)
			}
			h.rooms[client.BookingID][client] = true
			h.mu.Unlock()

			h.logger.Debug("client registered",
				zap.String("booking_id", client.BookingID.String()),
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.BookingID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.rooms, client.BookingID)
					}
				}
			}
			h.mu.Unlock()

			h.logger.Debug("client unregistered",
				zap.String("booking_id", client.BookingID.String()),
			)

		case update := <-h.broadcast:
			data, err := json.Marshal(map[string]interface{}{
				"type": "location_update",
				"data": update,
			})
			if err != nil {
				h.logger.Error("failed to marshal tracking update", zap.Error(err))
				continue
			}

			h.broadcastToRoom(update.BookingID, data)

		case chatMsg := <-h.chatBcast:
			data, err := json.Marshal(chatMsg)
			if err != nil {
				h.logger.Error("failed to marshal chat message", zap.Error(err))
				continue
			}

			h.broadcastToRoom(chatMsg.BookingID, data)
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a tracking update to all clients watching the specified booking.
func (h *Hub) Broadcast(update *TrackingUpdate) {
	h.broadcast <- update
}

// BroadcastChat sends a chat message to all clients watching the specified booking.
func (h *Hub) BroadcastChat(msg *ChatMessage) {
	h.chatBcast <- msg
}

// broadcastToRoom sends raw data to all clients in a booking room.
func (h *Hub) broadcastToRoom(bookingID uuid.UUID, data []byte) {
	h.mu.RLock()
	clients, ok := h.rooms[bookingID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.Send <- data:
		default:
			h.mu.Lock()
			delete(clients, client)
			close(client.Send)
			if len(clients) == 0 {
				delete(h.rooms, bookingID)
			}
			h.mu.Unlock()
		}
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub.
// It reads messages and discards them (clients only receive, they don't send tracking data).
func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				hub.logger.Warn("websocket read error", zap.Error(err))
			}
			break
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
func (c *Client) WritePump(hub *Hub) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Drain any queued messages into the current write.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte("\n"))
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
