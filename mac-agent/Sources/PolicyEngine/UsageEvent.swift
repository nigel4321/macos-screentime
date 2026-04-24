import Foundation

/// A closed interval during which a given app was the foreground application.
/// Events are append-only and immutable once recorded.
public struct UsageEvent: Hashable, Codable, Sendable {
    public let bundleID: BundleID
    public let start: Date
    public let end: Date

    public init(bundleID: BundleID, start: Date, end: Date) {
        self.bundleID = bundleID
        self.start = start
        self.end = end
    }

    public var duration: TimeInterval {
        end.timeIntervalSince(start)
    }
}
