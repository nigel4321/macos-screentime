import XCTest
@testable import SyncClient

private struct EchoBody: Codable, Equatable {
    let value: String
}

final class APIClientTests: XCTestCase {

    var session: URLSession!
    var credentials: InMemoryCredentialStore!
    var client: APIClient!

    override func setUp() {
        super.setUp()
        MockURLProtocol.reset()
        session = URLSession.mocked()
        credentials = InMemoryCredentialStore(jwt: "jwt-abc", deviceToken: "tok-xyz")
        // swiftlint:disable:next force_unwrapping
        client = APIClient(baseURL: URL(string: "https://example.test")!,
                           credentials: credentials,
                           session: session)
    }

    override func tearDown() {
        MockURLProtocol.reset()
        super.tearDown()
    }

    func testSendAttachesBearerHeader() async throws {
        MockURLProtocol.requestHandler = { _ in
            let response = HTTPURLResponse(url: URL(string: "https://example.test/x")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(#"{"value":"ok"}"#.utf8))
        }

        let _: EchoBody = try await client.send(method: "GET", path: "x",
                                                body: EmptyBody?.none,
                                                requireDeviceToken: false)

        let req = MockURLProtocol.capturedRequests.first
        XCTAssertEqual(req?.value(forHTTPHeaderField: "Authorization"), "Bearer jwt-abc")
        XCTAssertNil(req?.value(forHTTPHeaderField: "X-Device-Token"))
    }

    func testSendAttachesDeviceTokenHeaderWhenRequested() async throws {
        MockURLProtocol.requestHandler = { _ in
            let response = HTTPURLResponse(url: URL(string: "https://example.test/x")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(#"{"value":"ok"}"#.utf8))
        }

        let _: EchoBody = try await client.send(method: "POST", path: "x",
                                                body: EchoBody(value: "hi"),
                                                requireDeviceToken: true)

        let req = MockURLProtocol.capturedRequests.first
        XCTAssertEqual(req?.value(forHTTPHeaderField: "Authorization"), "Bearer jwt-abc")
        XCTAssertEqual(req?.value(forHTTPHeaderField: "X-Device-Token"), "tok-xyz")
    }

    func testMissingJWTSurfacesAsMissingCredentials() async {
        try? credentials.clear()

        do {
            let _: EchoBody = try await client.send(method: "GET", path: "x",
                                                    body: EmptyBody?.none,
                                                    requireDeviceToken: false)
            XCTFail("expected APIError.missingCredentials")
        } catch let error as APIError {
            XCTAssertEqual(error, .missingCredentials)
        } catch {
            XCTFail("unexpected error \(error)")
        }
    }

    func testUnauthorizedMapsTo401() async {
        MockURLProtocol.requestHandler = { _ in
            let response = HTTPURLResponse(url: URL(string: "https://example.test/x")!,
                                           statusCode: 401, httpVersion: nil, headerFields: nil)!
            return (response, Data())
        }

        do {
            let _: EchoBody = try await client.send(method: "GET", path: "x",
                                                    body: EmptyBody?.none,
                                                    requireDeviceToken: false)
            XCTFail("expected APIError.unauthorized")
        } catch let error as APIError {
            XCTAssertEqual(error, .unauthorized)
        } catch {
            XCTFail("unexpected error \(error)")
        }
    }

    func testFiveHundredMapsToServerError() async {
        MockURLProtocol.requestHandler = { _ in
            let response = HTTPURLResponse(url: URL(string: "https://example.test/x")!,
                                           statusCode: 503, httpVersion: nil, headerFields: nil)!
            return (response, Data("oops".utf8))
        }

        do {
            let _: EchoBody = try await client.send(method: "GET", path: "x",
                                                    body: EmptyBody?.none,
                                                    requireDeviceToken: false)
            XCTFail("expected APIError.serverError")
        } catch let error as APIError {
            if case .serverError(let status, let body) = error {
                XCTAssertEqual(status, 503)
                XCTAssertEqual(body, "oops")
            } else {
                XCTFail("unexpected APIError \(error)")
            }
        } catch {
            XCTFail("unexpected error \(error)")
        }
    }
}
