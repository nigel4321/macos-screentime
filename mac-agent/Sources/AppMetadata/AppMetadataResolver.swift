import AppKit
import Foundation
import PolicyEngine

/// Resolves a `BundleID` to a human-friendly display name. Lives behind a
/// protocol so tests can substitute a fake without spinning up `NSWorkspace`
/// or assuming particular apps are installed on the CI runner.
public protocol AppMetadataResolver: Sendable {
    /// Returns the human-friendly name for the given bundle id, falling
    /// back to the raw bundle id when no installed app can be located.
    /// Implementations are expected to cache results.
    func displayName(for bundleID: BundleID) -> String
}

/// Production resolver. Looks up the on-disk app via `NSWorkspace`, reads
/// `CFBundleDisplayName` (preferred) or `CFBundleName` from the bundle's
/// `Info.plist`, and caches the result. Cache is process-lifetime — the
/// menubar app is short-lived enough that staleness from a renamed app
/// isn't worth the complexity of invalidation.
public final class SystemAppMetadataResolver: AppMetadataResolver, @unchecked Sendable {
    private var cache: [BundleID: String] = [:]
    private let lock = NSLock()
    private let lookup: @Sendable (String) -> String?

    /// `lookup` is injectable so unit tests don't depend on which apps the
    /// CI runner happens to have installed. Production callers should use
    /// the parameterless `init()` which wires `systemLookup`.
    public init(lookup: @escaping @Sendable (String) -> String? = SystemAppMetadataResolver.systemLookup) {
        self.lookup = lookup
    }

    public func displayName(for bundleID: BundleID) -> String {
        lock.lock()
        if let cached = cache[bundleID] {
            lock.unlock()
            return cached
        }
        lock.unlock()

        let resolved = lookup(bundleID.value) ?? bundleID.value

        lock.lock()
        cache[bundleID] = resolved
        lock.unlock()
        return resolved
    }

    /// Filesystem-touching lookup used by the production resolver. Tries
    /// `CFBundleDisplayName`, then `CFBundleName`, then falls back to the
    /// `.app` directory name with the suffix stripped.
    public static let systemLookup: @Sendable (String) -> String? = { bundleID in
        guard let url = NSWorkspace.shared.urlForApplication(withBundleIdentifier: bundleID) else {
            return nil
        }
        if let bundle = Bundle(url: url) {
            if let display = bundle.localizedInfoDictionary?["CFBundleDisplayName"] as? String,
               !display.isEmpty {
                return display
            }
            if let name = bundle.infoDictionary?["CFBundleName"] as? String, !name.isEmpty {
                return name
            }
        }
        let last = FileManager.default.displayName(atPath: url.path)
        if last.hasSuffix(".app") {
            return String(last.dropLast(4))
        }
        return last.isEmpty ? nil : last
    }
}
