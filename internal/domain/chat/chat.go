package chat

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageType represents the type of chat message.
type MessageType string

const (
	MessageTypeText       MessageType = "text"
	MessageTypeImage      MessageType = "image"
	MessageTypeQuickReply MessageType = "quick_reply"
)

// IsValid returns true if the message type is recognized.
func (m MessageType) IsValid() bool {
	switch m {
	case MessageTypeText, MessageTypeImage, MessageTypeQuickReply:
		return true
	}
	return false
}

// ChatMessage is the aggregate root for chat messages.
type ChatMessage struct {
	id         uuid.UUID
	bookingID  uuid.UUID
	senderID   uuid.UUID
	senderRole string
	msgType    MessageType
	content    string
	createdAt  time.Time
}

// NewChatMessage creates a new chat message.
func NewChatMessage(bookingID, senderID uuid.UUID, senderRole string, msgType MessageType, content string) (*ChatMessage, error) {
	if !msgType.IsValid() {
		return nil, fmt.Errorf("invalid message type: %s", msgType)
	}
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	return &ChatMessage{
		id:         uuid.New(),
		bookingID:  bookingID,
		senderID:   senderID,
		senderRole: senderRole,
		msgType:    msgType,
		content:    content,
		createdAt:  time.Now().UTC(),
	}, nil
}

// Reconstruct rebuilds a ChatMessage from persistence.
func Reconstruct(id, bookingID, senderID uuid.UUID, senderRole string, msgType MessageType, content string, createdAt time.Time) *ChatMessage {
	return &ChatMessage{
		id:         id,
		bookingID:  bookingID,
		senderID:   senderID,
		senderRole: senderRole,
		msgType:    msgType,
		content:    content,
		createdAt:  createdAt,
	}
}

// Getters.
func (m *ChatMessage) ID() uuid.UUID         { return m.id }
func (m *ChatMessage) BookingID() uuid.UUID   { return m.bookingID }
func (m *ChatMessage) SenderID() uuid.UUID    { return m.senderID }
func (m *ChatMessage) SenderRole() string     { return m.senderRole }
func (m *ChatMessage) MessageType() MessageType { return m.msgType }
func (m *ChatMessage) Content() string        { return m.content }
func (m *ChatMessage) CreatedAt() time.Time   { return m.createdAt }
