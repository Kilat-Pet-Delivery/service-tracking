package application

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	"github.com/Kilat-Pet-Delivery/lib-common/kafka"
	"github.com/Kilat-Pet-Delivery/lib-proto/events"
	trackingDomain "github.com/Kilat-Pet-Delivery/service-tracking/internal/domain/tracking"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/ws"
)

// WaypointDTO represents a waypoint in API responses.
type WaypointDTO struct {
	ID         uuid.UUID `json:"id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Speed      float64   `json:"speed_kmh"`
	Heading    float64   `json:"heading_degrees"`
	RecordedAt time.Time `json:"recorded_at"`
}

// TrackingDTO represents tracking data in API responses.
type TrackingDTO struct {
	ID              uuid.UUID     `json:"id"`
	BookingID       uuid.UUID     `json:"booking_id"`
	RunnerID        uuid.UUID     `json:"runner_id"`
	Status          string        `json:"status"`
	TotalDistanceKm float64      `json:"total_distance_km"`
	StartedAt       time.Time     `json:"started_at"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	Waypoints       []WaypointDTO `json:"waypoints"`
}

// TrackingService implements the application use cases for the tracking domain.
type TrackingService struct {
	repo     trackingDomain.TripTrackRepository
	hub      *ws.Hub
	producer *kafka.Producer
	logger   *zap.Logger
}

// NewTrackingService creates a new TrackingService.
func NewTrackingService(
	repo trackingDomain.TripTrackRepository,
	hub *ws.Hub,
	producer *kafka.Producer,
	logger *zap.Logger,
) *TrackingService {
	return &TrackingService{
		repo:     repo,
		hub:      hub,
		producer: producer,
		logger:   logger,
	}
}

// HandleBookingAccepted creates a new TripTrack when a booking is accepted by a runner.
func (s *TrackingService) HandleBookingAccepted(ctx context.Context, event events.BookingAcceptedEvent) error {
	s.logger.Info("handling booking accepted event",
		zap.String("booking_id", event.BookingID.String()),
		zap.String("runner_id", event.RunnerID.String()),
	)

	// Check if tracking already exists for this booking.
	existing, _ := s.repo.FindByBookingID(ctx, event.BookingID)
	if existing != nil {
		s.logger.Warn("tracking already exists for booking, skipping",
			zap.String("booking_id", event.BookingID.String()),
		)
		return nil
	}

	track := trackingDomain.NewTripTrack(event.BookingID, event.RunnerID)

	if err := s.repo.Save(ctx, track); err != nil {
		s.logger.Error("failed to save trip track", zap.Error(err))
		return fmt.Errorf("failed to save trip track: %w", err)
	}

	// Publish TrackingStartedEvent.
	startedEvt := events.TrackingStartedEvent{
		TrackID:    track.ID(),
		BookingID:  track.BookingID(),
		RunnerID:   track.RunnerID(),
		StartedAt:  track.StartedAt(),
		OccurredAt: time.Now().UTC(),
	}
	cloudEvt, err := kafka.NewCloudEvent("service-tracking", events.TrackingStarted, startedEvt)
	if err != nil {
		s.logger.Error("failed to create cloud event", zap.Error(err))
	} else if err := s.producer.PublishEvent(ctx, events.TopicTrackingEvents, cloudEvt); err != nil {
		s.logger.Error("failed to publish tracking started event", zap.Error(err))
	}

	s.logger.Info("trip tracking started",
		zap.String("track_id", track.ID().String()),
		zap.String("booking_id", track.BookingID().String()),
	)
	return nil
}

// HandleRunnerLocationUpdate adds a waypoint and broadcasts the update via WebSocket.
func (s *TrackingService) HandleRunnerLocationUpdate(ctx context.Context, event events.RunnerLocationUpdateEvent) error {
	// Find the active track for this runner.
	track, err := s.repo.FindActiveByRunnerID(ctx, event.RunnerID)
	if err != nil {
		// No active tracking for this runner; ignore the location update.
		s.logger.Debug("no active tracking for runner, ignoring location update",
			zap.String("runner_id", event.RunnerID.String()),
		)
		return nil
	}

	// Add waypoint.
	waypoint, err := trackingDomain.NewWaypoint(
		event.Latitude,
		event.Longitude,
		event.Speed,
		event.Heading,
		event.Timestamp,
	)
	if err != nil {
		s.logger.Warn("invalid waypoint data, skipping", zap.Error(err))
		return nil
	}

	if err := s.repo.AddWaypoint(ctx, track.ID(), waypoint); err != nil {
		s.logger.Error("failed to add waypoint", zap.Error(err))
		return fmt.Errorf("failed to add waypoint: %w", err)
	}

	// Broadcast via WebSocket hub.
	update := &ws.TrackingUpdate{
		BookingID: track.BookingID(),
		RunnerID:  track.RunnerID(),
		Latitude:  event.Latitude,
		Longitude: event.Longitude,
		Speed:     event.Speed,
		Heading:   event.Heading,
		Timestamp: event.Timestamp,
	}
	s.hub.Broadcast(update)

	// Publish TrackingUpdatedEvent.
	updatedEvt := events.TrackingUpdatedEvent{
		TrackID:    track.ID(),
		BookingID:  track.BookingID(),
		RunnerID:   track.RunnerID(),
		Latitude:   event.Latitude,
		Longitude:  event.Longitude,
		Speed:      event.Speed,
		OccurredAt: time.Now().UTC(),
	}
	cloudEvt, err := kafka.NewCloudEvent("service-tracking", events.TrackingUpdated, updatedEvt)
	if err != nil {
		s.logger.Error("failed to create cloud event", zap.Error(err))
	} else if err := s.producer.PublishEvent(ctx, events.TopicTrackingEvents, cloudEvt); err != nil {
		s.logger.Error("failed to publish tracking updated event", zap.Error(err))
	}

	return nil
}

// HandleDeliveryConfirmed completes the trip tracking when the delivery is confirmed.
func (s *TrackingService) HandleDeliveryConfirmed(ctx context.Context, event events.DeliveryConfirmedEvent) error {
	s.logger.Info("handling delivery confirmed event",
		zap.String("booking_id", event.BookingID.String()),
	)

	track, err := s.repo.FindByBookingID(ctx, event.BookingID)
	if err != nil {
		s.logger.Error("tracking not found for booking", zap.Error(err))
		return fmt.Errorf("tracking not found for booking %s: %w", event.BookingID.String(), err)
	}

	if !track.IsActive() {
		s.logger.Warn("tracking already completed or cancelled",
			zap.String("booking_id", event.BookingID.String()),
			zap.String("status", string(track.Status())),
		)
		return nil
	}

	// Calculate total distance from waypoints.
	waypoints, err := s.repo.GetWaypoints(ctx, track.ID())
	if err != nil {
		s.logger.Warn("failed to get waypoints for distance calculation", zap.Error(err))
	}
	totalDistance := calculateTotalDistance(waypoints)

	if err := track.Complete(totalDistance); err != nil {
		return fmt.Errorf("failed to complete tracking: %w", err)
	}

	if err := s.repo.Update(ctx, track); err != nil {
		return fmt.Errorf("failed to update tracking: %w", err)
	}

	// Publish TrackingCompletedEvent.
	completedEvt := events.TrackingCompletedEvent{
		TrackID:       track.ID(),
		BookingID:     track.BookingID(),
		RunnerID:      track.RunnerID(),
		TotalDistance: totalDistance,
		CompletedAt:  *track.CompletedAt(),
		OccurredAt:   time.Now().UTC(),
	}
	cloudEvt, err := kafka.NewCloudEvent("service-tracking", events.TrackingCompleted, completedEvt)
	if err != nil {
		s.logger.Error("failed to create cloud event", zap.Error(err))
	} else if err := s.producer.PublishEvent(ctx, events.TopicTrackingEvents, cloudEvt); err != nil {
		s.logger.Error("failed to publish tracking completed event", zap.Error(err))
	}

	s.logger.Info("trip tracking completed",
		zap.String("track_id", track.ID().String()),
		zap.String("booking_id", track.BookingID().String()),
		zap.Float64("total_distance_km", totalDistance),
	)
	return nil
}

// GetTracking returns the tracking data for a booking.
func (s *TrackingService) GetTracking(ctx context.Context, bookingID uuid.UUID) (*TrackingDTO, error) {
	track, err := s.repo.FindByBookingID(ctx, bookingID)
	if err != nil {
		return nil, domain.NewNotFoundError("tracking", bookingID.String())
	}

	waypoints, err := s.repo.GetWaypoints(ctx, track.ID())
	if err != nil {
		s.logger.Warn("failed to load waypoints", zap.Error(err))
		waypoints = nil
	}

	waypointDTOs := make([]WaypointDTO, 0, len(waypoints))
	for _, wp := range waypoints {
		waypointDTOs = append(waypointDTOs, WaypointDTO{
			ID:         wp.ID,
			Latitude:   wp.Latitude,
			Longitude:  wp.Longitude,
			Speed:      wp.Speed,
			Heading:    wp.Heading,
			RecordedAt: wp.RecordedAt,
		})
	}

	result := &TrackingDTO{
		ID:              track.ID(),
		BookingID:       track.BookingID(),
		RunnerID:        track.RunnerID(),
		Status:          string(track.Status()),
		TotalDistanceKm: track.TotalDistanceKm(),
		StartedAt:       track.StartedAt(),
		CompletedAt:     track.CompletedAt(),
		Waypoints:       waypointDTOs,
	}

	return result, nil
}

// GetRouteGeoJSON returns the route as a GeoJSON string.
func (s *TrackingService) GetRouteGeoJSON(ctx context.Context, bookingID uuid.UUID) (string, error) {
	track, err := s.repo.FindByBookingID(ctx, bookingID)
	if err != nil {
		return "", domain.NewNotFoundError("tracking", bookingID.String())
	}

	geoJSON, err := s.repo.GetRouteAsGeoJSON(ctx, track.ID())
	if err != nil {
		return "", fmt.Errorf("failed to get route GeoJSON: %w", err)
	}

	return geoJSON, nil
}

// calculateTotalDistance computes the total distance from a sequence of waypoints
// using the Haversine formula.
func calculateTotalDistance(waypoints []trackingDomain.Waypoint) float64 {
	if len(waypoints) < 2 {
		return 0
	}

	var totalKm float64
	for i := 1; i < len(waypoints); i++ {
		totalKm += haversineKm(
			waypoints[i-1].Latitude, waypoints[i-1].Longitude,
			waypoints[i].Latitude, waypoints[i].Longitude,
		)
	}
	return math.Round(totalKm*1000) / 1000 // Round to 3 decimal places
}

// haversineKm calculates the great-circle distance in kilometers between two coordinates.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
