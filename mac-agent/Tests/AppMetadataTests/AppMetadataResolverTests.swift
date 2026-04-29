import XCTest
import PolicyEngine
@testable import AppMetadata

final class AppMetadataResolverTests: XCTestCase {

    func testReturnsLookupResult() {
        let resolver = SystemAppMetadataResolver(lookup: { id in
            id == "com.google.Chrome" ? "Google Chrome" : nil
        })
        XCTAssertEqual(resolver.displayName(for: BundleID("com.google.Chrome")),
                       "Google Chrome")
    }

    func testFallsBackToBundleIDWhenLookupReturnsNil() {
        let resolver = SystemAppMetadataResolver(lookup: { _ in nil })
        XCTAssertEqual(resolver.displayName(for: BundleID("com.example.A")),
                       "com.example.A")
    }

    func testCachesResolvedNames() {
        let counter = CallCounter()
        let resolver = SystemAppMetadataResolver(lookup: { _ in
            counter.increment()
            return "X"
        })

        _ = resolver.displayName(for: BundleID("com.example.A"))
        _ = resolver.displayName(for: BundleID("com.example.A"))
        _ = resolver.displayName(for: BundleID("com.example.A"))

        XCTAssertEqual(counter.value, 1, "lookup must be invoked at most once per bundle id")
    }

    func testCachesNegativeResultsToo() {
        // A bundle id that won't ever resolve (uninstalled app) should also
        // be cached, otherwise we'd hit NSWorkspace on every menubar tick.
        let counter = CallCounter()
        let resolver = SystemAppMetadataResolver(lookup: { _ in
            counter.increment()
            return nil
        })

        _ = resolver.displayName(for: BundleID("com.uninstalled"))
        _ = resolver.displayName(for: BundleID("com.uninstalled"))

        XCTAssertEqual(counter.value, 1)
    }

    func testDifferentBundleIDsGetIndependentLookups() {
        let counter = CallCounter()
        let resolver = SystemAppMetadataResolver(lookup: { id in
            counter.increment()
            return id.uppercased()
        })

        XCTAssertEqual(resolver.displayName(for: BundleID("com.a")), "COM.A")
        XCTAssertEqual(resolver.displayName(for: BundleID("com.b")), "COM.B")
        XCTAssertEqual(counter.value, 2)
    }
}

/// Thread-safe counter for capturing call counts inside the `@Sendable`
/// lookup closures the production resolver expects.
private final class CallCounter: @unchecked Sendable {
    private var count = 0
    private let lock = NSLock()

    func increment() {
        lock.lock()
        count += 1
        lock.unlock()
    }

    var value: Int {
        lock.lock()
        defer { lock.unlock() }
        return count
    }
}
