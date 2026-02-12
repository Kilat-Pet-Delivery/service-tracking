package repository

import (
	"context"
	"time"

	shareDomain "github.com/Kilat-Pet-Delivery/service-tracking/internal/domain/share"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SharedTripModel is the GORM model for the shared_trips table.
type SharedTripModel struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	BookingID  uuid.UUID `gorm:"type:uuid;not null;index"`
	ShareToken string    `gorm:"type:varchar(64);uniqueIndex;not null"`
	ExpiresAt  time.Time `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

// TableName sets the table name.
func (SharedTripModel) TableName() string { return "shared_trips" }

// GormSharedTripRepository implements SharedTripRepository using GORM.
type GormSharedTripRepository struct {
	db *gorm.DB
}

// NewGormSharedTripRepository creates a new GormSharedTripRepository.
func NewGormSharedTripRepository(db *gorm.DB) *GormSharedTripRepository {
	return &GormSharedTripRepository{db: db}
}

// Save persists a new shared trip.
func (r *GormSharedTripRepository) Save(ctx context.Context, st *shareDomain.SharedTrip) error {
	model := toShareModel(st)
	return r.db.WithContext(ctx).Create(&model).Error
}

// FindByToken returns a shared trip by its token.
func (r *GormSharedTripRepository) FindByToken(ctx context.Context, token string) (*shareDomain.SharedTrip, error) {
	var model SharedTripModel
	if err := r.db.WithContext(ctx).Where("share_token = ?", token).First(&model).Error; err != nil {
		return nil, err
	}
	return toShareDomain(&model), nil
}

// FindByBookingID returns a shared trip by booking ID.
func (r *GormSharedTripRepository) FindByBookingID(ctx context.Context, bookingID uuid.UUID) (*shareDomain.SharedTrip, error) {
	var model SharedTripModel
	if err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).Order("created_at DESC").First(&model).Error; err != nil {
		return nil, err
	}
	return toShareDomain(&model), nil
}

func toShareModel(s *shareDomain.SharedTrip) SharedTripModel {
	return SharedTripModel{
		ID:         s.ID(),
		BookingID:  s.BookingID(),
		ShareToken: s.ShareToken(),
		ExpiresAt:  s.ExpiresAt(),
		CreatedAt:  s.CreatedAt(),
	}
}

func toShareDomain(m *SharedTripModel) *shareDomain.SharedTrip {
	return shareDomain.Reconstruct(
		m.ID,
		m.BookingID,
		m.ShareToken,
		m.ExpiresAt,
		m.CreatedAt,
	)
}
