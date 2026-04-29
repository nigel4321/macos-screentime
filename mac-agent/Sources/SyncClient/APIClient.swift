import Foundation

/// Errors produced by `APIClient`. Surfaced verbatim to callers so each can
/// decide whether to retry, prompt the user, or drop the work.
public enum APIError: Error, Equatable {
    /// 401 from the server — the JWT is missing/expired/invalid. The caller
    /// is expected to clear the JWT and re-prompt the user for sign-in.
    case unauthorized
    /// 4xx other than 401 — request was malformed; do not retry blindly.
    case clientError(status: Int, body: String)
    /// 5xx — server problem; safe to retry with backoff.
    case serverError(status: Int, body: String)
    /// Transport-level failure (network, TLS, timeout). Safe to retry.
    case transport(message: String)
    /// JSON decode/encode failure for a request or response body.
    case decoding(message: String)
    /// SyncClient was asked to call an endpoint without a JWT in the
    /// credential store. Caller should sign in first.
    case missingCredentials
}

/// Strongly-typed wrapper around `URLSession` that knows how to send a
/// `Codable` body, attach the `Authorization` and `X-Device-Token` headers,
/// and decode the response into a `Codable` value. Auth headers are pulled
/// from `CredentialStore` on every call so a sign-out elsewhere takes effect
/// without a restart.
public final class APIClient: Sendable {
    public let baseURL: URL
    private let session: URLSession
    private let credentials: CredentialStore
    private let encoder: JSONEncoder
    private let decoder: JSONDecoder

    public init(
        baseURL: URL,
        credentials: CredentialStore,
        session: URLSession = .shared
    ) {
        self.baseURL = baseURL
        self.session = session
        self.credentials = credentials

        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        self.encoder = encoder

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        self.decoder = decoder
    }

    /// Send a JSON `POST`/`GET` request and decode the response. The
    /// `requireDeviceToken` flag picks which auth headers to attach: device
    /// registration only needs the JWT, but `usage:batchUpload` also needs
    /// the device token.
    public func send<Request: Encodable, Response: Decodable>(
        method: String,
        path: String,
        body: Request?,
        requireDeviceToken: Bool,
        responseType: Response.Type = Response.self
    ) async throws -> Response {
        let url = baseURL.appendingPathComponent(path)
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")

        guard let jwt = try credentials.readJWT(), !jwt.isEmpty else {
            throw APIError.missingCredentials
        }
        request.setValue("Bearer \(jwt)", forHTTPHeaderField: "Authorization")

        if requireDeviceToken {
            guard let token = try credentials.readDeviceToken(), !token.isEmpty else {
                throw APIError.missingCredentials
            }
            request.setValue(token, forHTTPHeaderField: "X-Device-Token")
        }

        if let body {
            do {
                request.httpBody = try encoder.encode(body)
            } catch {
                throw APIError.decoding(message: "encode request: \(error)")
            }
        }

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch {
            throw APIError.transport(message: error.localizedDescription)
        }
        guard let http = response as? HTTPURLResponse else {
            throw APIError.transport(message: "non-HTTP response")
        }

        switch http.statusCode {
        case 200..<300:
            do {
                return try decoder.decode(Response.self, from: data)
            } catch {
                throw APIError.decoding(message: "decode response: \(error)")
            }
        case 401:
            throw APIError.unauthorized
        case 400..<500:
            throw APIError.clientError(status: http.statusCode, body: bodyString(data))
        default:
            throw APIError.serverError(status: http.statusCode, body: bodyString(data))
        }
    }

    private func bodyString(_ data: Data) -> String {
        String(data: data, encoding: .utf8) ?? ""
    }
}

/// Empty body marker for requests/responses that carry no payload.
public struct EmptyBody: Codable {
    public init() {}
}
