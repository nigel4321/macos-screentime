import Foundation

/// `URLProtocol` subclass that intercepts `URLRequest`s sent through a
/// session whose configuration includes it in `protocolClasses`. The
/// `requestHandler` produces a canned response, lets tests assert on the
/// outgoing request, and lets a single test sequence multiple responses by
/// re-assigning the closure between calls.
///
/// Used instead of a Vapor mock server (the roadmap's original suggestion)
/// because a `URLProtocol` runs in-process, has no port-binding flake, and
/// lets tests inspect the exact `URLRequest` the production code sent.
final class MockURLProtocol: URLProtocol, @unchecked Sendable {
    /// Set this before constructing the URLSession to be tested. Each call
    /// receives the outgoing request and must return the response tuple.
    nonisolated(unsafe) static var requestHandler: ((URLRequest) throws -> (HTTPURLResponse, Data))?

    /// Optional sink for every request the protocol intercepts. Useful for
    /// asserting that headers / bodies match what the test expected.
    nonisolated(unsafe) static var capturedRequests: [URLRequest] = []

    static func reset() {
        requestHandler = nil
        capturedRequests = []
    }

    override class func canInit(with request: URLRequest) -> Bool { true }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        // The system strips httpBody on URLRequest before handing it to
        // URLProtocol; pull it out of the body stream so tests can inspect.
        var captured = request
        if let stream = request.httpBodyStream {
            stream.open()
            defer { stream.close() }
            var data = Data()
            let buffer = UnsafeMutablePointer<UInt8>.allocate(capacity: 4096)
            defer { buffer.deallocate() }
            while stream.hasBytesAvailable {
                let read = stream.read(buffer, maxLength: 4096)
                if read <= 0 { break }
                data.append(buffer, count: read)
            }
            captured.httpBody = data
        }
        Self.capturedRequests.append(captured)

        guard let handler = Self.requestHandler else {
            client?.urlProtocol(self, didFailWithError: URLError(.badServerResponse))
            return
        }
        do {
            let (response, data) = try handler(captured)
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}
}

extension URLSession {
    /// Build a `URLSession` that routes every request through `MockURLProtocol`.
    static func mocked() -> URLSession {
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [MockURLProtocol.self]
        return URLSession(configuration: config)
    }
}
