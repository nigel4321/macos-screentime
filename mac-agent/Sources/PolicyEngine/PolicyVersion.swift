import Foundation

/// A monotonic version assigned by the sync backend. The client with the
/// higher version wins; ties never occur because the server owns the
/// counter.
public struct PolicyVersion: Hashable, Codable, Sendable, Comparable {
    public let value: UInt64

    public init(_ value: UInt64) {
        self.value = value
    }

    public static let zero = PolicyVersion(0)

    public static func < (lhs: PolicyVersion, rhs: PolicyVersion) -> Bool {
        lhs.value < rhs.value
    }
}
