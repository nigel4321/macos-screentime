import Foundation
import GRDB

struct PolicyRow: Codable, FetchableRecord, PersistableRecord {
    static let databaseTableName = "policy"

    var version: Int64
    var bodyJson: String
    var receivedAt: Int64
    var appliedAt: Int64?

    enum CodingKeys: String, CodingKey {
        case version
        case bodyJson   = "body_json"
        case receivedAt = "received_at"
        case appliedAt  = "applied_at"
    }
}
