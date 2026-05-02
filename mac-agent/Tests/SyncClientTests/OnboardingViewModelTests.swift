import XCTest
@testable import SyncClient

@MainActor
final class OnboardingViewModelTests: XCTestCase {

    var session: URLSession!
    var credentials: InMemoryCredentialStore!
    var api: APIClient!
    var auth: AuthClient!

    override func setUp() async throws {
        try await super.setUp()
        MockURLProtocol.reset()
        session = URLSession.mocked()
        credentials = InMemoryCredentialStore()
        // swiftlint:disable:next force_unwrapping
        api = APIClient(baseURL: URL(string: "https://example.test")!,
                        credentials: credentials,
                        session: session)
        auth = AuthClient(api: api, credentials: credentials)
    }

    override func tearDown() async throws {
        MockURLProtocol.reset()
        try await super.tearDown()
    }

    func testSignInTransitionsIdleLoadingIdleAndFiresCallback() async {
        MockURLProtocol.requestHandler = { _ in
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/auth/apple")!,
                                           statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(#"{"jwt":"jwt-1","account_id":"acc-1"}"#.utf8))
        }

        let vm = OnboardingViewModel(authClient: auth)
        var fired: Authenticated?
        vm.onAuthenticated = { fired = $0 }

        XCTAssertEqual(vm.phase, .idle)
        await vm.signIn(identityToken: "tok")

        XCTAssertEqual(vm.phase, .idle)
        XCTAssertEqual(fired, Authenticated(jwt: "jwt-1", accountID: "acc-1"))
    }

    func testUnauthorizedSurfacesAsErrorPhaseWithoutFiringCallback() async {
        MockURLProtocol.requestHandler = { _ in
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/auth/apple")!,
                                           statusCode: 401, httpVersion: nil, headerFields: nil)!
            return (response, Data())
        }

        let vm = OnboardingViewModel(authClient: auth)
        var fired = false
        vm.onAuthenticated = { _ in fired = true }

        await vm.signIn(identityToken: "bad")

        XCTAssertFalse(fired)
        if case .error(let msg) = vm.phase {
            XCTAssertTrue(msg.contains("Apple"))
        } else {
            XCTFail("expected .error phase, got \(vm.phase)")
        }
    }

    func testServerErrorSurfacesNetworkMessage() async {
        MockURLProtocol.requestHandler = { _ in
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/auth/apple")!,
                                           statusCode: 503, httpVersion: nil, headerFields: nil)!
            return (response, Data())
        }

        let vm = OnboardingViewModel(authClient: auth)
        await vm.signIn(identityToken: "tok")

        if case .error(let msg) = vm.phase {
            XCTAssertTrue(msg.contains("server") || msg.contains("connection"),
                          "expected server/connection wording, got \(msg)")
        } else {
            XCTFail("expected .error phase, got \(vm.phase)")
        }
    }

    func testDismissErrorReturnsToIdle() async {
        MockURLProtocol.requestHandler = { _ in
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/auth/apple")!,
                                           statusCode: 401, httpVersion: nil, headerFields: nil)!
            return (response, Data())
        }

        let vm = OnboardingViewModel(authClient: auth)
        await vm.signIn(identityToken: "bad")
        guard case .error = vm.phase else {
            XCTFail("expected .error, got \(vm.phase)")
            return
        }

        vm.dismissError()
        XCTAssertEqual(vm.phase, .idle)
    }

    func testRetryAfterErrorSucceeds() async {
        var responseQueue: [(Int, Data)] = [
            (503, Data()),
            (200, Data(#"{"jwt":"jwt-2","account_id":"acc-2"}"#.utf8))
        ]
        MockURLProtocol.requestHandler = { _ in
            let next = responseQueue.removeFirst()
            // swiftlint:disable:next force_unwrapping
            let response = HTTPURLResponse(url: URL(string: "https://example.test/v1/auth/apple")!,
                                           statusCode: next.0, httpVersion: nil, headerFields: nil)!
            return (response, next.1)
        }

        let vm = OnboardingViewModel(authClient: auth)
        var fired: Authenticated?
        vm.onAuthenticated = { fired = $0 }

        await vm.signIn(identityToken: "tok")
        guard case .error = vm.phase else {
            XCTFail("expected .error phase after first attempt")
            return
        }

        await vm.signIn(identityToken: "tok")
        XCTAssertEqual(vm.phase, .idle)
        XCTAssertEqual(fired, Authenticated(jwt: "jwt-2", accountID: "acc-2"))
    }
}
