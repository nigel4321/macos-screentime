import XCTest
import PolicyEngine
@testable import LocalStore

final class LocalStoreTests: XCTestCase {

    var database: AppDatabase!

    override func setUpWithError() throws {
        database = try AppDatabase()
    }

    // MARK: - §1.7 Migration

    func testFreshDatabaseHasV1Schema() throws {
        // If migration fails, setUp() throws and the test is marked as an error.
        XCTAssertNotNil(database)
    }

    func testMigrationIsIdempotent() throws {
        // Running a second migration against the same schema should be a no-op.
        XCTAssertNoThrow(try AppDatabase())
    }

    // MARK: - UsageEventDAO

    func testInsertAndFetchUnsynced() throws {
        let dao = UsageEventDAO(database: database)
        let event = UsageEvent(
            bundleID: "com.example.A",
            start: Date(timeIntervalSince1970: 1_000_000),
            end: Date(timeIntervalSince1970: 1_003_600)
        )
        try dao.insert(event)

        let rows = try dao.fetchUnsynced()
        XCTAssertEqual(rows.count, 1)
        XCTAssertEqual(rows[0].event.bundleID, BundleID("com.example.A"))
    }

    func testMarkSyncedRemovesFromUnsynced() throws {
        let dao = UsageEventDAO(database: database)
        try dao.insert(UsageEvent(
            bundleID: "com.example.A",
            start: Date(timeIntervalSince1970: 1_000_000),
            end: Date(timeIntervalSince1970: 1_003_600)
        ))

        let before = try dao.fetchUnsynced()
        XCTAssertEqual(before.count, 1)

        try dao.markSynced(ids: before.map(\.id), at: Date())

        XCTAssertTrue(try dao.fetchUnsynced().isEmpty)
    }

    func testFetchUnsyncedExcludesSyncedEvents() throws {
        let dao = UsageEventDAO(database: database)
        try dao.insert(UsageEvent(
            bundleID: "com.example.A",
            start: Date(timeIntervalSince1970: 1_000_000),
            end: Date(timeIntervalSince1970: 1_001_800)
        ))
        try dao.insert(UsageEvent(
            bundleID: "com.example.B",
            start: Date(timeIntervalSince1970: 1_001_800),
            end: Date(timeIntervalSince1970: 1_003_600)
        ))

        let rows = try dao.fetchUnsynced()
        XCTAssertEqual(rows.count, 2)

        try dao.markSynced(ids: [rows[0].id], at: Date())

        let remaining = try dao.fetchUnsynced()
        XCTAssertEqual(remaining.count, 1)
        XCTAssertEqual(remaining[0].event.bundleID, rows[1].event.bundleID)
    }

    func testMarkSyncedWithEmptyIdsIsNoOp() throws {
        let dao = UsageEventDAO(database: database)
        XCTAssertNoThrow(try dao.markSynced(ids: [], at: Date()))
    }

    // MARK: - PolicyDAO

    func testReadFromEmptyDatabaseReturnsNil() throws {
        XCTAssertNil(try PolicyDAO(database: database).read())
    }

    func testWriteThenReadRoundTrips() throws {
        let dao = PolicyDAO(database: database)
        let policy = Policy(
            version: PolicyVersion(1),
            appLimits: [AppLimit(bundleID: "com.example.A", dailyLimit: 3_600)],
            blockList: ["com.example.B"]
        )
        try dao.write(policy, receivedAt: Date())

        let read = try dao.read()
        XCTAssertEqual(read?.version, PolicyVersion(1))
        XCTAssertEqual(read?.appLimits.first?.bundleID, BundleID("com.example.A"))
        XCTAssertEqual(read?.blockList, [BundleID("com.example.B")])
    }

    func testReadReturnsLatestVersion() throws {
        let dao = PolicyDAO(database: database)
        try dao.write(Policy(version: PolicyVersion(1)), receivedAt: Date())
        try dao.write(Policy(version: PolicyVersion(2)), receivedAt: Date())

        XCTAssertEqual(try dao.read()?.version, PolicyVersion(2))
    }
}
