import AppMetadata
import Foundation
import PolicyEngine

/// Test double for `AppMetadataResolver`. Tests pass a `[bundleID: name]`
/// map; bundle IDs not in the map fall back to the bundle id itself, which
/// matches the production `SystemAppMetadataResolver` contract on a miss.
/// `BatchUploader` uses that "resolved == bundleID.value" round-trip as
/// the signal to drop the entry from the outgoing payload, so a fake-with-
/// empty-map exercises the "no app_metadata field on the wire" path.
final class FakeAppMetadataResolver: AppMetadataResolver, @unchecked Sendable {
    private let names: [String: String]
    private let lock = NSLock()
    private var lookups: [String] = []

    init(names: [String: String] = [:]) {
        self.names = names
    }

    func displayName(for bundleID: BundleID) -> String {
        lock.withLock { lookups.append(bundleID.value) }
        return names[bundleID.value] ?? bundleID.value
    }

    var capturedLookups: [String] {
        lock.withLock { lookups }
    }
}
