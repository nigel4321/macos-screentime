import AuthenticationServices
import SwiftUI
import SyncClient

/// Menu-bar window contents. Always renders the local usage list at the
/// top — collection runs regardless of auth state, so there's no reason
/// to gate the data behind sign-in. The footer adapts:
///   - unauthenticated: a small `SignInWithAppleButton`, plus inline
///     loading / error states from the auth view model.
///   - authenticated:   a "Sign out" button.
/// An "About" disclosure at the very bottom expands to show version +
/// description.
struct TodayView: View {
    var viewModel: TodayViewModel
    @Bindable var authViewModel: OnboardingViewModel
    var authPhase: AppContainer.AuthPhase
    var onSignOut: () -> Void

    @State private var aboutExpanded: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
            Divider()
            if viewModel.topApps.isEmpty {
                emptyState
            } else {
                appList
            }
            Divider()
            authFooter
            Divider()
            aboutDisclosure
        }
        .frame(width: 280)
    }

    private var header: some View {
        Text("Today")
            .font(.headline)
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
    }

    private var emptyState: some View {
        Text("No usage recorded yet")
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, alignment: .center)
            .padding(24)
    }

    private var appList: some View {
        VStack(spacing: 0) {
            ForEach(viewModel.topApps) { app in
                HStack {
                    Text(app.displayName)
                        .lineLimit(1)
                        .truncationMode(.middle)
                    Spacer()
                    Text(app.formattedDuration)
                        .foregroundStyle(.secondary)
                        .monospacedDigit()
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
            }
        }
        .padding(.vertical, 4)
    }

    // No "Quit" button — see MacAgentApp's empty .appTermination
    // CommandGroup. Sign-out is the only in-app exit; the agent itself
    // keeps running.
    @ViewBuilder
    private var authFooter: some View {
        switch authPhase {
        case .unauthenticated:
            unauthenticatedFooter
        case .authenticated:
            authenticatedFooter
        }
    }

    @ViewBuilder
    private var unauthenticatedFooter: some View {
        VStack(alignment: .leading, spacing: 6) {
            if case .error(let message) = authViewModel.phase {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.red)
                    .fixedSize(horizontal: false, vertical: true)
            }
            switch authViewModel.phase {
            case .loading:
                HStack {
                    ProgressView()
                        .controlSize(.small)
                    Text("Signing in…")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .frame(height: 28)
            case .idle, .error:
                signInButton
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
    }

    private var signInButton: some View {
        SignInWithAppleButton(.signIn) { request in
            request.requestedScopes = [.email]
        } onCompletion: { result in
            handle(result)
        }
        .signInWithAppleButtonStyle(.black)
        .frame(height: 28)
    }

    private var authenticatedFooter: some View {
        Button("Sign out") { onSignOut() }
            .buttonStyle(.plain)
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
    }

    private var aboutDisclosure: some View {
        VStack(alignment: .leading, spacing: 0) {
            Button {
                withAnimation(.easeInOut(duration: 0.15)) { aboutExpanded.toggle() }
            } label: {
                HStack {
                    Text("About")
                        .foregroundStyle(.secondary)
                    Spacer()
                    Image(systemName: aboutExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .padding(.horizontal, 12)
            .padding(.vertical, 8)

            if aboutExpanded {
                Divider()
                AboutView()
            }
        }
    }

    private func handle(_ result: Result<ASAuthorization, Error>) {
        switch result {
        case .success(let authorization):
            guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
                  let tokenData = credential.identityToken,
                  let token = String(data: tokenData, encoding: .utf8) else {
                Task { @MainActor in
                    await authViewModel.signIn(identityToken: "")  // surfaces a generic error
                }
                return
            }
            Task { @MainActor in
                await authViewModel.signIn(identityToken: token)
            }
        case .failure:
            // ASAuthorizationError including user-cancellation lands here.
            // Cancellation should leave the UI on idle so the user can
            // retry; surface other failures via dismissError → idle so
            // the button reappears.
            authViewModel.dismissError()
        }
    }
}
