import AppMetadata
import Foundation
import IOKit
import LocalStore
import LoginItem
import Observation
import SyncClient
import UsageCollector
import os

/// Owns every long-lived dependency and wires them together.
/// Created once at app launch and torn down on quit.
@Observable
@MainActor
final class AppContainer {
    let todayViewModel: TodayViewModel
    let onboardingViewModel: OnboardingViewModel

    /// Drives the menubar UI. Hydrated from `CredentialStore` at init so a
    /// returning user goes straight to `TodayView`.
    enum AuthPhase: Equatable {
        case unauthenticated
        case authenticated
    }

    private(set) var authPhase: AuthPhase

    private let usageEventDAO: UsageEventDAO
    private let source: NSWorkspaceSource
    private let collector: UsageCollector
    private let syncClient: SyncClient
    private let authClient: AuthClient
    private let credentials: CredentialStore
    let loginItem: LoginItemController
    private let logger = Logger(subsystem: "com.macos-screentime.MacAgent", category: "AppContainer")

    /// Periodic flush cadence. 60s is the smallest cadence that meaningfully
    /// lowers data loss on unexpected quit while keeping wakeup pressure low.
    private static let flushInterval: TimeInterval = 60

    // nonisolated(unsafe) so `deinit` (which runs nonisolated) can cancel
    // the Task without an actor hop. `Task.cancel()` is thread-safe; we
    // only ever assign this property on the main actor, so the unsafe
    // exception is benign.
    private nonisolated(unsafe) var flushTask: Task<Void, Never>?

    init(baseURL: URL = AppContainer.defaultBaseURL) {
        // swiftlint:disable:next force_try
        let db = try! AppDatabase(path: Self.databasePath())
        usageEventDAO = UsageEventDAO(database: db)
        source = NSWorkspaceSource()
        let dao = usageEventDAO
        let vm = TodayViewModel(dao: dao)
        todayViewModel = vm
        collector = UsageCollector(source: source) { event in
            try? dao.insert(event)
            Task { @MainActor in vm.refresh() }
        }
        let credentials = KeychainCredentialStore()
        self.credentials = credentials
        let api = APIClient(baseURL: baseURL, credentials: credentials)
        authClient = AuthClient(api: api, credentials: credentials)
        let onboardingVM = OnboardingViewModel(authClient: authClient)
        onboardingViewModel = onboardingVM
        syncClient = SyncClient(
            baseURL: baseURL,
            credentials: credentials,
            dao: dao,
            fingerprint: Self.deviceFingerprint(),
            resolver: SystemAppMetadataResolver()
        )
        loginItem = LoginItemController.makeDefault()

        // Seed initial auth state from the keychain so a returning user
        // skips onboarding. A keychain read failure means we can't trust
        // any cached JWT — treat as unauthenticated and let the user sign
        // in again.
        let hasJWT = (try? credentials.readJWT())?.isEmpty == false
        authPhase = hasJWT ? .authenticated : .unauthenticated

        // Launch-at-login is mandatory: re-register on every launch.
        // ensureEnabled() is idempotent. macOS keeps the user's choice
        // authoritative if they disable it via System Settings, so this
        // can leave us in `.requiresApproval` — log it but don't fail.
        do {
            try loginItem.ensureEnabled()
        } catch {
            logger.error("login-item ensureEnabled failed: \(String(describing: error), privacy: .public)")
        }
        let currentStatus = loginItem.status()
        if currentStatus != .enabled {
            logger.info("login-item status after ensureEnabled: \(String(describing: currentStatus), privacy: .public)")
        }

        onboardingVM.onAuthenticated = { [weak self] _ in
            guard let self else { return }
            self.authPhase = .authenticated
            // Fire an immediate flush so the device registers with the
            // backend instead of waiting up to one flushInterval.
            Task { _ = await self.syncClient.flush() }
        }

        startPeriodicFlush()
    }

    /// Wipes JWT + device id + device token and flips the menubar UI back
    /// to `OnboardingView`. The collector keeps running — events recorded
    /// while signed out remain in `LocalStore` and will sync once the user
    /// signs in again (the device gets re-registered on the next flush).
    func signOut() async {
        do {
            try await authClient.signOut()
        } catch {
            logger.error("sign-out failed: \(error.localizedDescription, privacy: .public)")
        }
        authPhase = .unauthenticated
    }

    private func startPeriodicFlush() {
        flushTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: UInt64(Self.flushInterval * 1_000_000_000))
                guard let self else { return }
                let outcome = await self.syncClient.flush()
                if case .completed(let synced) = outcome, synced > 0 {
                    self.logger.debug("periodic flush synced \(synced) events")
                }
            }
        }
    }

    deinit {
        flushTask?.cancel()
    }

    private static func databasePath() -> String {
        // swiftlint:disable:next force_unwrapping
        let support = FileManager.default.urls(
            for: .applicationSupportDirectory,
            in: .userDomainMask
        ).first!
        let dir = support.appendingPathComponent("MacAgent", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir.appendingPathComponent("data.db").path
    }

    /// Hardware-anchored fingerprint sent to `/v1/devices/register`. Backend
    /// keys idempotency on `(account, fingerprint)` so re-registering the
    /// same Mac collapses to a token rotation instead of a duplicate row.
    /// Falls back to a process-stable UUID when IOKit is unavailable.
    private static func deviceFingerprint() -> String {
        if let uuid = ioPlatformUUID() {
            return "macos-\(uuid)"
        }
        return "macos-\(UUID().uuidString)"
    }

    private static func ioPlatformUUID() -> String? {
        let entry = IOServiceGetMatchingService(
            kIOMainPortDefault,
            IOServiceMatching("IOPlatformExpertDevice")
        )
        guard entry != 0 else { return nil }
        defer { IOObjectRelease(entry) }
        guard let raw = IORegistryEntryCreateCFProperty(
            entry,
            kIOPlatformUUIDKey as CFString,
            kCFAllocatorDefault,
            0
        ) else { return nil }
        return (raw.takeRetainedValue() as? String)
    }

    /// Production base URL. Overridable via `MACAGENT_BACKEND_URL` so a
    /// developer can point a debug build at a local backend without
    /// rebuilding.
    static let defaultBaseURL: URL = {
        if let raw = ProcessInfo.processInfo.environment["MACAGENT_BACKEND_URL"],
           let url = URL(string: raw) {
            return url
        }
        // swiftlint:disable:next force_unwrapping
        return URL(string: "https://macos-screentime-backend.fly.dev")!
    }()
}
