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

### 1.4 PolicyEngine — no-policy case (TDD)
- [x] RED: `evaluate` returns empty actions when policy is nil
- [x] GREEN: minimal `evaluate` implementation
- [x] RED: `evaluate` returns empty when policy has no rules
- [x] GREEN: extend implementation

### 1.5 PolicyEngine — per-app daily limits (TDD)
- [x] RED: limit not crossed → no shield
- [x] GREEN: accumulate today's usage for bundleId, compare to limit
- [x] RED: limit exactly crossed → shield emitted
- [x] RED: yesterday's data alone does not trip today's limit
- [x] GREEN: scope accumulation to "today" via injected clock
- [x] RED: multiple apps with independent limits
- [x] GREEN: handle N apps

### 1.6 PolicyEngine — downtime windows (TDD)
- [x] RED: outside window → no shield
- [x] RED: inside window → shield all apps on block list
- [x] RED: window crossing midnight
- [x] RED: DST spring-forward boundary
- [x] RED: DST fall-back boundary
- [x] GREEN: implement window evaluation with injected `Clock` + `TimeZone`
- [x] Property test: window-active ⇔ now ∈ [start, end]

### 1.7 LocalStore
- [x] Add GRDB dependency to `Core`
- [x] Add `LocalStore` product
- [x] Schema v1 migration: `usage_event`, `policy`
- [x] Migration runner with version tracking
- [x] Migration test: fresh DB → v1
- [x] Migration test: re-run is idempotent
- [x] DAO: insert usage event (idempotent on client id)
- [x] DAO: query unsynced events
- [x] DAO: mark events synced
- [x] DAO: read/write current policy
- [x] DAO tests with in-memory GRDB

### 1.8 UsageCollector
- [x] Add `UsageCollector` product
- [x] Abstract `WorkspaceSource` protocol
- [x] Real implementation wrapping `NSWorkspace.didActivateApplicationNotification`
- [x] Fake implementation for tests
- [x] Translate activation stream to closed `(bundleId, start, end)` events
- [x] Handle system sleep → close open event
- [x] Handle system wake → resume on active app
- [x] Handle screen lock → close open event
- [x] Unit tests for each transition

### 1.9 App wiring *(mac-only)*
- [x] Root `MenuBarExtra` scene
- [x] Hide Dock icon via `LSUIElement` in Info.plist
- [x] Dependency container wiring UsageCollector → LocalStore
- [x] Start collection on app launch
- [x] Graceful shutdown on quit (flush open event)

### 1.10 Today view *(mac-only)*
- [x] `TodayViewModel` querying LocalStore for today's aggregated usage
- [x] Top-5 apps list UI in the menu popover
- [x] Live updates as events land
- [x] Empty state
- [x] First-launch onboarding screen (observe-only copy)

### 1.11 CI
- [x] `.github/workflows/mac.yml` on `macos-14`
- [x] Run SwiftLint *(via SPM build-tool plugin during `swift build`)*
- [x] Run `xcodebuild build -scheme MacAgent` for Xcode-toolchain validation of the App target *(library tests covered by `swift test` above)*
- [x] Cache SPM packages
- [x] Status badge in README
- [x] Green build on `main`

---

## Milestone 2 — Backend + Android read-only

### 2.1 Backend: module scaffolding
- [x] Init `backend/` Go module
- [x] Layout: `cmd/server`, `internal/{api,db,auth,model}`, `migrations`
- [x] HTTP server with chi router
- [x] `/healthz` endpoint
- [x] Env-var config loader
- [x] Structured logging via `log/slog`
- [x] Graceful shutdown on SIGTERM
- [x] `golangci-lint` config
- [x] `Makefile` with run/test/lint targets

### 2.2 Backend: Postgres + migrations
- [x] Choose migration tool (goose)
- [x] Add `pgtestdb` for integration tests
- [x] Migration 001: `account` *(includes `account_identity` per ARCHITECTURE §3.3)*
- [x] Migration 002: `device`
- [x] Migration 003: `usage_event` parent + monthly partitioning
- [x] Migration 004: `policy`
- [x] Helper to create next month's partition
- [x] Connection pool with `pgxpool`
- [x] Include DB ping in `/healthz`

### 2.3 Backend: auth
- [x] JWT signing key management (env var, rotatable) *(ES256, kid from SHA-256 of SPKI; `JWT_VERIFICATION_KEYS` carries rotated-out keys)*
- [x] Apple identity-token verifier (JWKS fetch + cache)
- [x] Google identity-token verifier (JWKS fetch + cache)
- [x] `POST /v1/auth/apple` → backend JWT
- [x] `POST /v1/auth/google` → backend JWT
- [x] `POST /v1/account:pair-init` — Mac requests a 6-digit pairing code (~10m TTL)
- [x] `POST /v1/account:pair-complete` — Android redeems the code; merges Google account into Apple account, moves `account_identity` rows *(also moves any `device` rows defensively; merge runs in a SERIALIZABLE transaction)*
- [x] `Authenticator` middleware: verify JWT, load account
- [x] `DeviceContext` middleware: resolve device from token
- [x] Tests for each verifier with signed fixtures *(local httptest JWKS server; provider-specific iss/aud/expiry checks)*
- [x] Tests for authz denial paths
- [x] Tests for the pairing flow (init → complete, expired code, code reuse, double-pair)

### 2.4 Backend: `POST /v1/devices/register`
- [x] Route + request/response types
- [x] Insert device row keyed to account
- [x] Return device id + device token *(plaintext returned once; server stores SHA-256 hash)*
- [x] Idempotent on (account, device fingerprint) *(UPSERT rotates token, bumps last_seen_at)*
- [x] Handler tests *(unit + integration: register, idempotent rotation, cross-account isolation, validation)*

### 2.5 Backend: `POST /v1/usage:batchUpload`
- [x] Route handler accepting batch JSON
- [x] Idempotency on client-supplied event id *(`UNIQUE (device_id, client_event_id, started_at)` + `ON CONFLICT DO NOTHING`)*
- [x] Validate device owns account *(via `auth.DeviceContext` middleware: device row is account-scoped at registration time)*
- [x] Insert into partitioned table *(startup ensures prev/current/next month partitions; validation window stays inside that range)*
- [x] Return per-event accept/reject *(`accepted` | `duplicate` | `rejected`, in input order)*
- [x] Handler tests including duplicate suppression *(unit + integration: accepted→duplicate, mixed-batch validation, distinct start times, out-of-window)*

### 2.6 Backend: `GET /v1/usage:summary`
- [x] Accept query params `from`, `to`, `groupBy` *(RFC3339 bounds; `groupBy` comma-separated; range capped at 90 days)*
- [x] SQL: sum durations grouped by bundleId and/or day *(dynamic SELECT/GROUP BY; day bucketed via `date_trunc('day', started_at AT TIME ZONE 'UTC')`)*
- [x] Return JSON aggregates *(`{"results":[{bundle_id?, day?, duration_seconds}]}`; no-grouping returns single total row)*
- [x] Enforce account ownership of devices in query *(JOIN device d ON d.id = ue.device_id WHERE d.account_id = $1; cross-account isolation covered by integration test)*
- [x] Handler tests *(unit: success, no-grouping, both axes, unauth, bad params, store error; integration: total + by-bundle, by-day, cross-account, empty range)*

### 2.7 Backend: `GET /v1/policy/current` (stub)
- [x] Route handler returning empty policy v0 *(`{version:0, app_limits:[], downtime_windows:[], block_list:[]}`; mounted under Authenticator group, no DB deps until M3 persistence lands)*
- [x] Handler test *(unit: success, empty arrays vs null on the wire, 401 unauth; router test confirms route is disabled when JWTVerifier is absent)*

### 2.8 Backend CI
- [x] `.github/workflows/backend.yml` with path filter `backend/**`
- [x] Postgres service container *(postgres:17, healthcheck-gated; matches Fly's `postgres-flex:17.x` runtime)*
- [x] `go vet`, `go test`, `golangci-lint`
- [x] Cache Go modules *(via `actions/setup-go@v5` built-in cache)*

### 2.9 Backend deploy
- [x] Dockerfile (multi-stage, distroless base) *(`golang:1.26-bookworm` builder → `distroless/static-debian12:nonroot`; static `CGO_ENABLED=0` build, ~23 MB image)*
- [x] Fly.io (or Railway) app provisioned *(Fly app `macos-screentime-backend`, primary region `lhr`)*
- [x] Postgres provisioned *(legacy unmanaged `fly postgres` cluster `macos-screentime-pg`, shared-cpu-1x / 1 GB volume / postgres-flex 17.x; cheaper than Fly Managed Postgres for dev)*
- [x] Secrets set (JWT key, DB URL) *(`DATABASE_URL` wired via `flyctl postgres attach`; `JWT_SIGNING_KEY` generated by `scripts/gen-jwt-key.sh` and pushed via `flyctl secrets set`)*
- [x] First deploy *(green; `/healthz` reports `status=ok`, `database=ok`)*
- [x] Smoke-test script hitting `/healthz` *(`scripts/smoke-deploy.sh`, asserts `"status":"ok"` JSON)*

### 2.10 Mac agent: SyncClient
- [ ] Add `SyncClient` product to Core
- [ ] Device-registration flow, token stored in Keychain
- [ ] `BatchUploader` reading unsynced events
- [ ] Exponential backoff with jitter
- [ ] Test against a local Vapor mock server
- [ ] Wire into app lifecycle (periodic flush + on-quit flush)

### 2.11 Android: project setup
- [x] Init `android-app/` Gradle project
- [x] Kotlin 2.1, AGP 8.13, Gradle 8.13, JDK 21 toolchain, `minSdk 31`, `compileSdk 36`, `targetSdk 36` *(minSdk 31 chosen for full Material You 3.0 + `RenderEffect.createBlurEffect` glass surfaces in §2.17)*
- [x] Compose BOM, Material 3
- [x] Hilt plugin + dependencies
- [x] Retrofit, OkHttp, kotlinx.serialization
- [x] Room
- [x] Vico charts
- [x] ktlint plugin
- [x] detekt plugin
- [x] Version catalog in `libs.versions.toml`
- [x] `.github/workflows/android.yml` with path filter `android-app/**` *(foundational CI pulled forward from §2.19 so PRs from §2.12+ are gated)*
- [x] CI runs `ktlintCheck`, `detekt`, `assembleDebug`
- [x] Gradle cache via `gradle/actions/setup-gradle`

### 2.12 Android: module split
- [x] `build-logic` includedBuild with convention plugins (`screentime.android.application(.compose)`, `screentime.android.library(.compose)`, `screentime.android.feature`, `screentime.kotlin.library`) so each module's `build.gradle.kts` is a thin plugin/dependency block
- [x] `:app` (Compose entry, NavHost, DI root) — uses `screentime.android.application.compose`
- [x] `:core-domain` (pure Kotlin) — uses `screentime.kotlin.library`
- [x] `:core-data` (Retrofit + Room + repositories) — uses `screentime.android.library` + Hilt
- [x] `:core-ui` (Material 3 theme + `dynamicColorScheme`, bento tile composables, glass-surface helpers, motion utilities, haptics) — uses `screentime.android.library.compose`
- [x] `:feature-onboarding` — uses `screentime.android.feature`
- [x] `:feature-dashboard` — uses `screentime.android.feature`
- [x] Wire Hilt across modules (`@HiltAndroidApp` in `:app`, `@AndroidEntryPoint` per Activity, per-module `@Module` shells)

### 2.13 Android: network layer
- [x] Retrofit `ScreentimeApi` interface for v1 endpoints (`/v1/auth/google`, `/v1/account:pair-complete`, `/v1/usage:summary`, `/v1/policy/current`)
- [x] `AuthInterceptor` adding `Authorization: Bearer <jwt>` (skipped when token is absent)
- [x] `AuthAuthenticator` for 401 — clears token, flips `TokenStore.authState` to `Anonymous` so the UI re-routes to sign-in *(backend has no refresh-token endpoint by design — re-auth goes through Google Sign-In again)*
- [x] `kotlinx.serialization` `@Serializable` DTOs in `:core-data` + DTO→domain mappers; domain types live in `:core-domain`
- [x] `AuthRepository`, `UsageRepository`, `PolicyRepository` in `:core-data`
- [x] MockWebServer unit tests covering API round-trip, interceptor header, 401 → token-clear, repository mapping

### 2.14 Android: local cache
- [ ] `ScreentimeDatabase` (Room) in `:core-data` with KSP-generated impl, schema export to `core-data/schemas/`
- [ ] `UsageSummaryRowEntity` (single table keyed by `cache_key` + indexed; auto-id, `cached_at` epoch millis) and `CacheMetadataEntity` (per-cache-key `last_refresh_at`) — metadata is separate so an empty refresh still records freshness
- [ ] `UsageSummaryDao` — `observeByCacheKey` (Flow), `lastRefreshAt`, `replace` (transactional wipe-rows-+-upsert-metadata), `deleteOlderThan`, `insertAll`, `upsertMetadata`
- [ ] `UsageRepository` cache-first / network-refresh: `summary()` returns `Flow<UsageSummary>` from cache; `refresh()` fetches and replaces; `isStale()` reports against injected `Clock` and `DEFAULT_TTL = 5.minutes`
- [ ] Cache invalidation rules: per-key wipe-and-replace on refresh, TTL on read, global `purgeOlderThan` for app-launch sweep
- [ ] `DatabaseModule` (Hilt) provides `ScreentimeDatabase`, `UsageSummaryDao`, and a `Clock` singleton
- [ ] Tests: Robolectric `UsageSummaryDaoTest` exercising real Room schema; `UsageRepositoryTest` with `FakeUsageSummaryDao` + MockWebServer covering cache-miss, refresh-replace, TTL boundaries, purge

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
- [ ] Bento grid via `LazyVerticalGrid(GridCells.Fixed(2))` — total-today (2×1), top-N apps (1×1 each), category breakdown (2×1), downtime status (2×1)
- [ ] Vico bar chart inside the wide tiles
- [ ] Edge-to-edge layout (`enableEdgeToEdge()`); translucent top bar + bottom nav using `RenderEffect.createBlurEffect` (lives in `:core-ui`)
- [ ] Shared-element transition: tap an app tile → expand into app-detail screen (`SharedTransitionLayout`, Compose 1.7+)
- [ ] Long-press an app row → bottom sheet (set limit / hard block / view week); always reachable from a visible affordance for TalkBack
- [ ] Pull-to-refresh
- [ ] Loading skeleton
- [ ] Error + retry state
- [ ] Empty state
- [ ] Compose UI tests for each state

### 2.18 Android: dashboard — week
- [ ] `WeekViewModel` fetching per-day summary
- [ ] Vico stacked bars, full-bleed (chart extends to safe-area edges; gridlines respect insets)
- [ ] Tab navigation between Today and Week
- [ ] Shared-element transition: tap a day column → expand into day-detail
- [ ] Compose UI tests

### 2.19 Android CI — extension
*Foundational CI (workflow file, ktlint, detekt, assembleDebug, Gradle cache) lands in §2.11 so PRs are gated from §2.12 onward. This section adds the bits that depend on later milestones.*
- [x] CI runs unit tests for `:core-domain` and `:core-data` *(workflow `Unit tests` step added in §2.13; §2.14 will plug in Room cache tests)*
- [ ] Compose UI tests (or document deferral) *(requires §2.17/§2.18 dashboard tests)*
- [ ] CI runs `assembleRelease` *(requires keystore + signing config from §2.20)*

### 2.20 Android release via Fastlane
- [ ] `fastlane/` under `android-app/`
- [ ] `Fastfile` with `internal` lane
- [ ] Play Store service-account JSON (user-provided secret)
- [ ] Upload keystore secret
- [ ] `.github/workflows/android-release.yml` on `android-v*` tags
- [ ] First successful internal-track upload
- [ ] Verify install on a tester device

### 2.21 Android: UI polish + accessibility
*Should land before §2.20 first internal release if practical; numbering kept stable.*
- [ ] Haptic feedback on long-press, limit-cross, policy-saved (`HapticFeedback`)
- [ ] Accessibility audit: every gesture has a visible button alternative; TalkBack labels on all bento tiles and chart axes
- [ ] Motion review: shared transitions feel right at 120Hz; no animation duration > 400ms outside transitions; verify no jank on a low-end device (4 GB RAM, mid-tier SoC)
- [ ] Whitespace pass: re-pad after first usable build; tile inner padding ≥ 16dp, grid gutters ≥ 12dp

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
