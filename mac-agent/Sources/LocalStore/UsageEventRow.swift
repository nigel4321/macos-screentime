import Foundation
import GRDB
import PolicyEngine

struct UsageEventRow: Codable, FetchableRecord, MutablePersistableRecord {
    static let databaseTableName = "usage_event"

    var id: Int64?
    var bundleId: String
    var startedAt: Int64
    var endedAt: Int64
    var syncedAt: Int64?
    var clientEventId: String

    enum CodingKeys: String, CodingKey {
        case id
        case bundleId      = "bundle_id"
        case startedAt     = "started_at"
        case endedAt       = "ended_at"
        case syncedAt      = "synced_at"
        case clientEventId = "client_event_id"
    }

    mutating func didInsert(_ inserted: InsertionSuccess) {
        id = inserted.rowID
    }
}

extension UsageEventRow {
    init(event: UsageEvent, clientEventID: String = UUID().uuidString) {
        bundleId      = event.bundleID.value
        startedAt     = Int64(event.start.timeIntervalSince1970)
        endedAt       = Int64(event.end.timeIntervalSince1970)
        syncedAt      = nil
        clientEventId = clientEventID
    }

    func toUsageEvent() -> UsageEvent {
        UsageEvent(
            bundleID: BundleID(bundleId),
            start: Date(timeIntervalSince1970: Double(startedAt)),
            end: Date(timeIntervalSince1970: Double(endedAt))
        )
    }
}
