import XCTest
@testable import SyncClient

final class BackoffTests: XCTestCase {

    func testFirstAttemptUsesBaseDelay() {
        // With random fixed at the upper bound, attempt 0 must be exactly `base`.
        let backoff = Backoff(base: 1, cap: 60, random: { range in range.upperBound })
        XCTAssertEqual(backoff.delay(forAttempt: 0), 1, accuracy: 1e-9)
    }

    func testDelayDoublesPerAttempt() {
        let backoff = Backoff(base: 1, cap: 1_000, random: { range in range.upperBound })
        XCTAssertEqual(backoff.delay(forAttempt: 1), 2, accuracy: 1e-9)
        XCTAssertEqual(backoff.delay(forAttempt: 2), 4, accuracy: 1e-9)
        XCTAssertEqual(backoff.delay(forAttempt: 3), 8, accuracy: 1e-9)
    }

    func testDelayCapsAtCeiling() {
        let backoff = Backoff(base: 1, cap: 5, random: { range in range.upperBound })
        XCTAssertEqual(backoff.delay(forAttempt: 10), 5, accuracy: 1e-9)
    }

    func testJitterStaysWithinRange() {
        let backoff = Backoff(base: 1, cap: 60)
        for attempt in 0..<5 {
            let delay = backoff.delay(forAttempt: attempt)
            XCTAssertGreaterThanOrEqual(delay, 0)
            let upper = min(pow(2, Double(attempt)), 60)
            XCTAssertLessThan(delay, upper + 1e-9,
                              "attempt \(attempt) delay \(delay) exceeded ceiling \(upper)")
        }
    }
}
