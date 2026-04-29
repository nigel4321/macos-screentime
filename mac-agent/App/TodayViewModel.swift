import Foundation
import Observation
import AppMetadata
import LocalStore
import PolicyEngine

struct AppUsageSummary: Identifiable {
    let id: BundleID
    let bundleID: BundleID
    let displayName: String
    let duration: TimeInterval

    var formattedDuration: String {
        let hours   = Int(duration) / 3_600
        let minutes = (Int(duration) % 3_600) / 60
        return hours > 0 ? "\(hours)h \(minutes)m" : "\(minutes)m"
    }
}

@Observable
@MainActor
final class TodayViewModel {
    private(set) var topApps: [AppUsageSummary] = []

    private let dao: UsageEventDAO
    private let calendar: Calendar
    private let resolver: AppMetadataResolver

    init(
        dao: UsageEventDAO,
        calendar: Calendar = .current,
        resolver: AppMetadataResolver = SystemAppMetadataResolver()
    ) {
        self.dao = dao
        self.calendar = calendar
        self.resolver = resolver
        refresh()
    }

    func refresh() {
        let todayStart = calendar.startOfDay(for: Date())
        guard let events = try? dao.fetch(since: todayStart) else { return }

        var totals: [BundleID: TimeInterval] = [:]
        for event in events {
            totals[event.bundleID, default: 0] += event.duration
        }

        topApps = totals
            .map { bundleID, duration in
                AppUsageSummary(
                    id: bundleID,
                    bundleID: bundleID,
                    displayName: resolver.displayName(for: bundleID),
                    duration: duration
                )
            }
            .sorted { $0.duration > $1.duration }
            .prefix(5)
            .map { $0 }
    }
}
