package application

import (
	"context"
	"fmt"
	"time"

	shareDomain "github.com/Kilat-Pet-Delivery/service-tracking/internal/domain/share"
	trackingDomain "github.com/Kilat-Pet-Delivery/service-tracking/internal/domain/tracking"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SharedTripDTO is the API response for a shared trip link.
type SharedTripDTO struct {
	ID         uuid.UUID `json:"id"`
	BookingID  uuid.UUID `json:"booking_id"`
	ShareToken string    `json:"share_token"`
	ShareURL   string    `json:"share_url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// SharedTrackingDTO is the public tracking data for a shared trip.
type SharedTrackingDTO struct {
	BookingID  uuid.UUID      `json:"booking_id"`
	Status     string         `json:"status"`
	Waypoints  []WaypointDTO  `json:"waypoints"`
	ExpiresAt  time.Time      `json:"expires_at"`
}

// WaypointDTO is the API representation of a waypoint.
type WaypointDTO struct {
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Speed      float64   `json:"speed_kmh"`
	Heading    float64   `json:"heading_degrees"`
	RecordedAt time.Time `json:"recorded_at"`
}

// ShareService handles trip sharing use cases.
type ShareService struct {
	shareRepo    shareDomain.SharedTripRepository
	trackingRepo trackingDomain.TripTrackRepository
	logger       *zap.Logger
}

// NewShareService creates a new ShareService.
func NewShareService(shareRepo shareDomain.SharedTripRepository, trackingRepo trackingDomain.TripTrackRepository, logger *zap.Logger) *ShareService {
	return &ShareService{shareRepo: shareRepo, trackingRepo: trackingRepo, logger: logger}
}

// CreateShareLink creates a new share link for a booking.
func (s *ShareService) CreateShareLink(ctx context.Context, bookingID uuid.UUID) (*SharedTripDTO, error) {
	st, err := shareDomain.NewSharedTrip(bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to create share link: %w", err)
	}

	if err := s.shareRepo.Save(ctx, st); err != nil {
		return nil, fmt.Errorf("failed to save share link: %w", err)
	}

	s.logger.Info("share link created",
		zap.String("booking_id", bookingID.String()),
		zap.String("token", st.ShareToken()),
	)

	return &SharedTripDTO{
		ID:         st.ID(),
		BookingID:  st.BookingID(),
		ShareToken: st.ShareToken(),
		ShareURL:   fmt.Sprintf("/api/v1/tracking/shared/%s", st.ShareToken()),
		ExpiresAt:  st.ExpiresAt(),
	}, nil
}

// GetSharedTracking returns public tracking data for a shared token (no auth needed).
func (s *ShareService) GetSharedTracking(ctx context.Context, token string) (*SharedTrackingDTO, error) {
	st, err := s.shareRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("share link not found")
	}

	if st.IsExpired() {
		return nil, fmt.Errorf("share link has expired")
	}

	track, err := s.trackingRepo.FindByBookingID(ctx, st.BookingID())
	if err != nil {
		return nil, fmt.Errorf("tracking data not found")
	}

	waypoints, err := s.trackingRepo.GetWaypoints(ctx, track.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to get waypoints: %w", err)
	}

	waypointDTOs := make([]WaypointDTO, len(waypoints))
	for i, wp := range waypoints {
		waypointDTOs[i] = WaypointDTO{
			Latitude:   wp.Latitude,
			Longitude:  wp.Longitude,
			Speed:      wp.Speed,
			Heading:    wp.Heading,
			RecordedAt: wp.RecordedAt,
		}
	}

	return &SharedTrackingDTO{
		BookingID:  st.BookingID(),
		Status:     string(track.Status()),
		Waypoints:  waypointDTOs,
		ExpiresAt:  st.ExpiresAt(),
	}, nil
}
