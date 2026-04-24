# Architecture

This document defines a specific, buildable approach for the macOS Screen Time project described in `PROJECT.md`. It is opinionated by design: choices are made so each iteration can be delivered, tested, and shipped independently.

---

## 1. System overview

Three components, each independently deployable:

```
 ┌────────────────────┐      HTTPS / WSS       ┌──────────────────────┐
 │  macOS agent       │  ───────────────────►  │  Sync backend        │
 │  (Swift, SwiftUI)  │  ◄───────────────────  │  (Go, Postgres)      │
 └────────────────────┘                        └──────────────────────┘
                                                         ▲
                                                         │ HTTPS / WSS
                                                         ▼
                                               ┌──────────────────────┐
                                               │  Android dashboard   │
                                               │  (Kotlin, Compose)   │
                                               └──────────────────────┘
```

- **macOS agent** — owns the device. Reads usage, enforces restrictions, publishes telemetry, subscribes to policy changes.
- **Sync backend** — the only shared state. Holds per-device usage history and the current policy. Authenticates both clients. Never executes policy itself.
- **Android dashboard** — read/write UI for a human. Never talks to the Mac directly.

The Mac is the source of truth for *what happened*. The backend is the source of truth for *what should happen*. The Android app is a view + editor on the backend.

---

## 2. macOS agent

### 2.1 Responsibilities

1. Collect per-app foreground usage (app bundle id, start, end).
2. Apply a policy: per-app daily limit, per-category daily limit, scheduled downtime windows, hard block list.
3. Push usage deltas and current policy-enforcement state to the backend.
4. Receive policy updates and apply them within seconds.

### 2.2 Technology choices

- **Language**: Swift 5.9+, targeting macOS 14 (Sonoma) and newer. macOS 14 is the first release where `FamilyControls` / `ManagedSettings` / `DeviceActivity` have usable parity with iOS; earlier macOS would force us into private APIs.
- **UI**: SwiftUI for a menubar-resident app (`MenuBarExtra`). No Dock icon by default.
- **Persistence**: SQLite via GRDB. Avoid Core Data — GRDB is easier to test and gives us plain SQL for the aggregation queries the Android dashboard will want.
- **Networking**: `URLSession` + `async/await`. WebSocket via `URLSessionWebSocketTask` for policy push.
- **Packaging**: Standard `.app` bundle, signed + notarized. Distribute via direct download initially; Mac App Store later (sandboxing will require adjustments).

### 2.3 Screen Time integration — the real constraint

`FamilyControls` on macOS requires **Family Sharing** and a parent Apple ID authorization. This is a hard constraint from Apple and shapes the whole project:

- The agent must request `AuthorizationCenter.shared.requestAuthorization(for: .individual)` on first launch.
- Without auth, the agent runs in **observe-only** mode: it can collect foreground-app samples via `NSWorkspace.shared.notificationCenter` (`didActivateApplicationNotification`) but cannot enforce.
- With auth, enforcement uses `ManagedSettingsStore` (shield tokens) and `DeviceActivityCenter` schedules.

**Decision**: build observe-only first (Milestone 1). Do not gate early progress on Family Controls authorization UX. Add enforcement in Milestone 3.

### 2.4 Module layout

```
MacAgent/
├── Core/
│   ├── UsageCollector/       # NSWorkspace polling + DeviceActivity bridge
│   ├── PolicyEngine/         # pure: (usage, policy, clock) -> actions
│   ├── Enforcer/             # ManagedSettings adapter
│   ├── LocalStore/           # GRDB schema + DAOs
│   └── SyncClient/           # REST + WebSocket, retry/backoff
├── App/
│   ├── MenuBarScene.swift
│   └── OnboardingView.swift
└── Tests/
    ├── PolicyEngineTests/    # 80% of the test mass lives here
    ├── LocalStoreTests/
    └── SyncClientTests/      # against a local mock server
```

`PolicyEngine` is deliberately a pure Swift module with no Apple-framework imports — this is what makes TDD work. `UsageCollector` and `Enforcer` are thin adapters around OS APIs, tested with fakes at the module boundary.

### 2.5 Data model (local)

```sql
CREATE TABLE usage_event (
  id INTEGER PRIMARY KEY,
  bundle_id TEXT NOT NULL,
  started_at INTEGER NOT NULL,  -- unix seconds
  ended_at   INTEGER NOT NULL,
  synced_at  INTEGER             -- NULL until pushed
);
CREATE INDEX idx_usage_unsynced ON usage_event(synced_at) WHERE synced_at IS NULL;

CREATE TABLE policy (
  version INTEGER PRIMARY KEY,   -- monotonic, from server
  body_json TEXT NOT NULL,
  received_at INTEGER NOT NULL,
  applied_at INTEGER
);
```

Usage events are append-only. Sync is a pull-unsynced / push / mark-synced loop — safe to crash at any point.

---

## 3. Sync backend

### 3.1 Why a backend exists

The Mac and the phone are rarely on the same network, often not online at the same time, and the Android app must work without the Mac being reachable. A backend is the simplest correct answer. A P2P design (Bluetooth, local mDNS) is explicitly rejected for Milestone 1–3.

### 3.2 Technology choices

- **Language**: Go 1.22. Chosen for small deployable binary, good HTTP/WebSocket story, fast test suite (matters for TDD cycle time).
- **Storage**: Postgres 16. `usage_event` is partitioned by month.
- **Auth**: Sign in with Apple on macOS, Sign in with Google on Android, both exchanged server-side for a backend-issued JWT. One `account` row owns one or more `device` rows.
- **Hosting**: Fly.io or Railway for Milestone 1 — one region, one Postgres. Migrate to a real cloud only if we outgrow it.

### 3.3 API surface (v1)

| Method | Path | Caller | Purpose |
|---|---|---|---|
| `POST` | `/v1/devices/register` | Mac | Register a device, get a device token |
| `POST` | `/v1/usage:batchUpload` | Mac | Push a batch of usage events (idempotent by client-supplied id) |
| `GET`  | `/v1/policy/current` | Mac, Android | Fetch current policy for the device |
| `PUT`  | `/v1/policy` | Android | Replace current policy; server increments version |
| `GET`  | `/v1/usage:summary?from=&to=&groupBy=` | Android | Pre-aggregated usage for the dashboard |
| `WS`   | `/v1/policy/subscribe` | Mac | Server pushes new policy versions |

All endpoints versioned under `/v1/`. Breaking changes bump to `/v2/` — old clients keep working through one deprecation cycle.

---

## 4. Android dashboard

### 4.1 Responsibilities

1. Show daily/weekly usage per app and per category.
2. Edit the policy: per-app limits, downtime windows, block list.
3. Reflect policy-apply status from the Mac (e.g. "applied 4s ago", "offline since 10:12").

### 4.2 Technology choices

- **Language**: Kotlin 2.0, `minSdk` 26, `targetSdk` 34.
- **UI**: Jetpack Compose + Material 3. Single-activity architecture.
- **Architecture**: MVVM with `ViewModel` + `StateFlow`. No Fragments.
- **DI**: Hilt.
- **Networking**: Retrofit + OkHttp + kotlinx.serialization.
- **Local cache**: Room, read-through cache for usage summaries so the dashboard opens instantly offline.
- **Charts**: Vico (Compose-native, no WebView).

### 4.3 Module layout

```
android-app/
├── app/                      # Compose UI, navigation, DI wiring
├── core-domain/              # pure Kotlin: models, policy validation
├── core-data/                # Retrofit + Room + repositories
├── feature-dashboard/        # usage charts
├── feature-policy-editor/    # limits, schedules
└── feature-onboarding/       # Google sign-in, pairing to a Mac
```

`core-domain` has zero Android dependencies, which keeps the tests fast and lets us share validation logic conceptually with the Mac (we will not actually share code between Swift and Kotlin — the duplication cost is lower than the tooling cost).

---

## 5. Testing strategy

Per `PROJECT.md`, TDD is mandatory. The practical interpretation:

| Layer | Framework | What we test | Speed target |
|---|---|---|---|
| `PolicyEngine` (Swift) | XCTest | All policy transitions, time-boundary edge cases | <1s full suite |
| `LocalStore` (Swift) | XCTest + in-memory GRDB | Migration, idempotent upserts | <2s |
| `SyncClient` (Swift) | XCTest + Vapor test server | Retry, auth refresh, clock-skew | <5s |
| Backend handlers (Go) | `testing` + `httptest` | Every endpoint, authz rules | <3s |
| Backend DB layer (Go) | `testing` + `pgtestdb` (real Postgres) | Schema, partitions, complex queries | <15s |
| `core-domain` (Kotlin) | JUnit5 + Kotest | Policy validation, aggregations | <2s |
| `core-data` (Kotlin) | JUnit5 + MockWebServer + Room in-memory | Repository behavior | <5s |
| Compose UI | `ComposeTestRule` | Screens render given state, interactions dispatch intents | <10s |

**Not in Milestone 1–3**: end-to-end tests across all three components. They are high-value but slow to stabilize; add once a second contributor joins.

**Rule**: a PR that changes `PolicyEngine` or `core-domain` without adding or changing a test should be rejected by the reviewer by default. CI does not enforce coverage thresholds (they encourage gaming); reviewers enforce intent.

---

## 6. CI/CD

All workflows live in `.github/workflows/`.

### 6.1 `mac.yml`

- Trigger: `push` to any branch, `pull_request` to `main`.
- Runner: `macos-14`.
- Steps: `xcodebuild test` for `PolicyEngine`, `LocalStore`, `SyncClient` schemes; `swiftlint`; signed build artifact uploaded on `main`.

### 6.2 `android.yml`

- Trigger: same.
- Runner: `ubuntu-latest`.
- Steps: `./gradlew ktlintCheck detekt test`, then `./gradlew :app:assembleRelease` with secrets from GitHub Actions.

### 6.3 `backend.yml`

- Trigger: same, but path-filtered to `backend/**`.
- Runner: `ubuntu-latest` with a Postgres service container.
- Steps: `go vet`, `go test ./...`, `golangci-lint`.

### 6.4 `android-release.yml`

- Trigger: `push` tag matching `android-v*`.
- Runner: `ubuntu-latest`.
- Steps: Fastlane lane `internal` → uploads AAB to Play Store **internal testing track**.
- Secrets: `PLAY_STORE_JSON_KEY`, `ANDROID_KEYSTORE`, `ANDROID_KEYSTORE_PASSWORD`, `KEY_ALIAS`, `KEY_PASSWORD`.

### 6.5 `mac-release.yml`

- Trigger: `push` tag matching `mac-v*`.
- Runner: `macos-14`.
- Steps: build + codesign + `xcrun notarytool submit` + staple + upload `.dmg` to a GitHub Release.

---

## 7. Iterative delivery plan

Each milestone is a shippable unit. No milestone depends on a future one compiling.

### Milestone 1 — Observe-only Mac (2–3 weeks)

- Menubar app that logs foreground app usage to local SQLite.
- A tiny SwiftUI window showing today's top apps.
- Full XCTest suite for `PolicyEngine` (even though we don't enforce yet — the model is stable).
- `mac.yml` green on every push.
- **No backend, no Android yet.**

### Milestone 2 — Backend + Android read-only (3 weeks)

- Go backend with `register`, `batchUpload`, `usage:summary`.
- Mac pushes events.
- Android app signs in, pairs to a Mac's device id, shows today's usage.
- `backend.yml` + `android.yml` green.
- `android-release.yml` publishes to internal testing track.

### Milestone 3 — Policy (3 weeks)

- `PUT /v1/policy`, WebSocket subscribe, Mac enforcement via `ManagedSettings`.
- Android policy editor.
- Family Controls authorization flow on the Mac.

### Milestone 4 — Categories, schedules, polish (ongoing)

- Category definitions (synced from a server-owned list).
- Downtime windows.
- Notifications on Android when a limit is near.

---

## 8. Known risks and how we mitigate them

| Risk | Mitigation |
|---|---|
| Family Controls authorization is flaky or requires Family Sharing, blocking enforcement | Observe-only is a useful product on its own — ship it first. Keep enforcement behind a capability flag. |
| Apple rejects the app from notarization/App Store due to `ManagedSettings` usage | Distribute outside the App Store initially; engage with Apple Developer Support before MAS submission. |
| Backend cost / complexity creeps | Single-region Fly.io + Postgres until >100 active devices; no Kafka, no microservices. |
| Clock skew between Mac and server breaks usage windows | Server timestamps are authoritative for policy windows; Mac timestamps for raw events. |
| TDD discipline erodes as UI work dominates | Keep `PolicyEngine` / `core-domain` as pure modules; they naturally attract tests. |

---

## 9. Out of scope (explicitly)

- iOS app.
- Windows/Linux screen-time collection.
- Real-time screen mirroring or remote control.
- Multi-Mac-per-account support (planned for v2).
- Shared family accounts with child/parent roles (planned for v2).
- Offline Mac-to-phone pairing without a backend.

Saying no to these now is what makes Milestones 1–3 deliverable.
