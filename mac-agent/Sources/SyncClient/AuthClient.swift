import Foundation

/// Exchanges third-party identity tokens for a backend JWT and persists the
/// JWT in `CredentialStore`. The Mac onboarding flow obtains an Apple
/// identity token via `ASAuthorizationController` (driven by the SwiftUI
/// `SignInWithAppleButton`) and hands it to `signInWithApple(identityToken:)`.
///
/// `signOut()` is colocated here so the sign-in/sign-out lifecycle lives in
/// one place; clearing credentials is a one-liner over `CredentialStore.clear`.
public actor AuthClient {
    private let api: APIClient
    private let credentials: CredentialStore

    public init(api: APIClient, credentials: CredentialStore) {
        self.api = api
        self.credentials = credentials
    }

    private struct Request: Encodable {
        // swiftlint:disable identifier_name
        let id_token: String
        // swiftlint:enable identifier_name
    }

    private struct Response: Decodable {
        // swiftlint:disable identifier_name
        let jwt: String
        let account_id: String
        // swiftlint:enable identifier_name
    }

    @discardableResult
    public func signInWithApple(identityToken: String) async throws -> Authenticated {
        let response: Response = try await api.send(
            method: "POST",
            path: "v1/auth/apple",
            body: Request(id_token: identityToken),
            requireDeviceToken: false,
            requireJWT: false
        )
        try credentials.writeJWT(response.jwt)
        return Authenticated(jwt: response.jwt, accountID: response.account_id)
    }

    public func signOut() throws {
        try credentials.clear()
    }
}

public struct Authenticated: Equatable, Sendable {
    public let jwt: String
    public let accountID: String

    public init(jwt: String, accountID: String) {
        self.jwt = jwt
        self.accountID = accountID
    }
}
