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
        // v2: backend's batchUpload contract is idempotent on
        // (device_id, client_event_id, started_at). Each local event
        // gets a stable client-generated id at insert time so retries
        // collapse server-side into a single accepted row.
        migrator.registerMigration("v2_client_event_id") { db in
            // SQLite cannot ADD COLUMN with both NOT NULL and a non-constant
            // default, so we add the column nullable, backfill in-place,
            // then enforce uniqueness and rely on the DAO to never write NULL.
            try db.execute(sql: """
                ALTER TABLE usage_event ADD COLUMN client_event_id TEXT
            """)
            try db.execute(sql: """
                UPDATE usage_event
                   SET client_event_id = lower(hex(randomblob(16)))
                 WHERE client_event_id IS NULL
            """)
            try db.execute(sql: """
                CREATE UNIQUE INDEX idx_usage_client_event_id
                    ON usage_event(client_event_id)
            """)
        }
        try migrator.migrate(dbQueue)
    }
}
