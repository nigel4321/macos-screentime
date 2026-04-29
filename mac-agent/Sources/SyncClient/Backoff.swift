import Foundation

/// Exponential backoff with full jitter, kept pure so tests do not need to
/// sleep. Sequence: `min(base * 2^n, cap)` randomised in `[0, value]` per
/// AWS Architecture Blog "Exponential Backoff and Jitter".
///
/// Stateless — the caller is responsible for tracking the attempt count.
public struct Backoff: Sendable {
    public let base: TimeInterval
    public let cap: TimeInterval

    /// Source of randomness. Injectable so tests can pin the jitter to a
    /// known value (e.g. `{ _ in 1.0 }` for the upper edge).
    public let random: @Sendable (Range<Double>) -> Double

    public init(
        base: TimeInterval = 1,
        cap: TimeInterval = 60,
        random: @escaping @Sendable (Range<Double>) -> Double = { Double.random(in: $0) }
    ) {
        self.base = base
        self.cap = cap
        self.random = random
    }

    /// Delay for `attempt`, where the first failure has `attempt == 0`.
    public func delay(forAttempt attempt: Int) -> TimeInterval {
        let exponent = max(0, attempt)
        let raw = base * pow(2, Double(exponent))
        let bounded = min(raw, cap)
        guard bounded > 0 else { return 0 }
        return random(0..<bounded)
    }
}
