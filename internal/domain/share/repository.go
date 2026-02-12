package share

import (
	"context"

	"github.com/google/uuid"
)

// SharedTripRepository defines persistence operations for shared trips.
type SharedTripRepository interface {
	Save(ctx context.Context, st *SharedTrip) error
	FindByToken(ctx context.Context, token string) (*SharedTrip, error)
	FindByBookingID(ctx context.Context, bookingID uuid.UUID) (*SharedTrip, error)
}
