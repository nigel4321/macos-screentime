import Foundation
import GRDB
import PolicyEngine

public struct UsageEventDAO {
    private let dbQueue: DatabaseQueue

    public init(database: AppDatabase) {
        dbQueue = database.dbQueue
    }

    /// Appends a usage event. The row id is assigned by SQLite.
    public func insert(_ event: UsageEvent) throws {
        var row = UsageEventRow(event: event)
        try dbQueue.write { db in try row.insert(db) }
    }

    /// Returns all events that have not yet been pushed to the backend.
    public func fetchUnsynced() throws -> [(id: Int64, event: UsageEvent)] {
        try dbQueue.read { db in
            try UsageEventRow
                .filter(Column("synced_at") == nil)
                .fetchAll(db)
                .compactMap { row in
                    guard let id = row.id else { return nil }
                    return (id: id, event: row.toUsageEvent())
                }
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
