import Foundation

/// The complete set of restrictions in force for a device at a given version.
/// A `Policy` is plain data — evaluation is performed by `PolicyEngine`.
public struct Policy: Hashable, Codable, Sendable {
    public let version: PolicyVersion
    public let appLimits: [AppLimit]
    public let downtimeWindows: [DowntimeWindow]
    public let blockList: [BundleID]

    public init(
        version: PolicyVersion,
        appLimits: [AppLimit] = [],
        downtimeWindows: [DowntimeWindow] = [],
        blockList: [BundleID] = []
    ) {
        self.version = version
        self.appLimits = appLimits
        self.downtimeWindows = downtimeWindows
        self.blockList = blockList
    }

    public static let empty = Policy(version: .zero)
}

/// A per-app daily cap. `dailyLimit` is total foreground time allowed
/// per local-calendar day before the app is shielded.
public struct AppLimit: Hashable, Codable, Sendable {
    public let bundleID: BundleID
    public let dailyLimit: TimeInterval

    public init(bundleID: BundleID, dailyLimit: TimeInterval) {
        self.bundleID = bundleID
        self.dailyLimit = dailyLimit
    }
}

/// A recurring time window during which every app on the block list is
/// shielded. Times are seconds since local midnight, letting the engine
/// resolve them against an injected `Calendar` / `TimeZone`. A window
/// may cross midnight (`endSecondOfDay <= startSecondOfDay`).
public struct DowntimeWindow: Hashable, Codable, Sendable {
    public let startSecondOfDay: Int
    public let endSecondOfDay: Int
    public let daysOfWeek: Set<DayOfWeek>

    public init(
        startSecondOfDay: Int,
        endSecondOfDay: Int,
        daysOfWeek: Set<DayOfWeek>
    ) {
        self.startSecondOfDay = startSecondOfDay
        self.endSecondOfDay = endSecondOfDay
        self.daysOfWeek = daysOfWeek
    }
}

/// Day of the week numbered to match `Calendar.Component.weekday`
/// (1 = Sunday). This keeps conversions to / from `Calendar` trivial.
public enum DayOfWeek: Int, Hashable, Codable, Sendable, CaseIterable {
    case sunday = 1
    case monday
    case tuesday
    case wednesday
    case thursday
    case friday
    case saturday
}
