-- 004_create_shared_trips.sql
-- SharedTripModel was only ever created by the development AutoMigrate branch in
-- cmd/server/main.go, so "shared_trips" did not exist in any environment that
-- runs the SQL migrations instead. See KPD-61.
--
-- share_token is the public, unguessable half of a share link, so it is unique
-- and indexed -- every anonymous lookup goes through it.

CREATE TABLE IF NOT EXISTS shared_trips (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id  UUID        NOT NULL,
    share_token VARCHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shared_trips_booking ON shared_trips(booking_id);
-- Expiry sweeps scan this.
CREATE INDEX IF NOT EXISTS idx_shared_trips_expires_at ON shared_trips(expires_at);
