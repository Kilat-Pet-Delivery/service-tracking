package chat

import (
	"context"

	"github.com/google/uuid"
)

// ChatRepository defines persistence operations for chat messages.
type ChatRepository interface {
	Save(ctx context.Context, msg *ChatMessage) error
	FindByBookingID(ctx context.Context, bookingID uuid.UUID, limit, offset int) ([]*ChatMessage, int64, error)
}
