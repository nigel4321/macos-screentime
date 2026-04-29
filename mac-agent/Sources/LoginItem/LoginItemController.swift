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
    /// We respect this and never silently re-enable it.
    case requiresApproval
    /// Registry returned a status we don't model. Treat as not-enabled.
    case unknown
}

/// Errors surfaced by `LoginItemController`. Underlying registry errors
/// are wrapped so callers don't pull in `ServiceManagement`.
public enum LoginItemError: Error, Sendable {
    case registryFailed(String)
}

/// Persistence seam for the "user explicitly disabled this" flag. We use
/// it to avoid silently re-registering after the user opts out, since the
/// auto-register path runs on every launch.
public protocol LoginItemPreferenceStore: Sendable {
    /// Has `enableOnFirstLaunchIfNeeded` already run? Set to true after
    /// the first attempt regardless of outcome — we register exactly once
    /// without explicit user action, then leave it to the toggle.
    func didAutoRegister() -> Bool
    func recordAutoRegistered()
    /// True if the user previously toggled the item off. We never
    /// auto-register over an explicit disable.
    func userDisabled() -> Bool
    func recordUserDisabled()
    func clearUserDisabled()
}

/// Production-friendly default backed by `UserDefaults.standard`. Marked
/// `@unchecked Sendable` because `UserDefaults` is documented as
/// thread-safe but isn't formally Sendable-typed.
public struct UserDefaultsLoginItemPreferenceStore: LoginItemPreferenceStore, @unchecked Sendable {
    private let defaults: UserDefaults
    private let autoRegisterKey = "loginitem.auto_registered"
    private let userDisabledKey = "loginitem.user_disabled"

    public init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    public func didAutoRegister() -> Bool { defaults.bool(forKey: autoRegisterKey) }
    public func recordAutoRegistered() { defaults.set(true, forKey: autoRegisterKey) }
    public func userDisabled() -> Bool { defaults.bool(forKey: userDisabledKey) }
    public func recordUserDisabled() { defaults.set(true, forKey: userDisabledKey) }
    public func clearUserDisabled() { defaults.removeObject(forKey: userDisabledKey) }
}

/// The seam that wraps `SMAppService.mainApp`. Production uses the real
/// service; tests use `InMemoryLoginItemRegistry` so they don't need a
/// proper `.app` bundle context.
public protocol LoginItemRegistry: Sendable {
    func currentStatus() -> LoginItemStatus
    func register() throws
    func unregister() throws
}

/// Coordinates auto-registration on first launch and the menubar toggle.
/// Stateless beyond what the injected registry + preference store hold.
public struct LoginItemController: Sendable {
    public let registry: LoginItemRegistry
    public let preferences: LoginItemPreferenceStore

    public init(registry: LoginItemRegistry, preferences: LoginItemPreferenceStore) {
        self.registry = registry
        self.preferences = preferences
    }

    /// Convenience constructor wiring the production `SMAppService`-backed
    /// registry and `UserDefaults` preference store.
    #if canImport(ServiceManagement)
    public static func makeDefault() -> LoginItemController {
        LoginItemController(
            registry: SMAppServiceLoginItemRegistry(),
            preferences: UserDefaultsLoginItemPreferenceStore()
        )
    }
    #endif

    public func status() -> LoginItemStatus {
        registry.currentStatus()
    }

    /// Called from app launch. Registers the app exactly once, ever, and
    /// only when the user hasn't previously disabled it. Subsequent
    /// launches are a no-op so we never override the user's choice.
    public func enableOnFirstLaunchIfNeeded() throws {
        if preferences.didAutoRegister() { return }
        if preferences.userDisabled() { return }

        do {
            try registry.register()
            preferences.recordAutoRegistered()
        } catch {
            // Record the attempt so a misbehaving SMAppService doesn't
            // make us try forever. The toggle remains available for a
            // manual retry.
            preferences.recordAutoRegistered()
            throw LoginItemError.registryFailed(String(describing: error))
        }
    }

    /// Called by the menubar toggle. Wraps register/unregister and tracks
    /// the user-disabled flag so first-launch auto-register can respect it.
    public func setEnabled(_ enabled: Bool) throws {
        if enabled {
            do { try registry.register() } catch {
                throw LoginItemError.registryFailed(String(describing: error))
            }
            preferences.clearUserDisabled()
        } else {
            do { try registry.unregister() } catch {
                throw LoginItemError.registryFailed(String(describing: error))
            }
            preferences.recordUserDisabled()
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

    public func unregister() throws {
        guard #available(macOS 13, *) else {
            throw LoginItemError.registryFailed("macOS 13+ required for SMAppService")
        }
        try SMAppService.mainApp.unregister()
    }
}
#endif

/// In-memory registry for unit tests. The real `SMAppService` only does
/// anything inside a properly-launched `.app` context, so swift-test
/// substitutes this fake.
public final class InMemoryLoginItemRegistry: LoginItemRegistry, @unchecked Sendable {
    public private(set) var status: LoginItemStatus
    public var registerError: Error?
    public var unregisterError: Error?
    public private(set) var registerCalls = 0
    public private(set) var unregisterCalls = 0
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

    public func unregister() throws {
        lock.lock(); defer { lock.unlock() }
        unregisterCalls += 1
        if let error = unregisterError { throw error }
        status = .notRegistered
    }

    public func setStatus(_ status: LoginItemStatus) {
        lock.lock(); defer { lock.unlock() }
        self.status = status
    }
}

/// In-memory preference store for unit tests.
public final class InMemoryLoginItemPreferenceStore: LoginItemPreferenceStore, @unchecked Sendable {
    private var auto = false
    private var disabled = false
    private let lock = NSLock()

    public init(autoRegistered: Bool = false, userDisabled: Bool = false) {
        self.auto = autoRegistered
        self.disabled = userDisabled
    }

    public func didAutoRegister() -> Bool {
        lock.lock(); defer { lock.unlock() }
        return auto
    }
    public func recordAutoRegistered() {
        lock.lock(); defer { lock.unlock() }
        auto = true
    }
    public func userDisabled() -> Bool {
        lock.lock(); defer { lock.unlock() }
        return disabled
    }
    public func recordUserDisabled() {
        lock.lock(); defer { lock.unlock() }
        disabled = true
    }
    public func clearUserDisabled() {
        lock.lock(); defer { lock.unlock() }
        disabled = false
    }
}
