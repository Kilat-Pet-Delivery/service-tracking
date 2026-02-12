package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	trackingDomain "github.com/Kilat-Pet-Delivery/service-tracking/internal/domain/tracking"
)

// TripTrackModel is the GORM model for the trip_tracks table.
type TripTrackModel struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	BookingID       uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"`
	RunnerID        uuid.UUID  `gorm:"type:uuid;index;not null"`
	Status          string     `gorm:"type:varchar(20);not null;default:'active';index"`
	TotalDistanceKm float64   `gorm:"type:decimal(10,3);default:0"`
	StartedAt       time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	CompletedAt     *time.Time `gorm:"type:timestamptz"`
	Version         int64      `gorm:"not null;default:1"`
	CreatedAt       time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt       time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName overrides the default table name.
func (TripTrackModel) TableName() string {
	return "trip_tracks"
}

// WaypointModel is the GORM model for the waypoints table.
type WaypointModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TripTrackID uuid.UUID `gorm:"type:uuid;not null;index"`
	Latitude    float64   `gorm:"type:double precision;not null"`
	Longitude   float64   `gorm:"type:double precision;not null"`
	Speed       float64   `gorm:"type:decimal(6,2)"`
	Heading     float64   `gorm:"type:decimal(5,2)"`
	RecordedAt  time.Time `gorm:"type:timestamptz;not null"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName overrides the default table name.
func (WaypointModel) TableName() string {
	return "waypoints"
}

// GORMTripTrackRepository implements TripTrackRepository using GORM.
type GORMTripTrackRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewGORMTripTrackRepository creates a new GORM-based repository.
func NewGORMTripTrackRepository(db *gorm.DB, logger *zap.Logger) *GORMTripTrackRepository {
	return &GORMTripTrackRepository{
		db:     db,
		logger: logger,
	}
}

// FindByID retrieves a trip track by its unique identifier.
func (r *GORMTripTrackRepository) FindByID(ctx context.Context, id uuid.UUID) (*trackingDomain.TripTrack, error) {
	var model TripTrackModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find trip track by id: %w", err)
	}
	return toDomain(&model), nil
}

// FindByBookingID retrieves a trip track by its associated booking identifier.
func (r *GORMTripTrackRepository) FindByBookingID(ctx context.Context, bookingID uuid.UUID) (*trackingDomain.TripTrack, error) {
	var model TripTrackModel
	if err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find trip track by booking id: %w", err)
	}
	return toDomain(&model), nil
}

// FindActiveByRunnerID retrieves the currently active trip track for a runner.
func (r *GORMTripTrackRepository) FindActiveByRunnerID(ctx context.Context, runnerID uuid.UUID) (*trackingDomain.TripTrack, error) {
	var model TripTrackModel
	if err := r.db.WithContext(ctx).
		Where("runner_id = ? AND status = ?", runnerID, string(trackingDomain.TrackingActive)).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find active trip track for runner: %w", err)
	}
	return toDomain(&model), nil
}

// Save persists a new trip track.
func (r *GORMTripTrackRepository) Save(ctx context.Context, track *trackingDomain.TripTrack) error {
	model := toModel(track)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to save trip track: %w", err)
	}
	return nil
}

// Update persists changes to an existing trip track.
func (r *GORMTripTrackRepository) Update(ctx context.Context, track *trackingDomain.TripTrack) error {
	model := toModel(track)
	result := r.db.WithContext(ctx).
		Where("id = ? AND version = ?", model.ID, model.Version-1).
		Save(model)

	if result.Error != nil {
		return fmt.Errorf("failed to update trip track: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrOptimisticLock
	}
	return nil
}

// AddWaypoint records a new GPS waypoint for a trip track.
func (r *GORMTripTrackRepository) AddWaypoint(ctx context.Context, trackID uuid.UUID, waypoint trackingDomain.Waypoint) error {
	model := &WaypointModel{
		ID:          waypoint.ID,
		TripTrackID: trackID,
		Latitude:    waypoint.Latitude,
		Longitude:   waypoint.Longitude,
		Speed:       waypoint.Speed,
		Heading:     waypoint.Heading,
		RecordedAt:  waypoint.RecordedAt,
		CreatedAt:   time.Now().UTC(),
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to add waypoint: %w", err)
	}
	return nil
}

// GetWaypoints retrieves all waypoints for a trip track ordered by time.
func (r *GORMTripTrackRepository) GetWaypoints(ctx context.Context, trackID uuid.UUID) ([]trackingDomain.Waypoint, error) {
	var models []WaypointModel
	if err := r.db.WithContext(ctx).
		Where("trip_track_id = ?", trackID).
		Order("recorded_at ASC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get waypoints: %w", err)
	}

	waypoints := make([]trackingDomain.Waypoint, len(models))
	for i, m := range models {
		waypoints[i] = trackingDomain.Waypoint{
			ID:         m.ID,
			Latitude:   m.Latitude,
			Longitude:  m.Longitude,
			Speed:      m.Speed,
			Heading:    m.Heading,
			RecordedAt: m.RecordedAt,
		}
	}
	return waypoints, nil
}

// GetRouteAsGeoJSON returns the trip route as a GeoJSON LineString.
// Attempts PostGIS ST_MakeLine first; falls back to manual GeoJSON construction.
func (r *GORMTripTrackRepository) GetRouteAsGeoJSON(ctx context.Context, trackID uuid.UUID) (string, error) {
	// Try PostGIS approach first.
	var geoJSON string
	err := r.db.WithContext(ctx).Raw(`
		SELECT ST_AsGeoJSON(ST_MakeLine(
			ST_MakePoint(w.longitude, w.latitude) ORDER BY w.recorded_at
		)) FROM waypoints w WHERE w.trip_track_id = ?
	`, trackID).Scan(&geoJSON).Error

	if err == nil && geoJSON != "" {
		return geoJSON, nil
	}

	// Fallback: build GeoJSON manually from waypoints.
	r.logger.Debug("PostGIS not available or no data, building GeoJSON manually",
		zap.String("track_id", trackID.String()),
	)

	waypoints, err := r.GetWaypoints(ctx, trackID)
	if err != nil {
		return "", err
	}

	return buildGeoJSONLineString(waypoints)
}

// buildGeoJSONLineString constructs a GeoJSON LineString from waypoints.
func buildGeoJSONLineString(waypoints []trackingDomain.Waypoint) (string, error) {
	if len(waypoints) == 0 {
		return `{"type":"LineString","coordinates":[]}`, nil
	}

	coordinates := make([][]float64, len(waypoints))
	for i, wp := range waypoints {
		coordinates[i] = []float64{wp.Longitude, wp.Latitude}
	}

	geoJSON := map[string]interface{}{
		"type":        "LineString",
		"coordinates": coordinates,
	}

	data, err := json.Marshal(geoJSON)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GeoJSON: %w", err)
	}
	return string(data), nil
}

// toDomain converts a GORM model to a domain TripTrack.
func toDomain(model *TripTrackModel) *trackingDomain.TripTrack {
	return trackingDomain.Reconstruct(
		model.ID,
		model.BookingID,
		model.RunnerID,
		trackingDomain.TrackingStatus(model.Status),
		model.TotalDistanceKm,
		model.StartedAt,
		model.CompletedAt,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

// toModel converts a domain TripTrack to a GORM model.
func toModel(track *trackingDomain.TripTrack) *TripTrackModel {
	return &TripTrackModel{
		ID:              track.ID(),
		BookingID:       track.BookingID(),
		RunnerID:        track.RunnerID(),
		Status:          string(track.Status()),
		TotalDistanceKm: track.TotalDistanceKm(),
		StartedAt:       track.StartedAt(),
		CompletedAt:     track.CompletedAt(),
		Version:         track.Version(),
		CreatedAt:       track.CreatedAt(),
		UpdatedAt:       track.UpdatedAt(),
	}
}
