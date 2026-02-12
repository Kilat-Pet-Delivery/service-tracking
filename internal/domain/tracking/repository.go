package tracking

import (
	"context"

	"github.com/google/uuid"
)

// TripTrackRepository defines the persistence interface for TripTrack aggregates.
type TripTrackRepository interface {
	// FindByID retrieves a trip track by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (*TripTrack, error)

	// FindByBookingID retrieves a trip track by its associated booking identifier.
	FindByBookingID(ctx context.Context, bookingID uuid.UUID) (*TripTrack, error)

	// FindActiveByRunnerID retrieves the currently active trip track for a runner.
	FindActiveByRunnerID(ctx context.Context, runnerID uuid.UUID) (*TripTrack, error)

	// Save persists a new trip track.
	Save(ctx context.Context, track *TripTrack) error

	// Update persists changes to an existing trip track.
	Update(ctx context.Context, track *TripTrack) error

	// AddWaypoint records a new GPS waypoint for a trip track.
	AddWaypoint(ctx context.Context, trackID uuid.UUID, waypoint Waypoint) error

	// GetWaypoints retrieves all waypoints for a trip track ordered by time.
	GetWaypoints(ctx context.Context, trackID uuid.UUID) ([]Waypoint, error)

	// GetRouteAsGeoJSON returns the trip route as a GeoJSON LineString.
	GetRouteAsGeoJSON(ctx context.Context, trackID uuid.UUID) (string, error)
}
