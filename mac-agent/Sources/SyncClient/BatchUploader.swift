import AppMetadata
import Foundation
import LocalStore
import PolicyEngine
import os

/// Drains unsynced rows from `UsageEventDAO` and POSTs them to
/// `/v1/usage:batchUpload` in chunks bounded by the backend's `MaxBatchSize`
/// of 500. On every page:
///   - `accepted` and `duplicate` rows are marked synced — duplicate is the
///     server telling us "I already have this", which is the same outcome as
///     "I just stored this" from the client's perspective.
///   - `rejected` rows are *also* marked synced and logged. They failed
///     server-side validation (e.g. `started_at` outside the accepted
///     window), so retrying will not change the outcome and leaving them
///     unsynced would re-upload them on every flush.
public actor BatchUploader {
    public static let maxBatchSize = 500
    /// Backend `MaxAppMetadataEntries`. We mirror the cap here so a noisy
    /// page (somehow >100 unique bundles in one batch) doesn't get the
    /// whole request rejected with 413; we drop the overflow and trust the
    /// next batch to upsert the remainder.
    public static let maxAppMetadataEntries = 100

    private let api: APIClient
    private let dao: UsageEventDAO
    private let registrar: DeviceRegistrar
    private let resolver: AppMetadataResolver
    private let clock: @Sendable () -> Date
    private let logger = Logger(subsystem: "com.macos-screentime.MacAgent", category: "BatchUploader")

    public init(
        api: APIClient,
        dao: UsageEventDAO,
        registrar: DeviceRegistrar,
        resolver: AppMetadataResolver,
        clock: @escaping @Sendable () -> Date = { Date() }
    ) {
        self.api = api
        self.dao = dao
        self.registrar = registrar
        self.resolver = resolver
        self.clock = clock
    }

    private struct EventDTO: Encodable {
        // swiftlint:disable identifier_name
        let client_event_id: String
        let bundle_id: String
        let started_at: Date
        let ended_at: Date
        // swiftlint:enable identifier_name
    }

    private struct UploadRequest: Encodable {
        // swiftlint:disable identifier_name
        let events: [EventDTO]
        let app_metadata: [String: String]?
        // swiftlint:enable identifier_name
    }

    private struct EventResultDTO: Decodable {
        // swiftlint:disable identifier_name
        let client_event_id: String
        let status: String
        let reason: String?
        // swiftlint:enable identifier_name
    }

    private struct UploadResponse: Decodable {
        let results: [EventResultDTO]
    }

    /// Returns the number of local rows transitioned from unsynced to synced.
    /// Zero is the expected steady-state once the queue is drained.
    @discardableResult
    public func flush() async throws -> Int {
        // Make sure we have a device id + token before talking to
        // batchUpload. `register(force: false)` is a noop when both are
        // already cached.
        _ = try await registrar.register()

        var totalSynced = 0
        while true {
            let pending = try dao.fetchUnsynced()
            if pending.isEmpty { break }

            let page = Array(pending.prefix(Self.maxBatchSize))
            let payload = UploadRequest(
                events: page.map {
                    EventDTO(
                        client_event_id: $0.clientEventID,
                        bundle_id: $0.event.bundleID.value,
                        started_at: $0.event.start,
                        ended_at: $0.event.end
                    )
                },
                app_metadata: resolveAppMetadata(for: page)
            )

            let response: UploadResponse = try await api.send(
                method: "POST",
                path: "v1/usage:batchUpload",
                body: payload,
                requireDeviceToken: true
            )

            let resultByID = Dictionary(uniqueKeysWithValues: response.results.map { ($0.client_event_id, $0) })
            var idsToSync: [Int64] = []
            for row in page {
                guard let result = resultByID[row.clientEventID] else { continue }
                switch result.status {
                case "accepted", "duplicate":
                    idsToSync.append(row.id)
                case "rejected":
                    logger.error("server rejected event \(row.clientEventID, privacy: .public): \(result.reason ?? "", privacy: .public)")
                    idsToSync.append(row.id)
                default:
                    logger.error("unknown status \(result.status, privacy: .public) for event \(row.clientEventID, privacy: .public)")
                }
            }
            try dao.markSynced(ids: idsToSync, at: clock())
            totalSynced += idsToSync.count

            // If we just sent fewer than the full page, the queue is
            // drained — break to avoid a redundant empty fetch.
            if page.count < Self.maxBatchSize { break }
        }
        return totalSynced
    }

    /// Build the optional batch-level `app_metadata` map for this page.
    /// The resolver returns the bundle id itself when it can't find an
    /// installed app, so entries that round-trip back to the bundle id
    /// are dropped — sending them would just pollute the backend's
    /// `app_metadata` table with names indistinguishable from the key.
    /// Returns `nil` when nothing resolves so the JSON omits the field.
    private func resolveAppMetadata(for page: [UnsyncedUsageEvent]) -> [String: String]? {
        var seen: Set<String> = []
        var map: [String: String] = [:]
        for row in page {
            let bundleID = row.event.bundleID
            if !seen.insert(bundleID.value).inserted { continue }
            let resolved = resolver.displayName(for: bundleID)
            if resolved == bundleID.value { continue }
            map[bundleID.value] = resolved
            if map.count >= Self.maxAppMetadataEntries { break }
        }
        return map.isEmpty ? nil : map
    }
}
