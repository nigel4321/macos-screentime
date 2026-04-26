import Foundation
import Observation
import LocalStore
import PolicyEngine

struct AppUsageSummary: Identifiable {
    let id: BundleID
    let bundleID: BundleID
    let duration: TimeInterval

    var displayName: String { bundleID.value }

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

    init(dao: UsageEventDAO, calendar: Calendar = .current) {
        self.dao = dao
        self.calendar = calendar
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
            .map { AppUsageSummary(id: $0.key, bundleID: $0.key, duration: $0.value) }
            .sorted { $0.duration > $1.duration }
            .prefix(5)
            .map { $0 }
    }
}
