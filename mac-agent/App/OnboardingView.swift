import AuthenticationServices
import SwiftUI
import SyncClient

/// First-launch screen. Asks the user to sign in with Apple so the Mac
/// agent can talk to the backend on their behalf. Until sign-in succeeds,
/// the menu-bar UI cannot show usage data because no JWT is available to
/// fetch device/event ownership.
struct OnboardingView: View {
    @Bindable var viewModel: OnboardingViewModel

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: "timer")
                .font(.system(size: 48))
                .foregroundStyle(Color.accentColor)

            Text("MacAgent")
                .font(.title.bold())

            Text(
                "MacAgent quietly runs in your menu bar and records which apps you use. " +
                "Sign in to sync usage with your Android dashboard."
            )
            .multilineTextAlignment(.center)
            .foregroundStyle(.secondary)
            .fixedSize(horizontal: false, vertical: true)

            content
        }
        .padding(32)
        .frame(width: 360)
    }

    @ViewBuilder
    private var content: some View {
        switch viewModel.phase {
        case .idle:
            signInButton
        case .loading:
            ProgressView()
                .controlSize(.regular)
                .padding(.vertical, 8)
        case .error(let message):
            VStack(spacing: 12) {
                Text(message)
                    .multilineTextAlignment(.center)
                    .foregroundStyle(.red)
                    .fixedSize(horizontal: false, vertical: true)
                signInButton
            }
        }
    }

    private var signInButton: some View {
        SignInWithAppleButton(.signIn) { request in
            request.requestedScopes = [.email]
        } onCompletion: { result in
            handle(result)
        }
        .signInWithAppleButtonStyle(.black)
        .frame(height: 40)
    }

    private func handle(_ result: Result<ASAuthorization, Error>) {
        switch result {
        case .success(let authorization):
            guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
                  let tokenData = credential.identityToken,
                  let token = String(data: tokenData, encoding: .utf8) else {
                Task { @MainActor in
                    await viewModel.signIn(identityToken: "")  // surfaces a generic error
                }
                return
            }
            Task { @MainActor in
                await viewModel.signIn(identityToken: token)
            }
        case .failure:
            // ASAuthorizationError including user-cancellation lands here.
            // Cancellation should leave the UI on idle so the user can
            // retry; surface other failures via dismissError → idle so the
            // button reappears.
            viewModel.dismissError()
        }
    }
}
