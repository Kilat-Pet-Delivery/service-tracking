package repository

import (
	"context"
	"time"

	chatDomain "github.com/Kilat-Pet-Delivery/service-tracking/internal/domain/chat"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatMessageModel is the GORM model for the chat_messages table.
type ChatMessageModel struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	BookingID  uuid.UUID `gorm:"type:uuid;not null;index"`
	SenderID   uuid.UUID `gorm:"type:uuid;not null"`
	SenderRole string    `gorm:"type:varchar(20);not null"`
	MsgType    string    `gorm:"column:message_type;type:varchar(20);not null"`
	Content    string    `gorm:"type:text;not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

// TableName sets the table name.
func (ChatMessageModel) TableName() string { return "chat_messages" }

// GormChatRepository implements ChatRepository using GORM.
type GormChatRepository struct {
	db *gorm.DB
}

// NewGormChatRepository creates a new GormChatRepository.
func NewGormChatRepository(db *gorm.DB) *GormChatRepository {
	return &GormChatRepository{db: db}
}

// Save persists a new chat message.
func (r *GormChatRepository) Save(ctx context.Context, msg *chatDomain.ChatMessage) error {
	model := toChatModel(msg)
	return r.db.WithContext(ctx).Create(&model).Error
}

// FindByBookingID returns paginated chat messages for a booking.
func (r *GormChatRepository) FindByBookingID(ctx context.Context, bookingID uuid.UUID, limit, offset int) ([]*chatDomain.ChatMessage, int64, error) {
	var models []ChatMessageModel
	var total int64

	query := r.db.WithContext(ctx).Where("booking_id = ?", bookingID)
	query.Model(&ChatMessageModel{}).Count(&total)

	if err := query.Order("created_at ASC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	messages := make([]*chatDomain.ChatMessage, len(models))
	for i, m := range models {
		messages[i] = toChatDomain(&m)
	}
	return messages, total, nil
}

func toChatModel(m *chatDomain.ChatMessage) ChatMessageModel {
	return ChatMessageModel{
		ID:         m.ID(),
		BookingID:  m.BookingID(),
		SenderID:   m.SenderID(),
		SenderRole: m.SenderRole(),
		MsgType:    string(m.MessageType()),
		Content:    m.Content(),
		CreatedAt:  m.CreatedAt(),
	}
}

func toChatDomain(m *ChatMessageModel) *chatDomain.ChatMessage {
	return chatDomain.Reconstruct(
		m.ID,
		m.BookingID,
		m.SenderID,
		m.SenderRole,
		chatDomain.MessageType(m.MsgType),
		m.Content,
		m.CreatedAt,
	)
}
