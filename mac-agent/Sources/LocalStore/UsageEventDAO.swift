import Foundation
import GRDB
import PolicyEngine

/// One unsynced usage row plus the metadata SyncClient needs to upload it:
/// the local row id (for `markSynced`) and the stable client-generated event
/// id that backend's idempotency contract is keyed on.
public struct UnsyncedUsageEvent: Equatable {
    public let id: Int64
    public let clientEventID: String
    public let event: UsageEvent

    public init(id: Int64, clientEventID: String, event: UsageEvent) {
        self.id = id
        self.clientEventID = clientEventID
        self.event = event
    }
}

public struct UsageEventDAO {
    private let dbQueue: DatabaseQueue

    public init(database: AppDatabase) {
        dbQueue = database.dbQueue
    }

    /// Appends a usage event. The row id is assigned by SQLite; the
    /// client-event id is generated locally and is what backend uses for
    /// idempotent dedup on `(device_id, client_event_id, started_at)`.
    public func insert(_ event: UsageEvent) throws {
        var row = UsageEventRow(event: event)
        try dbQueue.write { db in try row.insert(db) }
    }

    /// Returns all events that have not yet been pushed to the backend,
    /// each tagged with its row id and stable `client_event_id`.
    public func fetchUnsynced() throws -> [UnsyncedUsageEvent] {
        try dbQueue.read { db in
            try UsageEventRow
                .filter(Column("synced_at") == nil)
                .fetchAll(db)
                .compactMap { row in
                    guard let id = row.id else { return nil }
                    return UnsyncedUsageEvent(
                        id: id,
                        clientEventID: row.clientEventId,
                        event: row.toUsageEvent()
                    )
                }
        }
    }

    /// Returns all events whose start timestamp is on or after `startDate`.
    public func fetch(since startDate: Date) throws -> [UsageEvent] {
        let threshold = Int64(startDate.timeIntervalSince1970)
        return try dbQueue.read { db in
            try UsageEventRow
                .filter(Column("started_at") >= threshold)
                .fetchAll(db)
                .map { $0.toUsageEvent() }
        }
    }

    /// Stamps the given rows with a sync timestamp so they are excluded
    /// from future `fetchUnsynced` calls.
    public func markSynced(ids: [Int64], at date: Date) throws {
        guard !ids.isEmpty else { return }
        let timestamp = Int64(date.timeIntervalSince1970)
        try dbQueue.write { db in
            _ = try UsageEventRow
                .filter(ids.contains(Column("id")))
                .updateAll(db, Column("synced_at").set(to: timestamp))
        }
    }
}
