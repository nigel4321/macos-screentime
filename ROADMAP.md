# Roadmap

Derived from `ARCHITECTURE.md`. Each item is a discrete, ticking unit of work — roughly one commit or one short session. Milestones are shippable; items within a milestone should generally land in order. Stop and check in at each milestone boundary.

Legend: `[ ]` todo · `[x]` done · `[~]` in progress

---

## Milestone 0 — Repo scaffolding

- [x] Decide monorepo layout: `mac-agent/`, `android-app/`, `backend/`, `.github/workflows/`, top-level docs
- [x] Add root `.gitignore` covering Swift/Xcode, Kotlin/Gradle, Go, macOS, IDEs
- [x] Expand `README.md` with project summary + pointers to `PROJECT.md`, `ARCHITECTURE.md`, `ROADMAP.md`
- [x] Add `CONTRIBUTING.md` with TDD expectation and branch/PR conventions
- [x] Add issue + PR templates under `.github/`

---

## Milestone 1 — Observe-only Mac agent

### 1.1 Xcode project setup
- [x] Create `mac-agent/` directory
- [ ] Create Xcode workspace `MacAgent.xcworkspace` *(mac-only)*
- [ ] Create app target `MacAgent` (SwiftUI, macOS) *(mac-only)*
- [x] Set deployment target to macOS 14.0 *(in `Package.swift`)*
- [x] Set Swift language version to 5.9 *(`swift-tools-version:5.9`)*
- [x] Create Swift Package with `Package.swift` at `mac-agent/Package.swift`
- [ ] Add package as a local dependency of the app target *(mac-only, after app target exists)*
- [ ] Configure local development code signing *(mac-only)*
- [x] Decide `.xcodeproj` tracking policy — tracked; user state excluded via root `.gitignore` (documented in `mac-agent/README.md`)

### 1.2 SwiftLint
- [x] Add SwiftLint as a Swift Package plugin *(dependency added to `Package.swift`; attached per-target in §1.3+)*
- [x] Add `.swiftlint.yml` with chosen rule set
- [ ] Add build-phase script to run SwiftLint on every build *(mac-only — Xcode build phase; SPM plugin covers non-Xcode builds)*
- [ ] Verify lint fails a deliberately bad commit *(mac-only — requires `swift build` with the plugin, and the first target to exist)*

### 1.3 PolicyEngine — scaffolding
- [x] Add `PolicyEngine` product to `Core` package
- [x] Define `BundleID` value type
- [x] Define `UsageEvent` struct (bundleId, start, end)
- [x] Define `Policy` struct with nested `AppLimit`, `DowntimeWindow`
- [x] Define `EnforcementAction` enum (`shield`, `clear`)
- [x] Define `PolicyVersion` type
- [x] Add `PolicyEngineTests` target

> **Sub-sections §1.4 – §1.10 require a macOS environment** to exercise meaningfully (TDD cycles need `swift test`; SwiftUI / AppKit code requires the macOS SDK). They are left unticked and will be resumed on the Mac.

### 1.4 PolicyEngine — no-policy case (TDD) *(awaits macOS)*
- [ ] RED: `evaluate` returns empty actions when policy is nil
- [ ] GREEN: minimal `evaluate` implementation
- [ ] RED: `evaluate` returns empty when policy has no rules
- [ ] GREEN: extend implementation

### 1.5 PolicyEngine — per-app daily limits (TDD) *(awaits macOS)*
- [ ] RED: limit not crossed → no shield
- [ ] GREEN: accumulate today's usage for bundleId, compare to limit
- [ ] RED: limit exactly crossed → shield emitted
- [ ] RED: yesterday's data alone does not trip today's limit
- [ ] GREEN: scope accumulation to "today" via injected clock
- [ ] RED: multiple apps with independent limits
- [ ] GREEN: handle N apps

### 1.6 PolicyEngine — downtime windows (TDD) *(awaits macOS)*
- [ ] RED: outside window → no shield
- [ ] RED: inside window → shield all apps on block list
- [ ] RED: window crossing midnight
- [ ] RED: DST spring-forward boundary
- [ ] RED: DST fall-back boundary
- [ ] GREEN: implement window evaluation with injected `Clock` + `TimeZone`
- [ ] Property test: window-active ⇔ now ∈ [start, end]

### 1.7 LocalStore *(awaits macOS)*
- [ ] Add GRDB dependency to `Core`
- [ ] Add `LocalStore` product
- [ ] Schema v1 migration: `usage_event`, `policy`
- [ ] Migration runner with version tracking
- [ ] Migration test: fresh DB → v1
- [ ] Migration test: re-run is idempotent
- [ ] DAO: insert usage event (idempotent on client id)
- [ ] DAO: query unsynced events
- [ ] DAO: mark events synced
- [ ] DAO: read/write current policy
- [ ] DAO tests with in-memory GRDB

### 1.8 UsageCollector *(awaits macOS)*
- [ ] Add `UsageCollector` product
- [ ] Abstract `WorkspaceSource` protocol
- [ ] Real implementation wrapping `NSWorkspace.didActivateApplicationNotification`
- [ ] Fake implementation for tests
- [ ] Translate activation stream to closed `(bundleId, start, end)` events
- [ ] Handle system sleep → close open event
- [ ] Handle system wake → resume on active app
- [ ] Handle screen lock → close open event
- [ ] Unit tests for each transition

### 1.9 App wiring *(mac-only)*
- [ ] Root `MenuBarExtra` scene
- [ ] Hide Dock icon via `LSUIElement` in Info.plist
- [ ] Dependency container wiring UsageCollector → LocalStore
- [ ] Start collection on app launch
- [ ] Graceful shutdown on quit (flush open event)

### 1.10 Today view *(mac-only)*
- [ ] `TodayViewModel` querying LocalStore for today's aggregated usage
- [ ] Top-5 apps list UI in the menu popover
- [ ] Live updates as events land
- [ ] Empty state
- [ ] First-launch onboarding screen (observe-only copy)

### 1.11 CI
- [x] `.github/workflows/mac.yml` on `macos-14`
- [x] Run SwiftLint *(via SPM build-tool plugin during `swift build`)*
- [ ] Run `xcodebuild test` for all Core test targets *(mac-only — requires Xcode app target from §1.1)*
- [x] Cache SPM packages
- [x] Status badge in README
- [ ] Green build on `main` *(awaits first push from a Mac; §1.4+ need verifiable tests first)*

---

## Milestone 2 — Backend + Android read-only

### 2.1 Backend: module scaffolding
- [ ] Init `backend/` Go module
- [ ] Layout: `cmd/server`, `internal/{api,db,auth,model}`, `migrations`
- [ ] HTTP server with chi router
- [ ] `/healthz` endpoint
- [ ] Env-var config loader
- [ ] Structured logging via `log/slog`
- [ ] Graceful shutdown on SIGTERM
- [ ] `golangci-lint` config
- [ ] `Makefile` with run/test/lint targets

### 2.2 Backend: Postgres + migrations
- [ ] Choose migration tool (goose)
- [ ] Add `pgtestdb` for integration tests
- [ ] Migration 001: `account`
- [ ] Migration 002: `device`
- [ ] Migration 003: `usage_event` parent + monthly partitioning
- [ ] Migration 004: `policy`
- [ ] Helper to create next month's partition
- [ ] Connection pool with `pgxpool`
- [ ] Include DB ping in `/healthz`

### 2.3 Backend: auth
- [ ] JWT signing key management (env var, rotatable)
- [ ] Apple identity-token verifier (JWKS fetch + cache)
- [ ] Google identity-token verifier (JWKS fetch + cache)
- [ ] `POST /v1/auth/apple` → backend JWT
- [ ] `POST /v1/auth/google` → backend JWT
- [ ] `Authenticator` middleware: verify JWT, load account
- [ ] `DeviceContext` middleware: resolve device from token
- [ ] Tests for each verifier with signed fixtures
- [ ] Tests for authz denial paths

### 2.4 Backend: `POST /v1/devices/register`
- [ ] Route + request/response types
- [ ] Insert device row keyed to account
- [ ] Return device id + device token
- [ ] Idempotent on (account, device fingerprint)
- [ ] Handler tests

### 2.5 Backend: `POST /v1/usage:batchUpload`
- [ ] Route handler accepting batch JSON
- [ ] Idempotency on client-supplied event id
- [ ] Validate device owns account
- [ ] Insert into partitioned table
- [ ] Return per-event accept/reject
- [ ] Handler tests including duplicate suppression

### 2.6 Backend: `GET /v1/usage:summary`
- [ ] Accept query params `from`, `to`, `groupBy`
- [ ] SQL: sum durations grouped by bundleId and/or day
- [ ] Return JSON aggregates
- [ ] Enforce account ownership of devices in query
- [ ] Handler tests

### 2.7 Backend: `GET /v1/policy/current` (stub)
- [ ] Route handler returning empty policy v0
- [ ] Handler test

### 2.8 Backend CI
- [ ] `.github/workflows/backend.yml` with path filter `backend/**`
- [ ] Postgres service container
- [ ] `go vet`, `go test`, `golangci-lint`
- [ ] Cache Go modules

### 2.9 Backend deploy
- [ ] Dockerfile (multi-stage, distroless base)
- [ ] Fly.io (or Railway) app provisioned
- [ ] Postgres provisioned
- [ ] Secrets set (JWT key, DB URL)
- [ ] First deploy
- [ ] Smoke-test script hitting `/healthz`

### 2.10 Mac agent: SyncClient
- [ ] Add `SyncClient` product to Core
- [ ] Device-registration flow, token stored in Keychain
- [ ] `BatchUploader` reading unsynced events
- [ ] Exponential backoff with jitter
- [ ] Test against a local Vapor mock server
- [ ] Wire into app lifecycle (periodic flush + on-quit flush)

### 2.11 Android: project setup
- [ ] Init `android-app/` Gradle project
- [ ] Kotlin 2.0, AGP latest, `minSdk 26`, `targetSdk 34`
- [ ] Compose BOM, Material 3
- [ ] Hilt plugin + dependencies
- [ ] Retrofit, OkHttp, kotlinx.serialization
- [ ] Room
- [ ] Vico charts
- [ ] ktlint plugin
- [ ] detekt plugin
- [ ] Version catalog in `libs.versions.toml`

### 2.12 Android: module split
- [ ] `:app` (Compose entry, navigation, DI)
- [ ] `:core-domain` (pure Kotlin)
- [ ] `:core-data` (Retrofit + Room + repositories)
- [ ] `:feature-onboarding`
- [ ] `:feature-dashboard`
- [ ] Wire Hilt across modules

### 2.13 Android: network layer
- [ ] Retrofit service interface for v1 endpoints
- [ ] Auth interceptor adding JWT
- [ ] 401 handler triggering token refresh
- [ ] JSON serializers for shared DTOs
- [ ] Repository classes in `:core-data`
- [ ] Unit tests against MockWebServer

### 2.14 Android: local cache
- [ ] Room DB in `:core-data`
- [ ] `UsageSummaryEntity` table
- [ ] DAOs
- [ ] Cache-first, network-refresh repository pattern
- [ ] Cache-invalidation rules

### 2.15 Android: onboarding
- [ ] Google Sign-In via Credential Manager
- [ ] Exchange Google ID token for backend JWT
- [ ] Store JWT in `EncryptedSharedPreferences`
- [ ] Account `StateFlow`
- [ ] Sign-out flow
- [ ] Compose screen with loading/error states

### 2.16 Android: device pairing
- [ ] Fetch registered devices for account
- [ ] UI to pick primary device
- [ ] Persist selected device id
- [ ] Zero-device state with copy: "install the Mac agent first"

### 2.17 Android: dashboard — today
- [ ] `TodayViewModel` fetching per-app summary
- [ ] Vico bar chart
- [ ] Loading skeleton
- [ ] Error + retry state
- [ ] Empty state
- [ ] Compose UI tests for each state

### 2.18 Android: dashboard — week
- [ ] `WeekViewModel` fetching per-day summary
- [ ] Vico stacked bars
- [ ] Tab navigation between Today and Week
- [ ] Compose UI tests

### 2.19 Android CI
- [ ] `.github/workflows/android.yml`
- [ ] ktlint, detekt
- [ ] Unit tests in `:core-domain`, `:core-data`
- [ ] Compose UI tests (or document deferral)
- [ ] Assemble debug + release
- [ ] Gradle cache

### 2.20 Android release via Fastlane
- [ ] `fastlane/` under `android-app/`
- [ ] `Fastfile` with `internal` lane
- [ ] Play Store service-account JSON (user-provided secret)
- [ ] Upload keystore secret
- [ ] `.github/workflows/android-release.yml` on `android-v*` tags
- [ ] First successful internal-track upload
- [ ] Verify install on a tester device

---

## Milestone 3 — Policy enforcement

### 3.1 Backend: policy mutation
- [ ] `PUT /v1/policy` handler
- [ ] Version increment on write
- [ ] Optimistic concurrency via `If-Match: version`
- [ ] Server-side policy shape validation
- [ ] Authz: only account owner writes
- [ ] Handler tests

### 3.2 Backend: WebSocket policy subscribe
- [ ] `WS /v1/policy/subscribe`
- [ ] Auth handshake on first message
- [ ] In-memory pub/sub registry
- [ ] Emit new version on PUT commit
- [ ] Heartbeat + idle timeout
- [ ] Document reconnection semantics

### 3.3 Mac: Family Controls authorization
- [ ] Entitlement + provisioning profile updates
- [ ] `AuthorizationCenter.shared.requestAuthorization(for: .individual)` flow
- [ ] Onboarding screen explaining the permission
- [ ] Denial path (remain observe-only)
- [ ] Surface auth status in menubar

### 3.4 Mac: Enforcer module
- [ ] Add `Enforcer` product to Core
- [ ] `ManagedSettingsStore` wrapper
- [ ] Apply `.shield(bundleId)` by adding token
- [ ] Apply `.clear(bundleId)` by removing token
- [ ] Reconcile current vs desired set (idempotent)
- [ ] Unit tests with a fake store

### 3.5 Mac: policy subscriber
- [ ] `PolicySubscriber` using `URLSessionWebSocketTask`
- [ ] Reconnect with backoff
- [ ] On new version: write to LocalStore, re-run PolicyEngine, apply via Enforcer
- [ ] Surface "applied vX at HH:MM" in menubar
- [ ] Tests against a local WebSocket echo server

### 3.6 Android: policy editor — app limits
- [ ] `:feature-policy-editor` module
- [ ] Fetch current policy
- [ ] Per-app daily-limit editor
- [ ] Add/remove apps
- [ ] Validation in `:core-domain` (unit-tested)
- [ ] Save with optimistic UI + server reconciliation
- [ ] Compose UI tests

### 3.7 Android: policy editor — downtime
- [ ] Window editor (start, end, days of week)
- [ ] Multiple windows per day
- [ ] Validation in `:core-domain`
- [ ] Compose UI tests

### 3.8 Android: policy editor — block list
- [ ] Hard-block editor
- [ ] Confirmation on destructive add
- [ ] Unit tests

### 3.9 Mac release pipeline
- [ ] Developer ID Application certificate provisioned
- [ ] Codesign in CI using secret-stored `.p12`
- [ ] `xcrun notarytool submit --wait`
- [ ] `xcrun stapler staple`
- [ ] `.dmg` via `create-dmg`
- [ ] `.github/workflows/mac-release.yml` on `mac-v*` tags
- [ ] Attach `.dmg` to GitHub Release
- [ ] Install + launch verified on a fresh Mac

---

## Milestone 4 — Polish

### 4.1 Categories
- [ ] Define `Category` domain model
- [ ] Backend: `categories` table seeded with standard set
- [ ] `GET /v1/categories`
- [ ] Mac syncs category list at launch
- [ ] Android syncs category list
- [ ] Category-based limits in the policy model
- [ ] PolicyEngine: resolve app → category at evaluate time
- [ ] End-to-end tests

### 4.2 Android notifications
- [ ] FCM project configured
- [ ] Backend: send FCM when a limit crosses 80%
- [ ] Android: foreground notification handler
- [ ] Android: background notification handler
- [ ] User toggle in settings

### 4.3 Weekly summary
- [ ] Backend cron producing weekly summary per account
- [ ] FCM push delivering summary
- [ ] Deep-link into dashboard week view

### 4.4 Crash reporting
- [ ] Sentry project
- [ ] Swift SDK wired into Mac app
- [ ] Android SDK wired in
- [ ] Scrub PII
- [ ] Opt-in toggle on both clients

---

## Cross-cutting (any milestone)

- [ ] Dependabot / Renovate config
- [ ] Secret scanning enabled in GitHub settings
- [ ] CODEOWNERS file once a second contributor joins
- [ ] Threat model doc before Milestone 3 ships externally
