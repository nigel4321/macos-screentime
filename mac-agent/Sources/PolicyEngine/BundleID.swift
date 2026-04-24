import Foundation

/// Strongly-typed wrapper around a macOS application bundle identifier
/// (e.g. `"com.apple.Safari"`). Wrapping the raw string prevents
/// accidental mixing with other string-typed ids.
public struct BundleID: Hashable, Codable, Sendable {
    public let value: String

    public init(_ value: String) {
        self.value = value
    }
}

extension BundleID: ExpressibleByStringLiteral {
    public init(stringLiteral value: String) {
        self.value = value
    }
}

extension BundleID: CustomStringConvertible {
    public var description: String { value }
}
