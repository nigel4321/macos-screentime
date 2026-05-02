import Foundation
import Observation
import os

/// Drives the sign-in flow on `OnboardingView`. The view obtains an Apple
/// identity token via `SignInWithAppleButton` and hands it to
/// `signIn(identityToken:)`; the view model exchanges it for a backend JWT
/// via `AuthClient` and notifies its listener on success.
///
/// `onAuthenticated` is the seam that lets the App layer flip the menubar
/// UI from onboarding to the today view without coupling the VM to
/// AppKit/SwiftUI navigation.
///
/// Lives in `SyncClient` rather than the App target so it can be unit-tested
/// via `swift test`. The class imports `Observation` but neither SwiftUI nor
/// AppKit, so the network/UI boundary stays intact.
@Observable
@MainActor
public final class OnboardingViewModel {
    public enum Phase: Equatable {
        case idle
        case loading
        case error(String)
    }

    public private(set) var phase: Phase = .idle

    public var onAuthenticated: ((Authenticated) -> Void)?

    private let authClient: AuthClient
    private let logger = Logger(subsystem: "com.macos-screentime.MacAgent", category: "Onboarding")

    public init(authClient: AuthClient) {
        self.authClient = authClient
    }

    /// Exchanges an Apple identity token for a backend JWT. Calls made
    /// while one is already in flight are dropped so a programmatic
    /// double-invocation (or a SwiftUI re-render) cannot start two
    /// concurrent sign-ins.
    public func signIn(identityToken: String) async {
        if phase == .loading { return }
        phase = .loading
        do {
            let result = try await authClient.signInWithApple(identityToken: identityToken)
            phase = .idle
            onAuthenticated?(result)
        } catch let error as APIError {
            logger.error("sign-in failed: \(String(describing: error), privacy: .public)")
            phase = .error(Self.message(for: error))
        } catch {
            logger.error("sign-in failed: \(error.localizedDescription, privacy: .public)")
            phase = .error("Sign-in failed. Please try again.")
        }
    }

    public func dismissError() {
        if case .error = phase { phase = .idle }
    }

    private static func message(for error: APIError) -> String {
        switch error {
        case .unauthorized:
            return "Apple rejected the sign-in. Please try again."
        case .serverError, .transport:
            return "Couldn't reach the server. Check your connection and try again."
        case .clientError, .decoding, .missingCredentials:
            return "Sign-in failed. Please try again."
        }
    }
}
