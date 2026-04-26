import XCTest
import PolicyEngine
@testable import UsageCollector

final class UsageCollectorTests: XCTestCase {

    var events: [UsageEvent] = []
    var source: FakeWorkspaceSource!
    var collector: UsageCollector!

    override func setUp() {
        source = FakeWorkspaceSource()
        events = []
        collector = UsageCollector(source: source) { [weak self] in self?.events.append($0) }
    }

    override func tearDown() {
        collector = nil
        source = nil
    }

    // MARK: - §1.8 Activation stream

    func testNoEventBeforeFirstActivation() {
        XCTAssertTrue(events.isEmpty)
    }

    func testAppSwitchEmitsClosedIntervalForPreviousApp() {
        let t0 = Date(timeIntervalSince1970: 1_000)
        let t1 = Date(timeIntervalSince1970: 1_060)

        source.emit(.appActivated(bundleID: "com.example.A", at: t0))
        source.emit(.appActivated(bundleID: "com.example.B", at: t1))

        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].bundleID, BundleID("com.example.A"))
        XCTAssertEqual(events[0].start, t0)
        XCTAssertEqual(events[0].end, t1)
    }

    // MARK: - §1.8 System sleep

    func testSystemSleepClosesOpenEvent() {
        let t0 = Date(timeIntervalSince1970: 1_000)
        let t1 = Date(timeIntervalSince1970: 1_060)

        source.emit(.appActivated(bundleID: "com.example.A", at: t0))
        source.emit(.systemWillSleep(at: t1))

        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].end, t1)
    }

    func testSystemSleepWithNoActiveAppEmitsNothing() {
        source.emit(.systemWillSleep(at: Date(timeIntervalSince1970: 1_000)))
        XCTAssertTrue(events.isEmpty)
    }

    // MARK: - §1.8 System wake

    func testSystemWakeAloneEmitsNoEvent() {
        source.emit(.systemWillSleep(at: Date(timeIntervalSince1970: 1_000)))
        source.emit(.systemDidWake(at: Date(timeIntervalSince1970: 1_300)))
        XCTAssertTrue(events.isEmpty)
    }

    func testActivationAfterWakeResumesTracking() {
        let t0 = Date(timeIntervalSince1970: 1_000)
        let t1 = Date(timeIntervalSince1970: 1_060)
        let t2 = Date(timeIntervalSince1970: 1_300)
        let t3 = Date(timeIntervalSince1970: 1_360)

        source.emit(.appActivated(bundleID: "com.example.A", at: t0))
        source.emit(.systemWillSleep(at: t1))
        source.emit(.systemDidWake(at: t2))
        source.emit(.appActivated(bundleID: "com.example.A", at: t3))

        XCTAssertEqual(events.count, 1)        // only the pre-sleep interval
        XCTAssertEqual(events[0].end, t1)
    }

    // MARK: - §1.8 Screen lock

    func testScreenSleepClosesOpenEvent() {
        let t0 = Date(timeIntervalSince1970: 1_000)
        let t1 = Date(timeIntervalSince1970: 1_060)

        source.emit(.appActivated(bundleID: "com.example.A", at: t0))
        source.emit(.screensDidSleep(at: t1))

        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].end, t1)
    }

    // MARK: - §1.8 Flush on quit

    func testFlushEmitsOpenEventAtClockTime() {
        let tFlush = Date(timeIntervalSince1970: 1_120)
        collector = UsageCollector(source: source, clock: { tFlush }) { [weak self] in
            self?.events.append($0)
        }
        source.emit(.appActivated(bundleID: "com.example.A", at: Date(timeIntervalSince1970: 1_000)))
        collector.flush()

        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].end, tFlush)
    }

    func testFlushWithNoActiveAppEmitsNothing() {
        collector.flush()
        XCTAssertTrue(events.isEmpty)
    }
}
