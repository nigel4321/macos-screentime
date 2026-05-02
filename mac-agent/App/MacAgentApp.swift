import SwiftUI

@main
struct MacAgentApp: App {
    @State private var container = AppContainer()

    var body: some Scene {
        MenuBarExtra("Screen Time", systemImage: "timer") {
            switch container.authPhase {
            case .unauthenticated:
                OnboardingView(viewModel: container.onboardingViewModel)
            case .authenticated:
                TodayView(
                    viewModel: container.todayViewModel,
                    onSignOut: { Task { await container.signOut() } }
                )
            }
        }
        .menuBarExtraStyle(.window)
        .commands {
            // SwiftUI doesn't expose a hookable terminate event, so we rely
            // on `NSApplication.willTerminateNotification` via an observer
            // installed as a side-effect of `body` being constructed.
            CommandGroup(replacing: .appTermination) {
                Button("Quit Screen Time") {
                    Task {
                        await container.flush()
                        NSApplication.shared.terminate(nil)
                    }
                }
                .keyboardShortcut("q", modifiers: .command)
            }
        }
    }
}
