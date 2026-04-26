import XCTest
@testable import PolicyEngine

/// RED-GREEN-REFACTOR tests across ROADMAP §1.4 (no-policy base case),
/// §1.5 (per-app daily limits), and §1.6 (downtime windows).
final class PolicyEngineTests: XCTestCase {

    // MARK: - Fixtures

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

    static let nyCalendar: Calendar = {
        var cal = Calendar(identifier: .gregorian)
        // swiftlint:disable:next force_unwrapping
        cal.timeZone = TimeZone(identifier: "America/New_York")!
        return cal
    }()

    /// Builds a Date from explicit components. The components' timezone overrides
    /// the calendar's timezone, so the returned Date is the absolute instant
    /// corresponding to the given local time.
    private static func makeDate(
        year: Int, month: Int, day: Int, hour: Int,
        minute: Int = 0, timezone: String = "UTC"
    ) -> Date {
        var comps = DateComponents()
        comps.year = year
        comps.month = month
        comps.day = day
        comps.hour = hour
        comps.minute = minute
        comps.timeZone = TimeZone(identifier: timezone)
        // swiftlint:disable:next force_unwrapping
        return utcCalendar.date(from: comps)!
    }

    // MARK: - §1.4 No-policy case

    func testNilPolicyReturnsEmpty() {
        XCTAssertTrue(
            PolicyEngine().evaluate(
                usage: [], policy: nil, now: Self.fixedNow, calendar: Self.utcCalendar
            ).isEmpty
        )
    }

    func testEmptyPolicyReturnsEmpty() {
        let event = UsageEvent(
            bundleID: "com.example.A",
            start: Self.fixedNow.addingTimeInterval(-3_600),
            end: Self.fixedNow
        )
        XCTAssertTrue(
            PolicyEngine().evaluate(
                usage: [event], policy: .empty, now: Self.fixedNow, calendar: Self.utcCalendar
            ).isEmpty
        )
    }

    // MARK: - §1.5 Per-app daily limits

    func testLimitNotCrossedNoShield() {
        let policy = Policy(version: .zero, appLimits: [AppLimit(bundleID: "com.example.A", dailyLimit: 3_600)])
        let event = UsageEvent(
            bundleID: "com.example.A",
            start: Self.fixedNow.addingTimeInterval(-1_800),
            end: Self.fixedNow
        )
        XCTAssertFalse(
            PolicyEngine().evaluate(
                usage: [event], policy: policy, now: Self.fixedNow, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }

    func testLimitExactlyCrossedEmitsShield() {
        let policy = Policy(version: .zero, appLimits: [AppLimit(bundleID: "com.example.A", dailyLimit: 3_600)])
        let event = UsageEvent(
            bundleID: "com.example.A",
            start: Self.fixedNow.addingTimeInterval(-3_600),
            end: Self.fixedNow
        )
        XCTAssertTrue(
            PolicyEngine().evaluate(
                usage: [event], policy: policy, now: Self.fixedNow, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }

    func testYesterdayDataAloneDoesNotTripTodayLimit() {
        let policy = Policy(version: .zero, appLimits: [AppLimit(bundleID: "com.example.A", dailyLimit: 3_600)])
        let yesterday = Self.fixedNow.addingTimeInterval(-86_400)
        let event = UsageEvent(
            bundleID: "com.example.A",
            start: yesterday.addingTimeInterval(-7_200),
            end: yesterday
        )
        XCTAssertFalse(
            PolicyEngine().evaluate(
                usage: [event], policy: policy, now: Self.fixedNow, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }

    func testMultipleAppsWithIndependentLimits() {
        let policy = Policy(version: .zero, appLimits: [
            AppLimit(bundleID: "com.example.A", dailyLimit: 3_600),
            AppLimit(bundleID: "com.example.B", dailyLimit: 3_600)
        ])
        let usageA = UsageEvent(
            bundleID: "com.example.A",
            start: Self.fixedNow.addingTimeInterval(-4_000),
            end: Self.fixedNow
        )
        let usageB = UsageEvent(
            bundleID: "com.example.B",
            start: Self.fixedNow.addingTimeInterval(-1_800),
            end: Self.fixedNow
        )
        let actions = PolicyEngine().evaluate(
            usage: [usageA, usageB],
            policy: policy,
            now: Self.fixedNow,
            calendar: Self.utcCalendar
        )
        XCTAssertTrue(actions.contains(.shield("com.example.A")))
        XCTAssertFalse(actions.contains(.shield("com.example.B")))
    }

    // MARK: - §1.6 Downtime windows

    func testOutsideWindowNoShield() {
        // fixedNow is 12:00 UTC; window is 22:00–23:00 UTC on Fridays.
        let policy = Policy(
            version: .zero,
            downtimeWindows: [DowntimeWindow(
                startSecondOfDay: 79_200, endSecondOfDay: 82_800, daysOfWeek: [.friday]
            )],
            blockList: ["com.example.A"]
        )
        XCTAssertFalse(
            PolicyEngine().evaluate(
                usage: [], policy: policy, now: Self.fixedNow, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }

    func testInsideWindowShieldsBlockList() {
        // fixedNow is 12:00 UTC; window is 11:00–13:00 UTC on Fridays.
        let policy = Policy(
            version: .zero,
            downtimeWindows: [DowntimeWindow(
                startSecondOfDay: 39_600, endSecondOfDay: 46_800, daysOfWeek: [.friday]
            )],
            blockList: ["com.example.A"]
        )
        XCTAssertTrue(
            PolicyEngine().evaluate(
                usage: [], policy: policy, now: Self.fixedNow, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }

    func testWindowCrossingMidnightEveningPortion() {
        // fixedNow + 11.5 h = 2026-04-24 23:30 UTC (Friday evening).
        let fridayNight = Self.fixedNow.addingTimeInterval(41_400)
        let policy = Policy(
            version: .zero,
            downtimeWindows: [DowntimeWindow(
                startSecondOfDay: 82_800, endSecondOfDay: 3_600, daysOfWeek: [.friday]
            )],
            blockList: ["com.example.A"]
        )
        XCTAssertTrue(
            PolicyEngine().evaluate(
                usage: [], policy: policy, now: fridayNight, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }

    func testWindowCrossingMidnightMorningPortion() {
        // fixedNow + 12.5 h = 2026-04-25 00:30 UTC (Saturday early morning).
        let saturdayEarly = Self.fixedNow.addingTimeInterval(45_000)
        let policy = Policy(
            version: .zero,
            downtimeWindows: [DowntimeWindow(
                startSecondOfDay: 82_800, endSecondOfDay: 3_600, daysOfWeek: [.friday]
            )],
            blockList: ["com.example.A"]
        )
        XCTAssertTrue(
            PolicyEngine().evaluate(
                usage: [], policy: policy, now: saturdayEarly, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }

    func testDSTSpringForward() {
        // 2026-03-08: US Eastern clocks spring forward at 02:00 → 03:00.
        // Window 01:30–02:30 NY local (5_400–9_000 s) on Sundays.
        // The hour 02:xx never exists in local time on this day.
        let policy = Policy(
            version: .zero,
            downtimeWindows: [DowntimeWindow(
                startSecondOfDay: 5_400, endSecondOfDay: 9_000, daysOfWeek: [.sunday]
            )],
            blockList: ["com.example.A"]
        )
        let inside = Self.makeDate(year: 2026, month: 3, day: 8, hour: 1, minute: 45,
                                   timezone: "America/New_York")
        XCTAssertTrue(
            PolicyEngine().evaluate(usage: [], policy: policy, now: inside, calendar: Self.nyCalendar)
                .contains(.shield("com.example.A")),
            "01:45 EST is inside the window"
        )
        let outside = Self.makeDate(year: 2026, month: 3, day: 8, hour: 3, minute: 15,
                                    timezone: "America/New_York")
        XCTAssertFalse(
            PolicyEngine().evaluate(usage: [], policy: policy, now: outside, calendar: Self.nyCalendar)
                .contains(.shield("com.example.A")),
            "03:15 EDT (after spring-forward; 02:xx never existed) is outside the window"
        )
    }

    func testDSTFallBack() {
        // 2026-11-01: US Eastern clocks fall back at 02:00 → 01:00.
        // Window 01:30–02:30 NY local (5_400–9_000 s) on Sundays.
        // 01:xx exists twice; both occurrences should be inside the window.
        let policy = Policy(
            version: .zero,
            downtimeWindows: [DowntimeWindow(
                startSecondOfDay: 5_400, endSecondOfDay: 9_000, daysOfWeek: [.sunday]
            )],
            blockList: ["com.example.A"]
        )
        // First 01:45 EDT (UTC-4) = 2026-11-01T05:45Z
        let firstOccurrence = Self.makeDate(year: 2026, month: 11, day: 1, hour: 1, minute: 45,
                                            timezone: "America/New_York")
        XCTAssertTrue(
            PolicyEngine().evaluate(usage: [], policy: policy, now: firstOccurrence, calendar: Self.nyCalendar)
                .contains(.shield("com.example.A")),
            "First 01:45 EDT is inside the window"
        )
        // Second 01:45 EST (UTC-5) = 2026-11-01T06:45Z — one real hour later
        let secondOccurrence = firstOccurrence.addingTimeInterval(3_600)
        XCTAssertTrue(
            PolicyEngine().evaluate(usage: [], policy: policy, now: secondOccurrence, calendar: Self.nyCalendar)
                .contains(.shield("com.example.A")),
            "Second 01:45 EST (after fall-back) is also inside the window"
        )
    }

    // Property: window is active iff now ∈ [startSecondOfDay, endSecondOfDay)

    func testWindowBoundaryStartInclusive() {
        // now is exactly at the window's start second (10:00:00 UTC).
        let exactStart = Self.makeDate(year: 2026, month: 4, day: 24, hour: 10)
        let policy = Policy(
            version: .zero,
            downtimeWindows: [DowntimeWindow(
                startSecondOfDay: 36_000, endSecondOfDay: 43_200, daysOfWeek: [.friday]
            )],
            blockList: ["com.example.A"]
        )
        XCTAssertTrue(
            PolicyEngine().evaluate(
                usage: [], policy: policy, now: exactStart, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }

    func testWindowBoundaryEndExclusive() {
        // fixedNow is exactly 12:00:00 UTC — the window's end second (exclusive).
        let policy = Policy(
            version: .zero,
            downtimeWindows: [DowntimeWindow(
                startSecondOfDay: 36_000, endSecondOfDay: 43_200, daysOfWeek: [.friday]
            )],
            blockList: ["com.example.A"]
        )
        XCTAssertFalse(
            PolicyEngine().evaluate(
                usage: [], policy: policy, now: Self.fixedNow, calendar: Self.utcCalendar
            ).contains(.shield("com.example.A"))
        )
    }
}
