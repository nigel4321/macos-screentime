import Foundation
import PolicyEngine

/// Domain events emitted by a `WorkspaceSource`. Each case carries the
/// wall-clock time it occurred so the collector can produce accurate intervals.
public enum WorkspaceEvent {
    case appActivated(bundleID: BundleID, at: Date)
    case systemWillSleep(at: Date)
    case systemDidWake(at: Date)
    case screensDidSleep(at: Date)
}
