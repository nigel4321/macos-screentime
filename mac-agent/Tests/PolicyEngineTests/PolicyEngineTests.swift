import XCTest
@testable import PolicyEngine

/// RED-GREEN-REFACTOR tests are added across ROADMAP §1.4 (no-policy
/// base case), §1.5 (per-app daily limits), and §1.6 (downtime windows).
/// This file currently holds only the test class and fixtures; each
/// sub-section adds its own `test…` methods.
final class PolicyEngineTests: XCTestCase {
    /// A fixed-instant calendar in a deterministic zone. Tests override
    /// `now` and, where they exercise DST, the time zone.
    static let utcCalendar: Calendar = {
        var calendar = Calendar(identifier: .gregorian)
        // swiftlint:disable:next force_unwrapping
        calendar.timeZone = TimeZone(identifier: "UTC")!
        return calendar
    }()

    /// 2026-04-24 12:00:00 UTC — a stable Friday inside a non-DST day.
    static let fixedNow: Date = {
        var components = DateComponents()
        components.year = 2026
        components.month = 4
        components.day = 24
        components.hour = 12
        components.timeZone = TimeZone(identifier: "UTC")
        // swiftlint:disable:next force_unwrapping
        return utcCalendar.date(from: components)!
    }()
}
