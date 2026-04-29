import Foundation
#if canImport(ServiceManagement)
import ServiceManagement
#endif

/// Status of the app's login-item registration. Mirrors the subset of
/// `SMAppService.Status` we care about; the production registry maps the
/// real enum into this one so the controller doesn't depend on
/// `ServiceManagement` directly.
public enum LoginItemStatus: Equatable, Sendable {
    /// macOS will launch the app at login.
    case enabled
    /// Not registered — either never registered, or unregistered.
    case notRegistered
    /// User disabled the item in System Settings → General → Login Items.
    /// macOS keeps the user's choice authoritative; calling `register()`
    /// from the app will not flip this back to `.enabled` until the user
    /// re-approves in System Settings. We log and move on.
    case requiresApproval
    /// Registry returned a status we don't model. Treat as not-enabled.
    case unknown
}

/// Errors surfaced by `LoginItemController`. Underlying registry errors
/// are wrapped so callers don't pull in `ServiceManagement`.
public enum LoginItemError: Error, Sendable {
    case registryFailed(String)
}

/// The seam that wraps `SMAppService.mainApp`. Production uses the real
/// service; tests use `InMemoryLoginItemRegistry` so they don't need a
/// proper `.app` bundle context.
public protocol LoginItemRegistry: Sendable {
    func currentStatus() -> LoginItemStatus
    func register() throws
}

/// Drives launch-at-login. By design there is no in-app opt-out: the agent
/// only collects events while it's running, so we re-register on every
/// launch and let macOS handle the rest. The user's only way to disable it
/// is **System Settings → General → Login Items**, which macOS keeps
/// authoritative — `register()` cannot override an explicit user disable
/// (status stays `.requiresApproval` until the user re-approves there).
public struct LoginItemController: Sendable {
    public let registry: LoginItemRegistry

    public init(registry: LoginItemRegistry) {
        self.registry = registry
    }

    /// Convenience constructor wiring the production `SMAppService`-backed
    /// registry.
    #if canImport(ServiceManagement)
    public static func makeDefault() -> LoginItemController {
        LoginItemController(registry: SMAppServiceLoginItemRegistry())
    }
    #endif

    public func status() -> LoginItemStatus {
        registry.currentStatus()
    }

    /// Idempotent — call on every launch. No-op when already enabled,
    /// otherwise registers. Throws `LoginItemError.registryFailed` only on
    /// unexpected `SMAppService` failures; a `.requiresApproval` outcome
    /// (user disabled it via System Settings) is *not* an error here, but
    /// the post-call `status()` will reflect it so the caller can log.
    public func ensureEnabled() throws {
        if registry.currentStatus() == .enabled { return }
        do {
            try registry.register()
        } catch {
            throw LoginItemError.registryFailed(String(describing: error))
        }
    }
}

#if canImport(ServiceManagement)
/// Production-only implementation of `LoginItemRegistry` backed by
/// `SMAppService.mainApp`. Marked `@available(macOS 13, *)` because the
/// API gates on that — but our deployment target is 14.0 so the guard is
/// effectively a no-op at runtime.
public struct SMAppServiceLoginItemRegistry: LoginItemRegistry {
    public init() {}

    public func currentStatus() -> LoginItemStatus {
        guard #available(macOS 13, *) else { return .unknown }
        switch SMAppService.mainApp.status {
        case .enabled:
            return .enabled
        case .notRegistered, .notFound:
            return .notRegistered
        case .requiresApproval:
            return .requiresApproval
        @unknown default:
            return .unknown
        }
    }

    public func register() throws {
        guard #available(macOS 13, *) else {
            throw LoginItemError.registryFailed("macOS 13+ required for SMAppService")
        }
        try SMAppService.mainApp.register()
    }
}
#endif

/// In-memory registry for unit tests. The real `SMAppService` only does
/// anything inside a properly-launched `.app` context, so swift-test
/// substitutes this fake.
public final class InMemoryLoginItemRegistry: LoginItemRegistry, @unchecked Sendable {
    public private(set) var status: LoginItemStatus
    public var registerError: Error?
    public private(set) var registerCalls = 0
    private let lock = NSLock()

    public init(initialStatus: LoginItemStatus = .notRegistered) {
        self.status = initialStatus
    }

    public func currentStatus() -> LoginItemStatus {
        lock.lock(); defer { lock.unlock() }
        return status
    }

    public func register() throws {
        lock.lock(); defer { lock.unlock() }
        registerCalls += 1
        if let error = registerError { throw error }
        status = .enabled
    }

    public func setStatus(_ status: LoginItemStatus) {
        lock.lock(); defer { lock.unlock() }
        self.status = status
    }
}
