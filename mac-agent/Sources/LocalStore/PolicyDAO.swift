import Foundation
import GRDB
import PolicyEngine

public struct PolicyDAO {
    private let dbQueue: DatabaseQueue

    public init(database: AppDatabase) {
        dbQueue = database.dbQueue
    }

    /// Returns the highest-version policy stored locally, or `nil` if none exists.
    public func read() throws -> Policy? {
        try dbQueue.read { db in
            guard let row = try PolicyRow.order(Column("version").desc).fetchOne(db) else { return nil }
            let data = Data(row.bodyJson.utf8)
            return try JSONDecoder().decode(Policy.self, from: data)
        }
    }

    /// Persists a policy snapshot. Each version is a separate row; the
    /// schema primary key prevents duplicate versions.
    public func write(_ policy: Policy, receivedAt: Date) throws {
        let data = try JSONEncoder().encode(policy)
        guard let json = String(data: data, encoding: .utf8) else { return }
        let row = PolicyRow(
            version: Int64(policy.version.value),
            bodyJson: json,
            receivedAt: Int64(receivedAt.timeIntervalSince1970),
            appliedAt: nil
        )
        try dbQueue.write { db in
            try row.insert(db, onConflict: .replace)
        }
    }
}
