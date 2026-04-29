import Foundation
import Security

/// Read/write the long-lived secrets the SyncClient needs.
///
/// Three pieces are persisted:
///   - `jwt`: the backend access token (issued by `POST /v1/auth/apple` once
///     §2.10a's Apple Sign-In flow lands).
///   - `deviceID`: returned once by `POST /v1/devices/register`.
///   - `deviceToken`: returned alongside; sent on every batch upload as the
///     `X-Device-Token` header.
///
/// The protocol exists so tests can substitute an in-memory fake. Production
/// uses `KeychainCredentialStore`.
public protocol CredentialStore: Sendable {
    func readJWT() throws -> String?
    func writeJWT(_ value: String) throws
    func deleteJWT() throws

    func readDeviceID() throws -> String?
    func writeDeviceID(_ value: String) throws

    func readDeviceToken() throws -> String?
    func writeDeviceToken(_ value: String) throws

    /// Wipe everything (JWT + device id + device token). Used by sign-out.
    func clear() throws
}

/// Errors surfaced by the keychain-backed store. The OSStatus is preserved
/// so logs can reference it; production callers should treat them as
/// recoverable (re-prompt the user for sign-in).
public enum CredentialStoreError: Error, Equatable {
    case unexpectedStatus(OSStatus)
    case dataCorrupted
}

/// Production implementation backed by the macOS Keychain. Items are stored
/// in the data-protection keychain (`kSecUseDataProtectionKeychain = true`)
/// so reads do not block on user authorization, and they live under a
/// caller-supplied service name so test runs can use an isolated namespace.
public struct KeychainCredentialStore: CredentialStore {
    private let service: String

    private enum AccountKey {
        static let jwt = "backend_jwt"
        static let deviceID = "device_id"
        static let deviceToken = "device_token"
    }

    public init(service: String = "com.macos-screentime.MacAgent") {
        self.service = service
    }

    public func readJWT() throws -> String? { try readString(account: AccountKey.jwt) }
    public func writeJWT(_ value: String) throws { try writeString(value, account: AccountKey.jwt) }
    public func deleteJWT() throws { try delete(account: AccountKey.jwt) }

    public func readDeviceID() throws -> String? { try readString(account: AccountKey.deviceID) }
    public func writeDeviceID(_ value: String) throws { try writeString(value, account: AccountKey.deviceID) }

    public func readDeviceToken() throws -> String? { try readString(account: AccountKey.deviceToken) }
    public func writeDeviceToken(_ value: String) throws { try writeString(value, account: AccountKey.deviceToken) }

    public func clear() throws {
        try delete(account: AccountKey.jwt)
        try delete(account: AccountKey.deviceID)
        try delete(account: AccountKey.deviceToken)
    }

    private func baseQuery(account: String) -> [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecUseDataProtectionKeychain as String: true
        ]
    }

    private func readString(account: String) throws -> String? {
        var query = baseQuery(account: account)
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne

        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        switch status {
        case errSecSuccess:
            guard let data = item as? Data, let str = String(data: data, encoding: .utf8) else {
                throw CredentialStoreError.dataCorrupted
            }
            return str
        case errSecItemNotFound:
            return nil
        default:
            throw CredentialStoreError.unexpectedStatus(status)
        }
    }

    private func writeString(_ value: String, account: String) throws {
        // swiftlint:disable:next force_unwrapping
        let data = value.data(using: .utf8)!
        let query = baseQuery(account: account)
        let attributes: [String: Any] = [kSecValueData as String: data]

        let updateStatus = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
        switch updateStatus {
        case errSecSuccess:
            return
        case errSecItemNotFound:
            var addQuery = query
            addQuery[kSecValueData as String] = data
            let addStatus = SecItemAdd(addQuery as CFDictionary, nil)
            if addStatus != errSecSuccess {
                throw CredentialStoreError.unexpectedStatus(addStatus)
            }
        default:
            throw CredentialStoreError.unexpectedStatus(updateStatus)
        }
    }

    private func delete(account: String) throws {
        let status = SecItemDelete(baseQuery(account: account) as CFDictionary)
        if status != errSecSuccess && status != errSecItemNotFound {
            throw CredentialStoreError.unexpectedStatus(status)
        }
    }
}

/// In-memory credential store for tests. Thread-safe via an internal lock so
/// tests that run async work against it remain deterministic.
public final class InMemoryCredentialStore: CredentialStore, @unchecked Sendable {
    private var jwt: String?
    private var deviceID: String?
    private var deviceToken: String?
    private let lock = NSLock()

    public init(jwt: String? = nil, deviceID: String? = nil, deviceToken: String? = nil) {
        self.jwt = jwt
        self.deviceID = deviceID
        self.deviceToken = deviceToken
    }

    public func readJWT() throws -> String? { lock.withLock { jwt } }
    public func writeJWT(_ value: String) throws { lock.withLock { jwt = value } }
    public func deleteJWT() throws { lock.withLock { jwt = nil } }

    public func readDeviceID() throws -> String? { lock.withLock { deviceID } }
    public func writeDeviceID(_ value: String) throws { lock.withLock { deviceID = value } }

    public func readDeviceToken() throws -> String? { lock.withLock { deviceToken } }
    public func writeDeviceToken(_ value: String) throws { lock.withLock { deviceToken = value } }

    public func clear() throws {
        lock.withLock {
            jwt = nil
            deviceID = nil
            deviceToken = nil
        }
    }
}
