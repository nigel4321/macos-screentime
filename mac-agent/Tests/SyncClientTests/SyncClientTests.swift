import XCTest
import LocalStore
import PolicyEngine
@testable import SyncClient

final class SyncClientTests: XCTestCase {

    var session: URLSession!
    var credentials: InMemoryCredentialStore!
    var database: AppDatabase!
    var dao: UsageEventDAO!

    override func setUpWithError() throws {
        try super.setUpWithError()
        MockURLProtocol.reset()
        session = URLSession.mocked()
        credentials = InMemoryCredentialStore()
        database = try AppDatabase()
        dao = UsageEventDAO(database: database)
    }

    override func tearDown() {
        MockURLProtocol.reset()
        super.tearDown()
    }

    private func makeSyncClient(maxAttempts: Int = 4) -> SyncClient {
        // swiftlint:disable:next force_unwrapping
        SyncClient(baseURL: URL(string: "https://example.test")!,
                   credentials: credentials,
                   dao: dao,
                   fingerprint: "fp-1",
                   resolver: FakeAppMetadataResolver(),
                   session: session,
                   backoff: Backoff(base: 0.001, cap: 0.001, random: { _ in 0.001 }),
                   maxAttempts: maxAttempts,
                   sleepNanos: { _ in })
    }

    func testFlushReturnsNoCredentialsWhenNotSignedIn() async {
        let outcome = await makeSyncClient().flush()
        XCTAssertEqual(outcome, .noCredentials)
        XCTAssertEqual(MockURLProtocol.capturedRequests.count, 0,
                       "must not hit the network without a JWT")
    }

    func testFlushCompletesEndToEnd() async throws {
        try credentials.writeJWT("jwt-abc")
        try dao.insert(UsageEvent(
            bundleID: BundleID("com.example.A"),
            start: Date(timeIntervalSince1970: 1_700_000_000),
            end: Date(timeIntervalSince1970: 1_700_000_005)
        ))
        let unsynced = try dao.fetchUnsynced()
        XCTAssertEqual(unsynced.count, 1)
        let cid = unsynced[0].clientEventID

        // First request: registers device. Second request: uploads batch.
        nonisolated(unsafe) var calls = 0
        MockURLProtocol.requestHandler = { request in
            calls += 1
            if request.url?.path == "/v1/devices/register" {
                let response = HTTPURLResponse(url: request.url!, statusCode: 200,
                                               httpVersion: nil, headerFields: nil)!
                return (response, Data(#"{"device_id":"dev-1","device_token":"tok-1"}"#.utf8))
            }
            if request.url?.path == "/v1/usage:batchUpload" {
                let response = HTTPURLResponse(url: request.url!, statusCode: 200,
                                               httpVersion: nil, headerFields: nil)!
                return (response, Data(#"{"results":[{"client_event_id":"\#(cid)","status":"accepted"}]}"#.utf8))
            }
            XCTFail("unexpected path \(String(describing: request.url?.path))")
            throw URLError(.badURL)
        }

        let outcome = await makeSyncClient().flush()
        XCTAssertEqual(outcome, .completed(synced: 1))
        XCTAssertEqual(calls, 2)
        XCTAssertTrue(try dao.fetchUnsynced().isEmpty)
    }

    func testFlushRetriesOn5xxThenSucceeds() async throws {
        try credentials.writeJWT("jwt-abc")
        try credentials.writeDeviceID("dev-1")
        try credentials.writeDeviceToken("tok-1")
        try dao.insert(UsageEvent(
            bundleID: BundleID("com.example.A"),
            start: Date(timeIntervalSince1970: 1_700_000_000),
            end: Date(timeIntervalSince1970: 1_700_000_005)
        ))
        let cid = try dao.fetchUnsynced()[0].clientEventID

        nonisolated(unsafe) var calls = 0
        MockURLProtocol.requestHandler = { request in
            calls += 1
            if calls == 1 {
                let response = HTTPURLResponse(url: request.url!, statusCode: 503,
                                               httpVersion: nil, headerFields: nil)!
                return (response, Data())
            }
            let response = HTTPURLResponse(url: request.url!, statusCode: 200,
                                           httpVersion: nil, headerFields: nil)!
            return (response, Data(#"{"results":[{"client_event_id":"\#(cid)","status":"accepted"}]}"#.utf8))
        }

        let outcome = await makeSyncClient().flush()
        XCTAssertEqual(outcome, .completed(synced: 1))
        XCTAssertEqual(calls, 2, "must retry 503 once before succeeding")
    }

    func testFlushGivesUpAfterMaxAttempts() async throws {
        try credentials.writeJWT("jwt-abc")
        try credentials.writeDeviceID("dev-1")
        try credentials.writeDeviceToken("tok-1")
        try dao.insert(UsageEvent(
            bundleID: BundleID("com.example.A"),
            start: Date(timeIntervalSince1970: 1_700_000_000),
            end: Date(timeIntervalSince1970: 1_700_000_005)
        ))

        nonisolated(unsafe) var calls = 0
        MockURLProtocol.requestHandler = { request in
            calls += 1
            let response = HTTPURLResponse(url: request.url!, statusCode: 503,
                                           httpVersion: nil, headerFields: nil)!
            return (response, Data())
        }

        let outcome = await makeSyncClient(maxAttempts: 3).flush()
        if case .gaveUp(let lastError) = outcome {
            if case .serverError = lastError { /* ok */ } else {
                XCTFail("expected serverError, got \(lastError)")
            }
        } else {
            XCTFail("expected gaveUp, got \(outcome)")
        }
        XCTAssertEqual(calls, 3, "must attempt exactly maxAttempts before giving up")
    }

    func testFlushDoesNotRetry401() async throws {
        try credentials.writeJWT("jwt-abc")
        try credentials.writeDeviceID("dev-1")
        try credentials.writeDeviceToken("tok-1")
        try dao.insert(UsageEvent(
            bundleID: BundleID("com.example.A"),
            start: Date(timeIntervalSince1970: 1_700_000_000),
            end: Date(timeIntervalSince1970: 1_700_000_005)
        ))

        nonisolated(unsafe) var calls = 0
        MockURLProtocol.requestHandler = { request in
            calls += 1
            let response = HTTPURLResponse(url: request.url!, statusCode: 401,
                                           httpVersion: nil, headerFields: nil)!
            return (response, Data())
        }

        let outcome = await makeSyncClient().flush()
        XCTAssertEqual(outcome, .gaveUp(lastError: .unauthorized))
        XCTAssertEqual(calls, 1, "401 must not retry")
    }

    func testShouldRetryClassification() {
        XCTAssertTrue(SyncClient.shouldRetry(.serverError(status: 500, body: "")))
        XCTAssertTrue(SyncClient.shouldRetry(.transport(message: "")))
        XCTAssertFalse(SyncClient.shouldRetry(.unauthorized))
        XCTAssertFalse(SyncClient.shouldRetry(.clientError(status: 400, body: "")))
        XCTAssertFalse(SyncClient.shouldRetry(.decoding(message: "")))
        XCTAssertFalse(SyncClient.shouldRetry(.missingCredentials))
    }
}
