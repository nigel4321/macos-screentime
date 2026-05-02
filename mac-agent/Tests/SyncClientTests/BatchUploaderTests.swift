import XCTest
import LocalStore
import PolicyEngine
@testable import SyncClient

final class BatchUploaderTests: XCTestCase {

    var session: URLSession!
    var credentials: InMemoryCredentialStore!
    var api: APIClient!
    var database: AppDatabase!
    var dao: UsageEventDAO!
    var registrar: DeviceRegistrar!

    override func setUpWithError() throws {
        try super.setUpWithError()
        MockURLProtocol.reset()
        session = URLSession.mocked()
        credentials = InMemoryCredentialStore(
            jwt: "jwt-abc",
            deviceID: "dev-1",
            deviceToken: "tok-xyz"
        )
        // swiftlint:disable:next force_unwrapping
        api = APIClient(baseURL: URL(string: "https://example.test")!,
                        credentials: credentials,
                        session: session)
        database = try AppDatabase()
        dao = UsageEventDAO(database: database)
        registrar = DeviceRegistrar(api: api, credentials: credentials, fingerprint: "fp-1")
    }

    override func tearDown() {
        MockURLProtocol.reset()
        super.tearDown()
    }

    private func seed(_ count: Int) throws -> [String] {
        for i in 0..<count {
            try dao.insert(UsageEvent(
                bundleID: BundleID("com.example.\(i)"),
                start: Date(timeIntervalSince1970: TimeInterval(1_700_000_000 + i * 10)),
                end: Date(timeIntervalSince1970: TimeInterval(1_700_000_005 + i * 10))
            ))
        }
        return try dao.fetchUnsynced().map(\.clientEventID)
    }

    func testFlushUploadsUnsyncedAndMarksAccepted() async throws {
        let ids = try seed(3)

        MockURLProtocol.requestHandler = { _ in
            let body = ids.map { #"{"client_event_id":"\#($0)","status":"accepted"}"# }.joined(separator: ",")
            let payload = #"{"results":[\#(body)]}"#
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/usage:batchUpload")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(payload.utf8))
        }

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar, resolver: FakeAppMetadataResolver())
        let synced = try await uploader.flush()
        XCTAssertEqual(synced, 3)
        XCTAssertTrue(try dao.fetchUnsynced().isEmpty)
    }

    func testDuplicateAndRejectedBothMarkedSynced() async throws {
        let ids = try seed(3)

        MockURLProtocol.requestHandler = { _ in
            let payload = """
            {"results":[
                {"client_event_id":"\(ids[0])","status":"accepted"},
                {"client_event_id":"\(ids[1])","status":"duplicate"},
                {"client_event_id":"\(ids[2])","status":"rejected","reason":"started_at outside accepted window"}
            ]}
            """
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/usage:batchUpload")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(payload.utf8))
        }

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar, resolver: FakeAppMetadataResolver())
        let synced = try await uploader.flush()
        // accepted + duplicate + rejected all mark synced — rejected is
        // permanent so we don't want it re-queued forever.
        XCTAssertEqual(synced, 3)
        XCTAssertTrue(try dao.fetchUnsynced().isEmpty)
    }

    func testFlushSendsExpectedHeadersAndBody() async throws {
        let ids = try seed(1)

        MockURLProtocol.requestHandler = { _ in
            let payload = #"{"results":[{"client_event_id":"\#(ids[0])","status":"accepted"}]}"#
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/usage:batchUpload")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(payload.utf8))
        }

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar, resolver: FakeAppMetadataResolver())
        _ = try await uploader.flush()

        let req = try XCTUnwrap(MockURLProtocol.capturedRequests.first)
        XCTAssertEqual(req.httpMethod, "POST")
        XCTAssertEqual(req.url?.path, "/v1/usage:batchUpload")
        XCTAssertEqual(req.value(forHTTPHeaderField: "Authorization"), "Bearer jwt-abc")
        XCTAssertEqual(req.value(forHTTPHeaderField: "X-Device-Token"), "tok-xyz")

        let body = try XCTUnwrap(req.httpBody)
        let json = try JSONSerialization.jsonObject(with: body) as? [String: Any]
        let events = try XCTUnwrap(json?["events"] as? [[String: Any]])
        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0]["client_event_id"] as? String, ids[0])
        XCTAssertEqual(events[0]["bundle_id"] as? String, "com.example.0")
        XCTAssertNotNil(events[0]["started_at"])
        XCTAssertNotNil(events[0]["ended_at"])
    }

    func testFlushAttachesAppMetadataForResolvedBundles() async throws {
        // Two distinct bundle ids; resolver knows one and not the other.
        // Expect the request to carry only the resolved entry.
        try dao.insert(UsageEvent(
            bundleID: BundleID("com.google.Chrome"),
            start: Date(timeIntervalSince1970: 1_700_000_000),
            end: Date(timeIntervalSince1970: 1_700_000_005)
        ))
        try dao.insert(UsageEvent(
            bundleID: BundleID("com.unknown.app"),
            start: Date(timeIntervalSince1970: 1_700_000_010),
            end: Date(timeIntervalSince1970: 1_700_000_015)
        ))
        let ids = try dao.fetchUnsynced().map(\.clientEventID)

        MockURLProtocol.requestHandler = { _ in
            let body = ids.map { #"{"client_event_id":"\#($0)","status":"accepted"}"# }.joined(separator: ",")
            let payload = #"{"results":[\#(body)]}"#
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/usage:batchUpload")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(payload.utf8))
        }

        let resolver = FakeAppMetadataResolver(names: ["com.google.Chrome": "Google Chrome"])
        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar, resolver: resolver)
        _ = try await uploader.flush()

        let req = try XCTUnwrap(MockURLProtocol.capturedRequests.first)
        let body = try XCTUnwrap(req.httpBody)
        let json = try XCTUnwrap(try JSONSerialization.jsonObject(with: body) as? [String: Any])

        let appMetadata = try XCTUnwrap(json["app_metadata"] as? [String: String])
        XCTAssertEqual(appMetadata, ["com.google.Chrome": "Google Chrome"])
    }

    func testFlushDeduplicatesRepeatedBundleIDs() async throws {
        // Three events sharing one bundle id should produce a single
        // app_metadata entry, not three duplicates.
        for offset in stride(from: 0, to: 30, by: 10) {
            try dao.insert(UsageEvent(
                bundleID: BundleID("com.tinyspeck.slackmacgap"),
                start: Date(timeIntervalSince1970: TimeInterval(1_700_000_000 + offset)),
                end: Date(timeIntervalSince1970: TimeInterval(1_700_000_005 + offset))
            ))
        }
        let ids = try dao.fetchUnsynced().map(\.clientEventID)

        MockURLProtocol.requestHandler = { _ in
            let body = ids.map { #"{"client_event_id":"\#($0)","status":"accepted"}"# }.joined(separator: ",")
            let payload = #"{"results":[\#(body)]}"#
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/usage:batchUpload")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(payload.utf8))
        }

        let resolver = FakeAppMetadataResolver(names: ["com.tinyspeck.slackmacgap": "Slack"])
        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar, resolver: resolver)
        _ = try await uploader.flush()

        let req = try XCTUnwrap(MockURLProtocol.capturedRequests.first)
        let body = try XCTUnwrap(req.httpBody)
        let json = try XCTUnwrap(try JSONSerialization.jsonObject(with: body) as? [String: Any])

        let appMetadata = try XCTUnwrap(json["app_metadata"] as? [String: String])
        XCTAssertEqual(appMetadata, ["com.tinyspeck.slackmacgap": "Slack"])
        // Resolver was consulted only once thanks to in-page dedupe.
        XCTAssertEqual(resolver.capturedLookups, ["com.tinyspeck.slackmacgap"])
    }

    func testFlushOmitsAppMetadataWhenNothingResolves() async throws {
        // Resolver returns the bundle id itself for every lookup — that's
        // the "no display name available" signal. The request must not
        // include an `app_metadata` field at all (omitempty on the wire),
        // so the backend doesn't write entries with name == bundle id.
        let ids = try seed(2)

        MockURLProtocol.requestHandler = { _ in
            let body = ids.map { #"{"client_event_id":"\#($0)","status":"accepted"}"# }.joined(separator: ",")
            let payload = #"{"results":[\#(body)]}"#
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/usage:batchUpload")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(payload.utf8))
        }

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar, resolver: FakeAppMetadataResolver())
        _ = try await uploader.flush()

        let req = try XCTUnwrap(MockURLProtocol.capturedRequests.first)
        let body = try XCTUnwrap(req.httpBody)
        let json = try XCTUnwrap(try JSONSerialization.jsonObject(with: body) as? [String: Any])
        XCTAssertNil(json["app_metadata"], "no resolved names → field must be omitted")
    }

    func testEmptyQueueIsNoop() async throws {
        // No seed; flush must not call the network.
        MockURLProtocol.requestHandler = { _ in
            XCTFail("network must not be called with empty queue")
            throw URLError(.badURL)
        }

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar, resolver: FakeAppMetadataResolver())
        let synced = try await uploader.flush()
        XCTAssertEqual(synced, 0)
        XCTAssertEqual(MockURLProtocol.capturedRequests.count, 0)
    }
}
