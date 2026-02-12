package application

import (
	"context"
	"time"

	chatDomain "github.com/Kilat-Pet-Delivery/service-tracking/internal/domain/chat"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/ws"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SendMessageRequest holds data to send a chat message.
type SendMessageRequest struct {
	MessageType string `json:"message_type" binding:"required"`
	Content     string `json:"content" binding:"required"`
}

// ChatMessageDTO is the API response representation of a chat message.
type ChatMessageDTO struct {
	ID         uuid.UUID `json:"id"`
	BookingID  uuid.UUID `json:"booking_id"`
	SenderID   uuid.UUID `json:"sender_id"`
	SenderRole string    `json:"sender_role"`
	MsgType    string    `json:"message_type"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// ChatService handles chat use cases.
type ChatService struct {
	repo   chatDomain.ChatRepository
	hub    *ws.Hub
	logger *zap.Logger
}

// NewChatService creates a new ChatService.
func NewChatService(repo chatDomain.ChatRepository, hub *ws.Hub, logger *zap.Logger) *ChatService {
	return &ChatService{repo: repo, hub: hub, logger: logger}
}

// SendMessage persists a chat message and broadcasts it via WebSocket.
func (s *ChatService) SendMessage(ctx context.Context, bookingID, senderID uuid.UUID, senderRole string, req SendMessageRequest) (*ChatMessageDTO, error) {
	msg, err := chatDomain.NewChatMessage(
		bookingID,
		senderID,
		senderRole,
		chatDomain.MessageType(req.MessageType),
		req.Content,
	)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, msg); err != nil {
		return nil, err
	}

	// Broadcast to WebSocket room
	s.hub.BroadcastChat(&ws.ChatMessage{
		Type:       "chat_message",
		BookingID:  bookingID,
		MessageID:  msg.ID(),
		SenderID:   senderID,
		SenderRole: senderRole,
		MsgType:    string(msg.MessageType()),
		Content:    msg.Content(),
		CreatedAt:  msg.CreatedAt(),
	})

	s.logger.Info("chat message sent",
		zap.String("booking_id", bookingID.String()),
		zap.String("sender_role", senderRole),
	)

	return toChatDTO(msg), nil
}

// GetMessages returns paginated chat history for a booking.
func (s *ChatService) GetMessages(ctx context.Context, bookingID uuid.UUID, page, limit int) ([]*ChatMessageDTO, int64, error) {
	offset := (page - 1) * limit
	messages, total, err := s.repo.FindByBookingID(ctx, bookingID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*ChatMessageDTO, len(messages))
	for i, m := range messages {
		dtos[i] = toChatDTO(m)
	}
	return dtos, total, nil
}

func toChatDTO(m *chatDomain.ChatMessage) *ChatMessageDTO {
	return &ChatMessageDTO{
		ID:         m.ID(),
		BookingID:  m.BookingID(),
		SenderID:   m.SenderID(),
		SenderRole: m.SenderRole(),
		MsgType:    string(m.MessageType()),
		Content:    m.Content(),
		CreatedAt:  m.CreatedAt(),
	}
}
