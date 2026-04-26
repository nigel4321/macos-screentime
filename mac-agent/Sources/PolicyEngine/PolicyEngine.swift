import Foundation

/// Pure-function evaluator: given the usage recorded for a device, the
/// current policy, and the current clock reading, return the set of
/// enforcement actions that should be applied.
///
/// The engine is deliberately free of Apple-framework imports. The TDD
/// suite in `PolicyEngineTests` exercises every branch.
public struct PolicyEngine {
    public init() {}

    /// - Parameters:
    ///   - usage: All usage events the engine may consider. The engine
    ///     scopes them to the relevant window internally.
    ///   - policy: Current policy, or `nil` if none has been received yet.
    ///   - now: The current instant, injected for determinism.
    ///   - calendar: The calendar to use for "today" boundaries and
    ///     day-of-week resolution. Carries the relevant `TimeZone`.
    /// - Returns: Zero or more enforcement actions. The set is
    ///   idempotent — applying it repeatedly has the same effect as
    ///   applying it once.
    public func evaluate(
        usage: [UsageEvent],
        policy: Policy?,
        now: Date,
        calendar: Calendar
    ) -> [EnforcementAction] {
        guard let policy else { return [] }

        var actions: [EnforcementAction] = []

        // Per-app daily limits: accumulate today's foreground time per app.
        let todayStart = calendar.startOfDay(for: now)
        for limit in policy.appLimits {
            let todayDuration = usage
                .filter { $0.bundleID == limit.bundleID && $0.start >= todayStart }
                .reduce(0.0) { $0 + $1.duration }
            if todayDuration >= limit.dailyLimit {
                actions.append(.shield(limit.bundleID))
            }
        }

        // Downtime windows: if now falls inside any window, shield every blocked app.
        if !policy.blockList.isEmpty {
            let inDowntime = policy.downtimeWindows.contains { isActive(window: $0, at: now, calendar: calendar) }
            if inDowntime {
                actions += policy.blockList.map { .shield($0) }
            }
        }

        return actions
    }

    // Returns true when `now` falls within the window's active period.
    // End boundary is exclusive ([start, end)); crossing-midnight windows
    // cover [start, midnight) on the named day and [midnight, end) on the
    // following day.
    private func isActive(window: DowntimeWindow, at now: Date, calendar: Calendar) -> Bool {
        let comps = calendar.dateComponents([.weekday, .hour, .minute, .second], from: now)
        guard
            let weekday = comps.weekday,
            let today = DayOfWeek(rawValue: weekday)
        else { return false }

        let secondOfDay = (comps.hour ?? 0) * 3_600
            + (comps.minute ?? 0) * 60
            + (comps.second ?? 0)

        if window.startSecondOfDay < window.endSecondOfDay {
            // Normal window — does not cross midnight.
            return window.daysOfWeek.contains(today)
                && secondOfDay >= window.startSecondOfDay
                && secondOfDay < window.endSecondOfDay
        } else {
            // Crosses midnight: evening portion belongs to the named day,
            // morning portion belongs to the following day (yesterday was named).
            let yesterdayRaw = weekday == 1 ? 7 : weekday - 1
            guard let yesterday = DayOfWeek(rawValue: yesterdayRaw) else { return false }
            let eveningActive = window.daysOfWeek.contains(today) && secondOfDay >= window.startSecondOfDay
            let morningActive = window.daysOfWeek.contains(yesterday) && secondOfDay < window.endSecondOfDay
            return eveningActive || morningActive
        }
    }
}
