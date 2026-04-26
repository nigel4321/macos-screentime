import Foundation
import PolicyEngine

/// Translates a stream of `WorkspaceEvent`s into closed `UsageEvent` intervals.
///
/// State machine:
/// - On `appActivated`: close the previous interval (if any), open a new one.
/// - On `systemWillSleep` / `screensDidSleep`: close the current interval.
/// - On `systemDidWake`: wait for the next `appActivated` to start tracking.
/// - `flush()`: close the current open interval using the injected clock — call
///   this on graceful app quit.
public final class UsageCollector {
    private var activeApp: (bundleID: BundleID, since: Date)?
    private let clock: () -> Date
    private let onEvent: (UsageEvent) -> Void

    /// - Parameters:
    ///   - source: The workspace-event source to subscribe to.
    ///   - clock: Returns the current instant. Injected for determinism; defaults to `Date()`.
    ///   - onEvent: Called with each completed `UsageEvent`.
    public init(
        source: WorkspaceSource,
        clock: @escaping () -> Date = Date.init,
        onEvent: @escaping (UsageEvent) -> Void
    ) {
        self.clock = clock
        self.onEvent = onEvent
        source.subscribe { [weak self] event in self?.handle(event) }
    }

    /// Closes the current open interval at the clock's current time.
    /// Call on graceful app quit to avoid losing the tail of a session.
    public func flush() {
        closeActive(at: clock())
    }

    private func handle(_ event: WorkspaceEvent) {
        switch event {
        case .appActivated(let bundleID, let at):
            closeActive(at: at)
            activeApp = (bundleID: bundleID, since: at)
        case .systemWillSleep(let at), .screensDidSleep(let at):
            closeActive(at: at)
        case .systemDidWake:
            break
        }
    }

    private func closeActive(at end: Date) {
        guard let active = activeApp else { return }
        onEvent(UsageEvent(bundleID: active.bundleID, start: active.since, end: end))
        activeApp = nil
    }
}
