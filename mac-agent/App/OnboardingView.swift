import SwiftUI

/// Shown on first launch to explain what the app does while in observe-only mode.
struct OnboardingView: View {
    var onDismiss: () -> Void

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: "timer")
                .font(.system(size: 48))
                .foregroundStyle(Color.accentColor)

            Text("MacAgent")
                .font(.title.bold())

            Text(
                "MacAgent quietly runs in your menu bar and records which apps you use. " +
                "Connect an Android device to view your usage and set limits."
            )
            .multilineTextAlignment(.center)
            .foregroundStyle(.secondary)
            .fixedSize(horizontal: false, vertical: true)

            Button("Get Started") { onDismiss() }
                .buttonStyle(.borderedProminent)
        }
        .padding(32)
        .frame(width: 360)
    }
}
