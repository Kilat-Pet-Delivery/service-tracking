package share

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// SharedTrip represents a publicly shareable link to a booking's tracking.
type SharedTrip struct {
	id         uuid.UUID
	bookingID  uuid.UUID
	shareToken string
	expiresAt  time.Time
	createdAt  time.Time
}

// NewSharedTrip creates a new shared trip with a random token and 24h expiry.
func NewSharedTrip(bookingID uuid.UUID) (*SharedTrip, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &SharedTrip{
		id:         uuid.New(),
		bookingID:  bookingID,
		shareToken: token,
		expiresAt:  now.Add(24 * time.Hour),
		createdAt:  now,
	}, nil
}

// Reconstruct rebuilds a SharedTrip from persistence.
func Reconstruct(id, bookingID uuid.UUID, shareToken string, expiresAt, createdAt time.Time) *SharedTrip {
	return &SharedTrip{
		id:         id,
		bookingID:  bookingID,
		shareToken: shareToken,
		expiresAt:  expiresAt,
		createdAt:  createdAt,
	}
}

// IsExpired returns true if the share link has expired.
func (s *SharedTrip) IsExpired() bool {
	return time.Now().UTC().After(s.expiresAt)
}

// Getters.
func (s *SharedTrip) ID() uuid.UUID        { return s.id }
func (s *SharedTrip) BookingID() uuid.UUID  { return s.bookingID }
func (s *SharedTrip) ShareToken() string    { return s.shareToken }
func (s *SharedTrip) ExpiresAt() time.Time  { return s.expiresAt }
func (s *SharedTrip) CreatedAt() time.Time  { return s.createdAt }

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
