# Changelog

All notable changes to service-tracking are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `cmd/migrate`: standalone command that applies the golang-migrate files in
  `migrations/` and exits. There was previously no way to apply the SQL schema
  without booting the service outside `APP_ENV=development`. (KPD-4)
- `CHANGELOG.md`: this file. Partially advances KPD-52.

### Changed

- README: the run block now points at the shared dev-infra stack and documents the
  two migration modes, including that `chat_messages` and `shared_trips` are
  dev-only today (KPD-61).

### Notes

- Migrations applied to `kilat_tracking` on the shared dev-infra stack as part of KPD-4.
