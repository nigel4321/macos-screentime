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
- [x] Create Xcode workspace `MacAgent.xcworkspace` *(superseded — the xcodeproj's embedded `project.xcworkspace` is the only workspace; a standalone wrapper adds nothing for a single-project, SPM-referencing build, and CI / `xcodebuild` open `MacAgent.xcodeproj` directly)*
- [x] Create app target `MacAgent` (SwiftUI, macOS) *(`mac-agent/App/project.yml` xcodegen spec → `MacAgent.xcodeproj`; bundle id `com.macagent.MacAgent`, `LSUIElement` menubar app)*
- [x] Set deployment target to macOS 14.0 *(in `Package.swift`)*
- [x] Set Swift language version to 5.9 *(`swift-tools-version:5.9`)*
- [x] Create Swift Package with `Package.swift` at `mac-agent/Package.swift`
- [x] Add package as a local dependency of the app target *(`MacAgentCore` wired via `path: ../` with `PolicyEngine`, `LocalStore`, `UsageCollector`, `SyncClient`, `AppMetadata`, `LoginItem` products)*
- [ ] Configure local development code signing *(mac-only — deferred; CI builds ad-hoc via `CODE_SIGNING_ALLOWED=NO`; real signing waits for §3.9 release pipeline)*
- [x] Decide `.xcodeproj` tracking policy — tracked; user state excluded via root `.gitignore` (documented in `mac-agent/README.md`)

### 1.2 SwiftLint
- [x] Add SwiftLint as a Swift Package plugin *(dependency added to `Package.swift`; attached per-target in §1.3+)*
- [x] Add `.swiftlint.yml` with chosen rule set
- [x] Add build-phase script to run SwiftLint on every build *(`preBuildScripts` SwiftLint phase in `mac-agent/App/project.yml` invokes the SPM-resolved `SwiftLintBinary` against `${SRCROOT}` — `.swiftlint.yml`'s `included: [App, Sources, Tests]` + `excluded: [App/build, App/MacAgent.xcodeproj, .build, .swiftpm]` scope it correctly. SPM plugin remains attached per-target for `swift build` and CI's library tests)*
- [x] Verify lint fails a deliberately bad commit *(verified live: a 161-char canary in `Sources/PolicyEngine/BundleID.swift` fails `swift build` with `error: Line Length Violation`; a 158-char canary in `App/MacAgentApp.swift` fails `xcodebuild` with the new SwiftLint phase script)*

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

### 1.12 Mac UI polish *(mac-only)*
*Late addition: M1 shipped with the Today view rendering raw bundle IDs ("com.google.Chrome"), which is unfriendly. Section reserved for Mac-side UI polish equivalent to Android's §2.21.*
- [x] App display-name resolution (`com.google.Chrome` → "Google Chrome") via `NSWorkspace` + `CFBundleDisplayName` / `CFBundleName`, with caching and a graceful bundle-id fallback for uninstalled apps *(`AppMetadata` SPM target with `SystemAppMetadataResolver`; injectable lookup keeps tests independent of CI runner state; positive + negative cache)*

### 1.11 CI
- [x] `.github/workflows/mac.yml` on `macos-14`
- [x] Run SwiftLint *(via SPM build-tool plugin during `swift build`)*
- [x] Run `xcodebuild build -scheme MacAgent` for Xcode-toolchain validation of the App target *(library tests covered by `swift test` above)*
- [x] Cache SPM packages
- [x] Status badge in README
- [x] Green build on `main`

### 1.13 Launch at login *(mac-only)*
*The MacAgent has to be running to collect events; for a parent's machine we want it auto-resumed on every login. macOS treats per-user "start at login" and post-boot launch as the same mechanism, since `NSWorkspace` lives inside a user session — a real boot-time `LaunchDaemon` would run as root before any user logs in and couldn't observe per-user app activations. **Launch-at-login is mandatory by design: there is no in-app opt-out.** The user's only way to disable it is **System Settings → General → Login Items**, which macOS keeps authoritative — `register()` cannot override an explicit user disable. §1.14 closes that gap for child accounts.*
- [x] `LoginItem` SPM target with a `LoginItemRegistry` protocol so the menubar code depends on a seam, not directly on `ServiceManagement`
- [x] `SMAppServiceLoginItemRegistry` production impl (macOS 13+; we target 14+)
- [x] `ensureEnabled()` runs on every launch — idempotent re-register, no in-app toggle
- [x] Tests with a fake registry (real `SMAppService` calls only meaningfully run from a properly-bundled `.app` context)

### 1.14 Tamper-resistant launch (system-wide LaunchAgent) *(mac-only)*
*§1.13's `SMAppService.mainApp` registers a **per-user** login item — meaning a child user can disable MacAgent in their own session via **System Settings → General → Login Items** with no password prompt. To make launch-at-login tamper-resistant on a child's account, we additionally install a **system-wide `LaunchAgent`** at `/Library/LaunchAgents/`, which runs for every user when they log in and which a non-admin child cannot remove. §1.13 and §1.14 are complementary, not replacements: §1.13 keeps the agent live on the parent's account before §1.14's admin install is run, and is what the parent's first launch flips on automatically.*
- [x] Choose install vector — **signed `.pkg`** preferred over shell script: single admin prompt, postinstall script runs as root, distributable via the same release pipeline as the `.dmg`
- [x] Author `LaunchAgent` plist (`mac-agent/installer/com.macagent.MacAgent.plist`) with `RunAtLoad`, `KeepAlive` (`SuccessfulExit=false` so a clean Cmd-Q from the parent's menubar isn't auto-respawned), `LimitLoadToSessionType=Aqua`, `Program` pointing at `/Applications/MacAgent.app/Contents/MacOS/MacAgent` *(Label uses the actual bundle id `com.macagent.MacAgent`, not the placeholder `com.macos-screentime.MacAgent` from the original spec)*
- [x] Postinstall script (`mac-agent/installer/postinstall`): set plist `root:wheel` mode `644`, `launchctl bootstrap gui/<uid>` into the active console user's domain (modern replacement for `launchctl load -w`); idempotent via `bootout` first; no-ops cleanly on headless installs
- [x] One-time installer entry point (`mac-agent/scripts/make-pkg.sh`) — composes the `.app` + plist + postinstall into a distribution `.pkg` via `pkgbuild` + `productbuild`. Opt-in signing via `MACAGENT_INSTALLER_SIGN_IDENTITY` env var; unsigned builds run for local testing via `installer -pkg`
- [ ] Verify on a non-admin user account: agent launches on login; user cannot remove it from System Settings (system LaunchAgents don't appear in the per-user Login Items list); user cannot delete `/Library/LaunchAgents/com.macagent.MacAgent.plist` without admin password — *blocked on §3.9 (Gatekeeper rejects unsigned binaries from `launchd`)*
- [x] Document the §1.13 vs §1.14 trade-off in `ARCHITECTURE.md` so the dual-mechanism rationale isn't lost — see `ARCHITECTURE.md` §2.6 "Launch-at-login: a dual mechanism" + §2.7 "Distribution shapes"
- [ ] Hard dependency on §3.9: the LaunchAgent's `Program` path must point at a Developer-ID-signed and notarized `MacAgent.app` binary, otherwise modern macOS Gatekeeper will refuse to launch it from `launchd` even with a valid plist

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
- [x] Add `SyncClient` product to Core
- [x] Device-registration flow, token stored in Keychain *(`KeychainCredentialStore` uses the data-protection keychain so reads never block on user authorization)*
- [x] `BatchUploader` reading unsynced events *(paged at backend's `MaxBatchSize=500`; accepted/duplicate/rejected all mark synced)*
- [x] Exponential backoff with jitter *(full-jitter; 5xx + transport retry, 4xx/decoding/missing-creds do not)*
- [x] Test against a `URLProtocol`-based local mock (no Vapor dep)
- [x] Wire into app lifecycle (periodic flush + on-quit flush) *(60s timer; quit replaces SwiftUI's default to call `await container.flush()`)*
- [x] LocalStore schema v2: add `client_event_id` to `usage_event` (required by backend's `(device_id, client_event_id, started_at)` idempotency contract)

> **§2.10 ships an offline-capable SyncClient** that no-ops gracefully when no JWT is present. It cannot actually push events end-to-end until §2.10a lands.

### 2.10a Mac agent: Apple Sign-In *(mac-only)*
- [x] `AuthenticationServices` `Sign in with Apple` button on `OnboardingView` *(SwiftUI's `SignInWithAppleButton`; entitlement `com.apple.developer.applesignin = ["Default"]` lives in `mac-agent/App/MacAgent.entitlements`, wired via `CODE_SIGN_ENTITLEMENTS` in xcodegen)*
- [x] `ASAuthorizationController` flow → Apple identity token *(driven by `SignInWithAppleButton.onCompletion`; identity token extracted from `ASAuthorizationAppleIDCredential.identityToken` and handed to the VM)*
- [x] Exchange identity token for backend JWT via `POST /v1/auth/apple` *(`SyncClient.AuthClient.signInWithApple(identityToken:)`; `APIClient.send` gained `requireJWT: Bool = true` so this and future public endpoints can opt out of the Authorization header)*
- [x] Persist JWT in Keychain via the `CredentialStore` from §2.10 *(success path writes `jwt` via `KeychainCredentialStore`)*
- [x] First-launch flow: sign in → register device → start periodic flush *(`AppContainer.onAuthenticated` flips `authPhase` and triggers an immediate `SyncClient.flush()`, which calls `DeviceRegistrar.register()` idempotently — device row appears on the backend within ~1s of sign-in instead of waiting up to 60s for the next periodic tick)*
- [x] Sign-out menubar action (clears JWT + device token) *("Sign out" button in `TodayView` calls `AppContainer.signOut() → AuthClient.signOut() → CredentialStore.clear()` and flips `authPhase` back to `.unauthenticated`, re-routing the menubar to `OnboardingView`)*
- [x] Tests with a fake `ASAuthorizationController` and `URLProtocol` mock backend *(9 new tests: `AuthClientTests` × 4 covers the network exchange + sign-out, `OnboardingViewModelTests` × 5 covers the state machine — idle→loading→idle, 401→error w/ Apple wording, 5xx→error w/ network wording, dismissError, retry-recovers-from-error. The "fake `ASAuthorizationController`" is realised by calling `viewModel.signIn(identityToken:)` directly with a synthetic token — the SwiftUI `SignInWithAppleButton` itself is a thin extractor and not unit-tested)*

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
- [x] `ScreentimeDatabase` (Room) in `:core-data` with KSP-generated impl, schema export to `core-data/schemas/`
- [x] `UsageSummaryRowEntity` (single table keyed by `cache_key` + indexed; auto-id, `cached_at` epoch millis) and `CacheMetadataEntity` (per-cache-key `last_refresh_at`) — metadata is separate so an empty refresh still records freshness
- [x] `UsageSummaryDao` — `observeByCacheKey` (Flow), `lastRefreshAt`, `replace` (transactional wipe-rows-+-upsert-metadata), `deleteOlderThan`, `insertAll`, `upsertMetadata`
- [x] `UsageRepository` cache-first / network-refresh: `summary()` returns `Flow<UsageSummary>` from cache; `refresh()` fetches and replaces; `isStale()` reports against injected `Clock` and `DEFAULT_TTL = 5.minutes`
- [x] Cache invalidation rules: per-key wipe-and-replace on refresh, TTL on read, global `purgeOlderThan` for app-launch sweep
- [x] `DatabaseModule` (Hilt) provides `ScreentimeDatabase`, `UsageSummaryDao`, and a `Clock` singleton
- [x] Tests: Robolectric `UsageSummaryDaoTest` exercising real Room schema; `UsageRepositoryTest` with `FakeUsageSummaryDao` + MockWebServer covering cache-miss, refresh-replace, TTL boundaries, purge

### 2.15 Android: onboarding
- [x] `GoogleSignInClient` interface in `:feature-onboarding` + `CredentialManagerGoogleSignInClient` (Credential Manager + `GetGoogleIdOption`); `GOOGLE_WEB_CLIENT_ID` BuildConfig sourced from `SCREENTIME_GOOGLE_WEB_CLIENT_ID` env var or `screentime.googleWebClientId` Gradle property
- [x] `AuthRepository.signInWithGoogle(idToken)` exchanges Google ID token for backend JWT *(landed in §2.13; reused here)*
- [x] `EncryptedSharedPreferencesTokenStore` (AES-256-GCM master key, AES256_SIV keys / AES256_GCM values) replaces `InMemoryTokenStore` as the production `@Binds` in `AuthModule`
- [x] `TokenStore.authState: StateFlow<AuthState>` seeded from disk at construction *(survives process death)*
- [x] Sign-out via `AuthRepository.signOut()` clears the persisted JWT and flips `authState` to `Anonymous`
- [x] `OnboardingViewModel` (Hilt) with state machine `Idle / Loading / Error / Authenticated`; combines transient sign-in state with `TokenStore.authState`
- [x] `OnboardingScreen` Compose UI: hero copy, "Sign in with Google" button, loading spinner, error banner with retry; uses `LocalContext` to walk to the hosting `Activity` for Credential Manager
- [x] App-level auth gate: `AuthGateViewModel` in `:app` collects `TokenStore.authState`; `MainActivity` `NavHost` start destination flips between `onboarding` ↔ `today` and re-routes on sign-in/sign-out
- [x] Tests: `OnboardingViewModelTest` (Mockito + `FakeGoogleSignInClient` + MockWebServer) covering Idle→Loading→Authenticated, Google failure → Error, backend 401 → Error, dismissError, initial-Authenticated. *(Encrypted store round-trip deferred to instrumented tests in §2.19 — Robolectric doesn't shim AndroidKeyStore.)*

### 2.16 Android: device pairing
- [x] Backend `GET /v1/devices` (added under §2.4 territory): `Store.ListDevicesForAccount` + `DevicesListHandler` mounted in the Authenticator group; `last_seen_at` is `omitempty`; integration test asserts cross-account isolation
- [x] `:core-domain` `Device` / `DeviceId` / `DevicePlatform`; `:core-data` `DeviceDto` + `DeviceListResponse` + mapper + `ScreentimeApi.listDevices()`
- [x] `DeviceRepository.list()` in `:core-data`; `SelectedDeviceStore` interface + `SharedPreferencesSelectedDeviceStore` (unencrypted — device id is server-issued and account-scoped, not sensitive)
- [x] `DeviceSession` (combines `TokenStore.authState` + `SelectedDeviceStore.selected` into `Anonymous / NeedsDevice / Ready`); replaces the §2.15 two-state `AuthGateViewModel` so the gate routes through pairing
- [x] `DevicePairingViewModel` (`Loading / Devices / ZeroDevices / Error`) + `DevicePairingScreen`: Material 3 radio list, "Continue" CTA, zero-device empty state with copy "Install the Mac agent first, then come back here.", error+retry
- [x] `MainActivity` NavHost gains `pairing` route between `onboarding` and `today`; auth gate re-navigates on every `SessionState` transition
- [x] Tests: backend unit (success / empty array not null / 401 / 500) + integration (cross-account isolation, ordering, empty); Android `DeviceRepositoryTest` (MockWebServer round-trip, empty response, unknown platform → `Unknown`); `DevicePairingViewModelTest` (Loading→Devices, Loading→ZeroDevices, Loading→Error, selectAndContinue persists, retry recovers)

### 2.17 Android: dashboard — today
- [x] `TodayViewModel` (Hilt) — observes the `:core-data` cache `Flow` for `[startOfDay, now)` grouped by `bundle_id`, refreshes via `LaunchedEffect` / pull-to-refresh / retry; sealed `UiState` (`Loading / Empty / Loaded(rows, totalDuration, isRefreshing) / Error`); `refresh()` is `suspend` so tests can await without racing OkHttp's real-thread I/O
- [x] Bento grid via `LazyVerticalGrid(GridCells.Fixed(2))` — total-today (2×1), top apps as 1×1 tiles capped at 4 (rank + name + duration + share-of-leader bar), categories placeholder (2×1, "coming with category aggregation in §4.1"), downtime status (2×1, "no active downtime" until §3.7 data lands)
- [x] Pull-to-refresh via Material 3 `PullToRefreshBox`
- [x] Loading skeleton (4 full-width placeholder tiles mirroring the loaded shape), error + retry, empty state ("No usage today yet")
- [x] `enableEdgeToEdge()` in `MainActivity`
- [x] Tests: `TodayViewModelTest` (initial Loading, Loading→Empty, Loading→Loaded sorted desc by duration, Loading→Error, refresh-recovers-from-Error, refresh-while-in-flight is a no-op, query window matches Clock + system zone with `groupBy=bundle_id`); `FormatTest` for the human duration formatter

*Deferred from §2.17 — re-anchored where their dependencies actually land:*
- [ ] Vico `CartesianChartHost` chart upgrade (replace the hand-rolled bars in TopAppsTile) — defer to §2.18 since the Week tab also needs charting
- [ ] Translucent (glass) top bar + bottom nav using `RenderEffect.createBlurEffect` — defer to §2.21 polish; `:app` only has a single top-level tab today, so the nav surface to glassify isn't there yet
- [ ] Shared-element transition (tap app tile → app-detail) — defer until an app-detail screen exists; no destination to transition into in M2
- [ ] Long-press app row → bottom sheet (set limit / hard block / view week) — defer to §3.6+ when policy-edit endpoints exist; until then the sheet would have no real actions
- [x] Compose UI tests per state — `TodayScreenSemanticsTest` covers loading/empty/error/loaded with semantic assertions; landed alongside §2.18 once Today/Week tab nav made them meaningful

### 2.18 Android: dashboard — week
- [x] `WeekViewModel` (Hilt) — runs two cache-keyed queries over a 7-day `[startOfRange, now)` window: `groupBy=day` for the chart, `groupBy=bundle_id` for the top-apps tiles. Sealed `UiState` (`Loading / Empty / Loaded(byDay, topApps, totalDuration, isRefreshing) / Error`). `densifyByDay` fills zero-duration buckets so the chart always shows 7 contiguous bars.
- [x] Per-day bar chart — hand-rolled `WeekBarChart` (7 columns, proportional heights, day-initial labels, content-description for TalkBack); empty days show the grey track only. Stacked-by-bundle-id upgrade deferred to §2.22-followup once apps have categories (§4.1) so the stacks are meaningful.
- [x] `WeekScreen` bento — total-this-week (2×1) + chart tile (2×1) + top-4 apps as 1×1 tiles (reuses `TopAppTile`) + Categories/Downtime placeholders (2×1 each). Pull-to-refresh, loading skeleton, error+retry, empty — same components as Today.
- [x] Tab navigation — `DashboardHost` Scaffold + Material 3 `NavigationBar` switching between `TodayScreen` and `WeekScreen` (single `dashboard` route in the auth gate's NavHost). Selection survives configuration change via `rememberSaveable`.
- [x] `WeekViewModelTest` mirrors `TodayViewModelTest` (initial-loading, refresh→empty, refresh→loaded with densified 7 days + sorted top apps, refresh→error, refresh recovers from error, concurrent-refresh is a no-op). Compose semantics tests (`TodayScreenSemanticsTest`, `WeekScreenSemanticsTest`) assert state-to-UI wiring; Roborazzi screenshot tests for both screens upload as the existing CI artifact.

*Deferred:*
- [ ] Stacked-bar chart (per-day, per-bundle stacks) — re-anchor to §2.22-followup after categories aggregation lands; meaningful stacking needs a small set of stable buckets, not raw bundle ids.
- [ ] Shared-element transition (tap day column → day-detail) — no day-detail screen exists; defer until that screen is needed.

### 2.19 Android CI — extension
*Foundational CI (workflow file, ktlint, detekt, assembleDebug, Gradle cache) lands in §2.11 so PRs are gated from §2.12 onward. This section adds the bits that depend on later milestones.*
- [x] CI runs unit tests for `:core-domain` and `:core-data` *(workflow `Unit tests` step added in §2.13; §2.14 will plug in Room cache tests)*
- [x] Compose screenshot tests via Roborazzi for `TodayScreen` states (loading / empty / error / loaded); CI uploads PNGs as `today-screenshots-<sha>` artifact for visual review *(no emulator needed — runs as a JVM Robolectric task)*
- [x] Functional Compose UI tests (semantic assertions) — `TodayScreenSemanticsTest` and `WeekScreenSemanticsTest` cover loading/empty/error/loaded; landed in §2.18
- [~] CI runs `assembleRelease` *(signing config from §2.20 is in place; the dedicated `assembleRelease` step in `.github/workflows/android.yml` is still pending — `android-release.yml` exercises `bundleRelease` end-to-end on every `android-v*` tag, which is the harder check)*

### 2.20 Android release via Fastlane
- [x] `fastlane/` under `android-app/` *(`Fastfile`, `Appfile`, `Pluginfile`; pinned via `Gemfile` ~> 2.224)*
- [x] `Fastfile` with `internal` lane *(plus `alpha` / `beta` / `production` — every lane uploads as `release_status: draft` so nothing reaches users without a manual roll-out click)*
- [x] `:app` signing config + tag-driven version *(`build.gradle.kts` parses `GITHUB_REF_NAME`: `android-vMAJOR.MINOR.PATCH` → `versionName=MAJOR.MINOR.PATCH`, `versionCode=MAJOR*1_000_000 + MINOR*1_000 + PATCH`; release `signingConfig` reads `RELEASE_KEYSTORE_PATH` / `_PASSWORD` / `RELEASE_KEY_ALIAS` / `RELEASE_KEY_PASSWORD` env vars; absent vars degrade gracefully to unsigned for local dev)*
- [x] `.github/workflows/android-release.yml` on `android-v*` tags *(also `workflow_dispatch` with a `track` input for promoting an existing tag up the alpha→beta→production ladder; uploads the AAB as a workflow artifact regardless)*
- [x] Local end-to-end validation *(built signed `app-release.aab` against tag `android-v0.2.5` with a throwaway keystore; merged manifest reports `versionCode=2005, versionName="0.2.5"`; jarsigner confirms signature attached)*
- [ ] Play Store service-account JSON (user-provided secret) *(see `android-app/RELEASE.md` for the Play Console + GCP setup; secret name `PLAY_STORE_SERVICE_ACCOUNT_JSON`)*
- [ ] Upload keystore secrets (user-provided) *(`ANDROID_KEYSTORE_BASE64` + `ANDROID_KEYSTORE_PASSWORD` + `ANDROID_KEY_ALIAS` + `ANDROID_KEY_PASSWORD`; `keytool` + `base64` recipe in `RELEASE.md`)*
- [ ] First successful internal-track upload *(blocked on Play Console listing creation + the Play-required first-manual-upload — doc'd in `RELEASE.md`)*
- [ ] Verify install on a tester device

### 2.21 Android: UI polish + accessibility
*Should land before §2.20 first internal release if practical; numbering kept stable.*
- [ ] Haptic feedback on long-press, limit-cross, policy-saved (`HapticFeedback`) — re-anchor to §3.6+ since none of the trigger events (long-press app row, limit crossed, policy saved) exist in M2; the M2 dashboard is read-only with no haptic surfaces. Will add the `HapticFeedback` plumbing alongside the editor screens that fire the events.
- [x] Accessibility audit *(every clickable in the M2 surfaces — sign-in, device-row select, retry, tab-switch — already has a visible button affordance. Each bento tile now exposes one merged TalkBack unit via `Modifier.semantics(mergeDescendants = true)`: "Total today: 45m"; "Rank N: <name>, <duration>"; "Daily totals: Mon 21 Apr: 2h 14m; Tue 22 Apr: ..."; "Categories: coming with category aggregation in 4.1"; "Downtime: no active downtime". Chart x-axis labels Mon/Tue/... rendered as visible Text and assertable.)*
- [x] Motion review *(no animations in M2 surfaces — no `animate*AsState`, `Crossfade`, `AnimatedVisibility`, or `tween` calls anywhere in `feature-dashboard`/`feature-onboarding`/`app`. State transitions are instantaneous; trivially under 400ms. Real motion review re-opens at §2.21-revisit once shared-element transitions land in §3.6+.)*
- [x] Whitespace pass *(audited at §2.18: TodayScreen / WeekScreen content padding 16dp, grid gutters 12dp, BentoTile inner padding 16dp. Internal in-tile spacers are deliberately tighter (2-12dp) and unaffected by the bullet's tile-level constraints.)*

### 2.22 App display-name resolution (Mac → backend → clients)
*The dashboards currently render raw bundle IDs (`com.tinyspeck.slackmacgap`). The Mac agent already has §1.12's `AppMetadata` resolver — making it the source of truth means every client (Android, future iOS, future web) gets human names "for free" without re-implementing macOS-specific lookup. Three independent half-PRs that can land in any order; clients gracefully fall back to bundle ID until step 2 lands.*

- [x] **Backend**: new `app_metadata` table keyed `(account_id, bundle_id) → (display_name, updated_at)`. `POST /v1/usage:batchUpload` accepts an optional batch-level `app_metadata: { bundle_id → display_name }` map (capped at 100 entries × 256 chars); `Store.UpsertAppMetadataBatch` lands them in a single transaction (latest write wins). `GET /v1/usage:summary` LEFT JOINs `app_metadata` when grouping by bundle and returns `display_name` per row (`omitempty` when never seen). Handler unit tests cover persist-before-events ordering, omitted-field is OK, oversized payload (413), oversized value (400), store-error short-circuit. Integration tests cover round-trip, latest-write-wins, cross-account isolation, empty-map no-op, skip empty values.
- [x] **Mac agent** *(mac-only)*: `BatchUploader` (§2.10) attaches a batch-level `app_metadata: { bundle_id → display_name }` map to every `POST /v1/usage:batchUpload`. The map is built per page from the unique bundle IDs in the batch via the injected `AppMetadataResolver`; entries that round-trip back to the bundle id (the resolver's "miss" signal) are dropped so the backend's `app_metadata` table doesn't get polluted with rows where `display_name == bundle_id`. Field is `omitempty` when nothing resolves. Capped at 100 entries per batch (mirrors backend `MaxAppMetadataEntries`). Tests with `FakeAppMetadataResolver` cover: includes resolved entries, dedupes repeated bundles within a page (one resolver call per unique bundle), omits the field entirely when nothing resolves. *Deferred — separate "new-bundle" upsert: skipped because the 60s flush cadence already lands names within a minute of first observation; a second endpoint would be redundant.*
- [x] **Android**: `UsageRow.displayName: String?` in `:core-domain`; `SummaryRowDto.displayName` in `:core-data`; cached in `UsageSummaryRowEntity` (Room schema bumped v1→v2 with an `ALTER TABLE ADD COLUMN display_name` migration); plumbed through `UsageRepository` (DTO → entity → domain). `TopAppTile` prefers `displayName` and falls back to `bundleId.value`, then `"Unknown"`. Test coverage: `ScreentimeApiTest` parses `display_name`, `UsageRepositoryTest` round-trips it through cache + maps null when absent. Screenshot sample mixes a metadata-less row so the artifact shows the fallback path.

## Milestone 3 — Policy enforcement

### 3.1 Backend: policy mutation
- [x] `PUT /v1/policy` handler
- [x] Version increment on write
- [x] Optimistic concurrency via `If-Match: version` *(unique-constraint-backed; concurrent writers translate `(account_id, version)` PK violation into `ErrVersionConflict`)*
- [x] Server-side policy shape validation *(`policy.Document.Validate` — bundle-id/length bounds, HH:MM time format, weekday allowlist, no zero-length windows, no duplicate days, daily-limit sanity)*
- [x] Authz: only account owner writes *(account id pulled from JWT context; mounted under the existing `Authenticator` group)*
- [x] Handler tests *(success, missing/bad/quoted/bare `If-Match`, validation error, version conflict surfacing current doc, body-version-ignored, unauth, store-error)*
- [x] Store integration tests *(round-trip, version increment, stale-expected conflict, two-goroutine race resolves to 1 win + 1 conflict, cross-account isolation)*

### 3.2 Backend: WebSocket policy subscribe
- [x] `WS /v1/policy/subscribe` *(coder/websocket; handler at `internal/api/policy_subscribe_handler.go`)*
- [x] Auth handshake on first message *(client sends `{"type":"auth","token":"<JWT>"}` within 5s; route mounted outside the HTTP Authenticator group so headers aren't required for browser-style clients)*
- [x] In-memory pub/sub registry *(`policy.Broker`: per-account map, RWMutex, buffered per-subscriber channel with non-blocking send so slow consumers can't backpressure the PUT path)*
- [x] Emit new version on PUT commit *(`policy.Publisher` plumbed into `PolicyPutHandler`; published only after a successful write — conflicts and validation errors don't emit)*
- [x] Heartbeat + idle timeout *(server pings every 30s; idle read deadline 90s; per-write deadline 10s)*
- [x] Document reconnection semantics *(no history replay — initial frame on connect carries `store.Current().Version`, then live deltas; clients re-fetch `GET /v1/policy/current` to reconcile after a suspected gap)*

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
- [ ] Developer ID Installer certificate provisioned *(distinct from the Application cert; required by `productsign` for the `.pkg` in §1.14)*
- [ ] Codesign in CI using secret-stored `.p12`
- [ ] `xcrun notarytool submit --wait` *(notarize both the `.app` inside the `.dmg` and the `.pkg` from §1.14 — Gatekeeper checks both)*
- [ ] `xcrun stapler staple` (applied to `.app`, `.dmg`, and `.pkg`)
- [ ] `.dmg` via `create-dmg` for the parent-machine install
- [ ] `.pkg` produced via `pkgbuild` + `productbuild` for §1.14's tamper-resistant install (bundles `MacAgent.app` + `LaunchAgent` plist, postinstall script drops plist into `/Library/LaunchAgents/`)
- [ ] `.github/workflows/mac-release.yml` on `mac-v*` tags
- [ ] Attach both `.dmg` and `.pkg` to the GitHub Release
- [ ] Install + launch verified on a fresh Mac (both the `.dmg` flow for the parent and the `.pkg` flow on a non-admin child account)

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
