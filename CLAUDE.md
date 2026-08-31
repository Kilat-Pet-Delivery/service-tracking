# Kilat Pet Delivery - service-tracking

Live trip tracking: GPS waypoints, the WebSocket hub the apps subscribe to, owner and runner chat, and public share links for a trip in progress.
Jira project **KPD** - GitHub `Kilat-Pet-Delivery/service-tracking` - stack **Go 1.24 - Gin - GORM - PostgreSQL - Kafka**. Global rules live in `~/.claude/`;
this file only adds what is specific here.

## Orient here first

- `.claude/memory/project_state.md` - **resume here** (`/continue` reads it, `/recap` rewrites it).
- `README.md` - how to run it. `CHANGELOG.md` - what changed.
- The workspace map: `~/Documents/kilat-pet-delivery/CLAUDE.md`.

## Commands

| Task | Command |
|---|---|
| install | `go mod download` |
| run | `go run ./cmd/server` (copy `.env.example` to `.env` first) |
| test | `go test ./...` |
| integration tests | none in this repo |
| lint | `gofmt -l . && go vet ./...` |
| build | `go build ./...` |
| migrate | `go run ./cmd/migrate` - applies `migrations/` and exits |

Needs the dev-infra stack: Postgres database `kilat_tracking`, Kafka on `localhost:9092` -> `cd ~/Documents/dev-infra; ./dev.ps1 up kilat`.

## Conventions that differ from the global rules

- **Ticket branches and PRs** - company repo, never commit on `main` (`branch-guard` enforces it).
- **One migration path.** `migrations/` owns the schema in every environment including development, and `cmd/server` applies it at startup. There is deliberately no GORM AutoMigrate branch - that is what let six services drift (KPD-56 through KPD-61).
- Protected paths (never edited in place, see `.claude/protected-paths.txt`): `migrations/*.sql`.

## Testing

No test files in any package. `go test ./...` has nothing to run.

## Where things are

- `cmd/server` - `cmd/migrate` - `internal/domain/{tracking,chat,share}` - the WebSocket hub is proxied by api-gateway at `/ws/tracking/:bookingId`

## Worth knowing

- chat_messages lives here, not in the empty service-chat scaffold. That is exactly what KPD-48 has to decide about.
