import Foundation

/// A single idempotent change to the enforcement state of one app.
/// `PolicyEngine.evaluate` returns a set of these; `Enforcer` applies
/// them against `ManagedSettingsStore`.
public enum EnforcementAction: Hashable, Sendable {
    case shield(BundleID)
    case clear(BundleID)

    public var bundleID: BundleID {
        switch self {
        case .shield(let id), .clear(let id):
            return id
        }
    }
}
