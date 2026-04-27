# macos-screentime

[![mac](https://github.com/nigel4321/macos-screentime/actions/workflows/mac.yml/badge.svg)](https://github.com/nigel4321/macos-screentime/actions/workflows/mac.yml)
[![backend](https://github.com/nigel4321/macos-screentime/actions/workflows/backend.yml/badge.svg)](https://github.com/nigel4321/macos-screentime/actions/workflows/backend.yml)


A native macOS agent that reads and enforces screen-time restrictions, paired with an Android dashboard for viewing usage and editing policy.

Two clients, one sync backend:

- **`mac-agent/`** — Swift / SwiftUI menubar app. Collects per-app usage, applies enforcement via Family Controls.
- **`android-app/`** — Kotlin / Jetpack Compose dashboard. Views usage, edits policy.
- **`backend/`** — Go + Postgres. The only shared state between the two clients.

## Documentation

- [`PROJECT.md`](PROJECT.md) — objective and ways of working.
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — component design, data model, delivery milestones, and rejected alternatives.
- [`ROADMAP.md`](ROADMAP.md) — the live checklist of work, organised by milestone.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — TDD expectations, branch and PR conventions.

## Status

Pre-Milestone 1. Repo scaffolding in progress. See `ROADMAP.md` for the current frontier.

## License

See [`LICENSE`](LICENSE).
