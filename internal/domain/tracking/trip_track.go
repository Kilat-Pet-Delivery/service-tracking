package tracking

import (
	"fmt"
	"time"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	"github.com/google/uuid"
)

// TrackingStatus represents the state of a trip tracking session.
type TrackingStatus string

const (
	TrackingActive    TrackingStatus = "active"
	TrackingCompleted TrackingStatus = "completed"
	TrackingCancelled TrackingStatus = "cancelled"
)

// Waypoint represents a single GPS point recorded during a trip.
type Waypoint struct {
	ID         uuid.UUID
	Latitude   float64
	Longitude  float64
	Speed      float64   // km/h
	Heading    float64   // degrees
	RecordedAt time.Time
}

// NewWaypoint creates a validated Waypoint with a generated UUID.
func NewWaypoint(lat, lng, speed, heading float64, recordedAt time.Time) (Waypoint, error) {
	if lat < -90 || lat > 90 {
		return Waypoint{}, fmt.Errorf("latitude must be between -90 and 90, got %f", lat)
	}
	if lng < -180 || lng > 180 {
		return Waypoint{}, fmt.Errorf("longitude must be between -180 and 180, got %f", lng)
	}
	if speed < 0 {
		speed = 0
	}
	if heading < 0 || heading > 360 {
		heading = 0
	}
	return Waypoint{
		ID:         uuid.New(),
		Latitude:   lat,
		Longitude:  lng,
		Speed:      speed,
		Heading:    heading,
		RecordedAt: recordedAt,
	}, nil
}

// TripTrack is the aggregate root for GPS tracking of a single booking trip.
type TripTrack struct {
	id              uuid.UUID
	bookingID       uuid.UUID
	runnerID        uuid.UUID
	status          TrackingStatus
	totalDistanceKm float64
	startedAt       time.Time
	completedAt     *time.Time
	version         int64
	createdAt       time.Time
	updatedAt       time.Time
}

// NewTripTrack creates a new active TripTrack for a booking.
func NewTripTrack(bookingID, runnerID uuid.UUID) *TripTrack {
	now := time.Now().UTC()
	return &TripTrack{
		id:              uuid.New(),
		bookingID:       bookingID,
		runnerID:        runnerID,
		status:          TrackingActive,
		totalDistanceKm: 0,
		startedAt:       now,
		completedAt:     nil,
		version:         1,
		createdAt:       now,
		updatedAt:       now,
	}
}

// --- Getters ---

// ID returns the trip track's unique identifier.
func (t *TripTrack) ID() uuid.UUID { return t.id }

// BookingID returns the associated booking identifier.
func (t *TripTrack) BookingID() uuid.UUID { return t.bookingID }

// RunnerID returns the associated runner identifier.
func (t *TripTrack) RunnerID() uuid.UUID { return t.runnerID }

// Status returns the current tracking status.
func (t *TripTrack) Status() TrackingStatus { return t.status }

// TotalDistanceKm returns the total distance traveled in kilometers.
func (t *TripTrack) TotalDistanceKm() float64 { return t.totalDistanceKm }

// StartedAt returns when tracking began.
func (t *TripTrack) StartedAt() time.Time { return t.startedAt }

// CompletedAt returns when tracking ended (nil if still active).
func (t *TripTrack) CompletedAt() *time.Time { return t.completedAt }

// Version returns the version for optimistic locking.
func (t *TripTrack) Version() int64 { return t.version }

// CreatedAt returns when the record was created.
func (t *TripTrack) CreatedAt() time.Time { return t.createdAt }

// UpdatedAt returns when the record was last updated.
func (t *TripTrack) UpdatedAt() time.Time { return t.updatedAt }

// --- Behavior ---

// Complete transitions the trip track from active to completed.
func (t *TripTrack) Complete(totalDistanceKm float64) error {
	if t.status != TrackingActive {
		return domain.NewInvalidStateError(string(t.status), string(TrackingCompleted))
	}
	now := time.Now().UTC()
	t.status = TrackingCompleted
	t.totalDistanceKm = totalDistanceKm
	t.completedAt = &now
	t.updatedAt = now
	return nil
}

// Cancel transitions the trip track from active to cancelled.
func (t *TripTrack) Cancel() error {
	if t.status != TrackingActive {
		return domain.NewInvalidStateError(string(t.status), string(TrackingCancelled))
	}
	now := time.Now().UTC()
	t.status = TrackingCancelled
	t.updatedAt = now
	return nil
}

// IncrementVersion bumps the version for optimistic locking.
func (t *TripTrack) IncrementVersion() {
	t.version++
	t.updatedAt = time.Now().UTC()
}

// IsActive returns true if the trip track is currently active.
func (t *TripTrack) IsActive() bool {
	return t.status == TrackingActive
}

// --- Reconstruction from persistence ---

// Reconstruct creates a TripTrack from persisted data (used by repositories).
func Reconstruct(
	id, bookingID, runnerID uuid.UUID,
	status TrackingStatus,
	totalDistanceKm float64,
	startedAt time.Time,
	completedAt *time.Time,
	version int64,
	createdAt, updatedAt time.Time,
) *TripTrack {
	return &TripTrack{
		id:              id,
		bookingID:       bookingID,
		runnerID:        runnerID,
		status:          status,
		totalDistanceKm: totalDistanceKm,
		startedAt:       startedAt,
		completedAt:     completedAt,
		version:         version,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}
