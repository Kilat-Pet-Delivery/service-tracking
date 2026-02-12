CREATE TABLE trip_tracks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id UUID UNIQUE NOT NULL,
    runner_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    total_distance_km DECIMAL(10,3) DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trip_tracks_booking ON trip_tracks(booking_id);
CREATE INDEX idx_trip_tracks_runner ON trip_tracks(runner_id);
CREATE INDEX idx_trip_tracks_status ON trip_tracks(status);
