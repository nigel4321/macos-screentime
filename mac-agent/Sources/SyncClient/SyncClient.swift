import Foundation
import LocalStore
import os

/// Facade that the app layer talks to. Owns an `APIClient`, a
/// `DeviceRegistrar`, and a `BatchUploader`, and exposes a single
/// `flush()` entry point that:
///
///   1. No-ops when no JWT is present (we are not signed in yet).
///   2. Registers the device on first run (or on token rotation).
///   3. Uploads any unsynced events with backoff on transport / 5xx errors.
///
/// The intent is that `AppContainer` calls `flush()` periodically and on
/// quit, and `SyncClient` decides whether to actually do anything.
public actor SyncClient {
    public let api: APIClient
    public let credentials: CredentialStore
    public let registrar: DeviceRegistrar
    public let uploader: BatchUploader
    public let backoff: Backoff
    /// Max number of attempts per `flush()` before giving up and surfacing
    /// the underlying error. Tuned for "called every 60s" — we'd rather
    /// give up and let the next tick try again than block flush forever.
    public let maxAttempts: Int

    private let logger = Logger(subsystem: "com.macos-screentime.MacAgent", category: "SyncClient")
    private let sleepNanos: @Sendable (UInt64) async throws -> Void

    public init(
        baseURL: URL,
        credentials: CredentialStore,
        dao: UsageEventDAO,
        fingerprint: String,
        session: URLSession = .shared,
        backoff: Backoff = Backoff(),
        maxAttempts: Int = 4,
        sleepNanos: @escaping @Sendable (UInt64) async throws -> Void = { try await Task.sleep(nanoseconds: $0) }
    ) {
        let api = APIClient(baseURL: baseURL, credentials: credentials, session: session)
        self.api = api
        self.credentials = credentials
        self.registrar = DeviceRegistrar(api: api, credentials: credentials, fingerprint: fingerprint)
        self.uploader = BatchUploader(api: api, dao: dao, registrar: self.registrar)
        self.backoff = backoff
        self.maxAttempts = maxAttempts
        self.sleepNanos = sleepNanos
    }

    /// Reason `flush()` returned without uploading anything. Useful for
    /// callers (and tests) to distinguish "noop because not signed in"
    /// from "drained the queue successfully".
    public enum FlushOutcome: Equatable {
        /// No JWT in the credential store — sign-in (§2.10a) hasn't
        /// happened yet. Caller should treat this as success.
        case noCredentials
        /// JWT present, queue drained. `synced` is the number of rows
        /// transitioned from unsynced to synced this call.
        case completed(synced: Int)
        /// Recoverable error after `maxAttempts` retries. Caller should
        /// log and try again on the next tick.
        case gaveUp(lastError: APIError)
    }

    public func flush() async -> FlushOutcome {
        do {
            guard let jwt = try credentials.readJWT(), !jwt.isEmpty else {
                logger.debug("flush: no JWT, skipping")
                return .noCredentials
            }
        } catch {
            logger.error("flush: credential read failed: \(error.localizedDescription, privacy: .public)")
            return .noCredentials
        }

        var attempt = 0
        var lastError: APIError = .transport(message: "no attempt made")
        while attempt < maxAttempts {
            do {
                let synced = try await uploader.flush()
                return .completed(synced: synced)
            } catch let error as APIError {
                lastError = error
                if !Self.shouldRetry(error) {
                    logger.error("flush: non-retryable error: \(String(describing: error), privacy: .public)")
                    return .gaveUp(lastError: error)
                }
                let delay = backoff.delay(forAttempt: attempt)
                logger.info("flush: retryable error \(String(describing: error), privacy: .public); sleeping \(delay)s before attempt \(attempt + 2)")
                try? await sleepNanos(UInt64(delay * 1_000_000_000))
                attempt += 1
            } catch {
                logger.error("flush: unexpected error: \(error.localizedDescription, privacy: .public)")
                return .gaveUp(lastError: .transport(message: error.localizedDescription))
            }
        }
        return .gaveUp(lastError: lastError)
    }

    /// Whether an `APIError` is worth retrying. 5xx and transport failures
    /// are transient; 4xx (other than 401) and decoding errors will not
    /// improve on retry. 401 propagates to `gaveUp` so the caller knows to
    /// re-prompt the user.
    static func shouldRetry(_ error: APIError) -> Bool {
        switch error {
        case .serverError, .transport:
            return true
        case .unauthorized, .clientError, .decoding, .missingCredentials:
            return false
        }
    }
}
