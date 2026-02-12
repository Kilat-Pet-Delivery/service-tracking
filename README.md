# service-tracking

Real-time GPS tracking service with WebSocket broadcasting and route visualization.

## Description

This service provides real-time GPS tracking for active pet transport bookings. It maintains trip tracks with GPS waypoints, broadcasts location updates to clients via WebSocket connections, and generates GeoJSON route exports using PostGIS for visualization.

## Features

- Trip track aggregate management
- GPS waypoint collection and storage
- WebSocket hub with room-based broadcasting
- Real-time location updates by booking ID
- GeoJSON route export for mapping
- PostGIS spatial queries
- Kafka event-driven architecture

## API Endpoints

| Method | Endpoint                       | Access | Description                    |
|--------|--------------------------------|--------|--------------------------------|
| GET    | /api/v1/tracking/:bookingId    | Auth   | Get trip track details         |
| GET    | /api/v1/tracking/:bookingId/route | Auth | Export route as GeoJSON     |
| WS     | /ws/tracking/:bookingId        | Auth   | WebSocket for live updates     |

## WebSocket Protocol

Clients connect to `/ws/tracking/:bookingId` with JWT authentication to receive real-time location updates:

```json
{
  "type": "location_update",
  "booking_id": "uuid",
  "latitude": 37.7749,
  "longitude": -122.4194,
  "timestamp": "2026-02-06T10:30:00Z"
}
```

## Kafka Integration

**Events Consumed:**
- **booking.accepted**: Creates new trip track
- **runner.location_update**: Adds waypoint and broadcasts to WebSocket clients
- **booking.delivery_confirmed**: Completes trip track

## Configuration

The service requires the following environment variables:

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=tracking_db
SERVICE_PORT=8005
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_PREFIX=kilat-pet-runner
```

## Tech Stack

- **Language**: Go 1.24
- **Web Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL with PostGIS extension
- **Message Queue**: Kafka (shopify/sarama)
- **WebSocket**: gorilla/websocket

## Running the Service

```bash
# Install dependencies
go mod download

# Enable PostGIS extension
psql -d tracking_db -c "CREATE EXTENSION IF NOT EXISTS postgis;"

# Run migrations
go run cmd/migrate/main.go

# Start the service
go run cmd/server/main.go
```

The service will start on port 8005.

## Database Schema

- **tracks**: Trip track aggregates linked to bookings
- **waypoints**: GPS coordinates with PostGIS geometry type
- **route_metadata**: Distance, duration, and route statistics

## WebSocket Hub

The service maintains an in-memory WebSocket hub with room-based broadcasting:
- Each booking ID represents a room
- Multiple clients can subscribe to the same booking
- Location updates are broadcast to all subscribers in real-time
- Automatic cleanup on client disconnect
