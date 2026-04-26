import Foundation
import GRDB

/// Owns the SQLite connection and runs all schema migrations.
/// Pass a file path for production; omit it to get an in-memory database (tests).
public struct AppDatabase {
    let dbQueue: DatabaseQueue

    public init(path: String) throws {
        dbQueue = try DatabaseQueue(path: path)
        try Self.migrate(dbQueue)
    }

    public init() throws {
        dbQueue = try DatabaseQueue()
        try Self.migrate(dbQueue)
    }

    private static func migrate(_ dbQueue: DatabaseQueue) throws {
        var migrator = DatabaseMigrator()
        migrator.registerMigration("v1") { db in
            try db.execute(sql: """
                CREATE TABLE usage_event (
                    id         INTEGER PRIMARY KEY AUTOINCREMENT,
                    bundle_id  TEXT    NOT NULL,
                    started_at INTEGER NOT NULL,
                    ended_at   INTEGER NOT NULL,
                    synced_at  INTEGER
                )
            """)
            try db.execute(sql: """
                CREATE INDEX idx_usage_unsynced
                    ON usage_event(synced_at)
                    WHERE synced_at IS NULL
            """)
            try db.execute(sql: """
                CREATE TABLE policy (
                    version     INTEGER PRIMARY KEY,
                    body_json   TEXT    NOT NULL,
                    received_at INTEGER NOT NULL,
                    applied_at  INTEGER
                )
            """)
        }
        try migrator.migrate(dbQueue)
    }
}
