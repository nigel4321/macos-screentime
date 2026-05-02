import XCTest
@testable import SyncClient

final class AuthClientTests: XCTestCase {

    var session: URLSession!
    var credentials: InMemoryCredentialStore!
    var api: APIClient!

    override func setUp() {
        super.setUp()
        MockURLProtocol.reset()
        session = URLSession.mocked()
        credentials = InMemoryCredentialStore()
        // swiftlint:disable:next force_unwrapping
        api = APIClient(baseURL: URL(string: "https://example.test")!,
                        credentials: credentials,
                        session: session)
    }

    override func tearDown() {
        MockURLProtocol.reset()
        super.tearDown()
    }

    func testSignInWithAppleExchangesTokenAndPersistsJWT() async throws {
        MockURLProtocol.requestHandler = { _ in
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/auth/apple")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(#"{"jwt":"backend-jwt-1","account_id":"acc-1"}"#.utf8))
        }

        let client = AuthClient(api: api, credentials: credentials)
        let result = try await client.signInWithApple(identityToken: "apple-id-token")

        XCTAssertEqual(result, Authenticated(jwt: "backend-jwt-1", accountID: "acc-1"))
        XCTAssertEqual(try credentials.readJWT(), "backend-jwt-1")

        // Outgoing request shape
        let req = MockURLProtocol.capturedRequests.first
        XCTAssertEqual(req?.httpMethod, "POST")
        XCTAssertEqual(req?.url?.path, "/v1/auth/apple")
        // sign-in must NOT attach an Authorization header — there is no JWT
        // yet, and the server mounts /v1/auth/apple outside its
        // Authenticator group.
        XCTAssertNil(req?.value(forHTTPHeaderField: "Authorization"))
        XCTAssertNil(req?.value(forHTTPHeaderField: "X-Device-Token"))

        let body = try XCTUnwrap(req?.httpBody)
        let json = try JSONSerialization.jsonObject(with: body) as? [String: String]
        XCTAssertEqual(json?["id_token"], "apple-id-token")
    }

    func testSignInPropagatesUnauthorized() async {
        MockURLProtocol.requestHandler = { _ in
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/auth/apple")!,
                                           statusCode: 401, httpVersion: nil, headerFields: nil)!
            return (response, Data())
        }

        let client = AuthClient(api: api, credentials: credentials)
        do {
            _ = try await client.signInWithApple(identityToken: "bad-token")
            XCTFail("expected APIError.unauthorized")
        } catch let error as APIError {
            XCTAssertEqual(error, .unauthorized)
        } catch {
            XCTFail("unexpected error \(error)")
        }
        XCTAssertNil(try? credentials.readJWT())
    }

    func testSignInPropagatesServerError() async {
        MockURLProtocol.requestHandler = { _ in
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/auth/apple")!,
                                           statusCode: 503, httpVersion: nil, headerFields: nil)!
            return (response, Data("oops".utf8))
        }

        let client = AuthClient(api: api, credentials: credentials)
        do {
            _ = try await client.signInWithApple(identityToken: "tok")
            XCTFail("expected APIError.serverError")
        } catch let error as APIError {
            if case .serverError = error {} else {
                XCTFail("unexpected APIError \(error)")
            }
        } catch {
            XCTFail("unexpected error \(error)")
        }
        XCTAssertNil(try? credentials.readJWT())
    }

    func testSignOutClearsAllCredentials() async throws {
        try credentials.writeJWT("jwt-x")
        try credentials.writeDeviceID("dev-x")
        try credentials.writeDeviceToken("tok-x")

        let client = AuthClient(api: api, credentials: credentials)
        try await client.signOut()

        XCTAssertNil(try credentials.readJWT())
        XCTAssertNil(try credentials.readDeviceID())
        XCTAssertNil(try credentials.readDeviceToken())
    }
}
