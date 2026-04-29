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

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar)
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

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar)
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

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar)
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

    func testEmptyQueueIsNoop() async throws {
        // No seed; flush must not call the network.
        MockURLProtocol.requestHandler = { _ in
            XCTFail("network must not be called with empty queue")
            throw URLError(.badURL)
        }

        let uploader = BatchUploader(api: api, dao: dao, registrar: registrar)
        let synced = try await uploader.flush()
        XCTAssertEqual(synced, 0)
        XCTAssertEqual(MockURLProtocol.capturedRequests.count, 0)
    }
}
