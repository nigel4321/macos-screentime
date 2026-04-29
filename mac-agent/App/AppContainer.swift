import Foundation
import IOKit
import LocalStore
import LoginItem
import SyncClient
import UsageCollector
import os

/// Owns every long-lived dependency and wires them together.
/// Created once at app launch and torn down on quit.
@MainActor
final class AppContainer {
    let todayViewModel: TodayViewModel

    private let usageEventDAO: UsageEventDAO
    private let source: NSWorkspaceSource
    private let collector: UsageCollector
    private let syncClient: SyncClient
    let loginItem: LoginItemController
    private let logger = Logger(subsystem: "com.macos-screentime.MacAgent", category: "AppContainer")

    /// Periodic flush cadence. 60s is the smallest cadence that meaningfully
    /// lowers data loss on unexpected quit while keeping wakeup pressure low.
    private static let flushInterval: TimeInterval = 60

    private var flushTask: Task<Void, Never>?

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
        syncClient = SyncClient(
            baseURL: baseURL,
            credentials: KeychainCredentialStore(),
            dao: dao,
            fingerprint: Self.deviceFingerprint()
        )
        loginItem = LoginItemController.makeDefault()
        // Auto-register exactly once, ever. Failures here are non-fatal —
        // the menubar toggle is the recovery path.
        do {
            try loginItem.enableOnFirstLaunchIfNeeded()
        } catch {
            logger.error("login-item auto-register failed: \(String(describing: error), privacy: .public)")
        }
        startPeriodicFlush()
    }

    /// Closes any open usage event and pushes one final batch to the server.
    /// Called on app quit; the await is bounded by the SyncClient's own
    /// `maxAttempts` × `backoff.cap`, so quit cannot hang indefinitely.
    func flush() async {
        collector.flush()
        let outcome = await syncClient.flush()
        switch outcome {
        case .completed(let synced) where synced > 0:
            logger.info("on-quit flush synced \(synced) events")
        case .gaveUp(let lastError):
            logger.error("on-quit flush gave up: \(String(describing: lastError), privacy: .public)")
        default:
            break
        }
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
