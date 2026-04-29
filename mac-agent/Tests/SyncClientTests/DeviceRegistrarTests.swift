import XCTest
@testable import SyncClient

final class DeviceRegistrarTests: XCTestCase {

    var session: URLSession!
    var credentials: InMemoryCredentialStore!
    var api: APIClient!

    override func setUp() {
        super.setUp()
        MockURLProtocol.reset()
        session = URLSession.mocked()
        credentials = InMemoryCredentialStore(jwt: "jwt-abc")
        // swiftlint:disable:next force_unwrapping
        api = APIClient(baseURL: URL(string: "https://example.test")!,
                        credentials: credentials,
                        session: session)
    }

    override func tearDown() {
        MockURLProtocol.reset()
        super.tearDown()
    }

    func testRegisterPostsAndPersistsTokens() async throws {
        MockURLProtocol.requestHandler = { _ in
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/devices/register")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(#"{"device_id":"dev-1","device_token":"tok-1"}"#.utf8))
        }

        let registrar = DeviceRegistrar(api: api, credentials: credentials, fingerprint: "fp-1")
        let registered = try await registrar.register()
        XCTAssertEqual(registered, RegisteredDevice(id: "dev-1", token: "tok-1"))
        XCTAssertEqual(try credentials.readDeviceID(), "dev-1")
        XCTAssertEqual(try credentials.readDeviceToken(), "tok-1")

        // Body and method
        let req = MockURLProtocol.capturedRequests.first
        XCTAssertEqual(req?.httpMethod, "POST")
        XCTAssertEqual(req?.url?.path, "/v1/devices/register")
        let body = try XCTUnwrap(req?.httpBody)
        let json = try JSONSerialization.jsonObject(with: body) as? [String: String]
        XCTAssertEqual(json?["platform"], "macos")
        XCTAssertEqual(json?["fingerprint"], "fp-1")
    }

    func testRegisterIsIdempotentWhenCredentialsCached() async throws {
        try credentials.writeDeviceID("dev-1")
        try credentials.writeDeviceToken("tok-1")
        MockURLProtocol.requestHandler = { _ in
            XCTFail("network must not be called when device already registered")
            throw URLError(.badURL)
        }

        let registrar = DeviceRegistrar(api: api, credentials: credentials, fingerprint: "fp-1")
        let registered = try await registrar.register()
        XCTAssertEqual(registered, RegisteredDevice(id: "dev-1", token: "tok-1"))
        XCTAssertEqual(MockURLProtocol.capturedRequests.count, 0)
    }

    func testForceRegisterRotatesToken() async throws {
        try credentials.writeDeviceID("dev-1")
        try credentials.writeDeviceToken("tok-old")

        MockURLProtocol.requestHandler = { _ in
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/devices/register")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(#"{"device_id":"dev-1","device_token":"tok-new"}"#.utf8))
        }

        let registrar = DeviceRegistrar(api: api, credentials: credentials, fingerprint: "fp-1")
        let registered = try await registrar.register(force: true)
        XCTAssertEqual(registered.token, "tok-new")
        XCTAssertEqual(try credentials.readDeviceToken(), "tok-new")
    }
}
