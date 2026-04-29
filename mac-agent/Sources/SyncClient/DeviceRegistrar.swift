import Foundation

/// Performs `POST /v1/devices/register` and persists the returned device id
/// + device token in `CredentialStore`. The backend treats this call as
/// idempotent on `(account, fingerprint)` so re-registering rotates the
/// token rather than creating a second device — which is exactly what we
/// want after a Mac reinstall or Keychain reset.
public actor DeviceRegistrar {
    private let api: APIClient
    private let credentials: CredentialStore
    private let fingerprint: String

    public init(api: APIClient, credentials: CredentialStore, fingerprint: String) {
        self.api = api
        self.credentials = credentials
        self.fingerprint = fingerprint
    }

    private struct Request: Encodable {
        let platform: String
        let fingerprint: String
    }

    private struct Response: Decodable {
        // swiftlint:disable identifier_name
        let device_id: String
        let device_token: String
        // swiftlint:enable identifier_name
    }

    /// Register (or re-register) this device. Skips the network round trip
    /// if a device id is already in the store *unless* `force == true`,
    /// because the only reasons to re-register are token rotation after a
    /// 401 or a deliberate user action — both of which should pass `force`.
    @discardableResult
    public func register(force: Bool = false) async throws -> RegisteredDevice {
        if !force,
           let id = try credentials.readDeviceID(),
           let token = try credentials.readDeviceToken(),
           !id.isEmpty, !token.isEmpty {
            return RegisteredDevice(id: id, token: token)
        }

        let response: Response = try await api.send(
            method: "POST",
            path: "v1/devices/register",
            body: Request(platform: "macos", fingerprint: fingerprint),
            requireDeviceToken: false
        )

        try credentials.writeDeviceID(response.device_id)
        try credentials.writeDeviceToken(response.device_token)
        return RegisteredDevice(id: response.device_id, token: response.device_token)
    }
}

public struct RegisteredDevice: Equatable, Sendable {
    public let id: String
    public let token: String
}
