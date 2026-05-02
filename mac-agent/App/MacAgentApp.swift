import SwiftUI

@main
struct MacAgentApp: App {
    @State private var container = AppContainer()

    var body: some Scene {
        MenuBarExtra("Screen Time", systemImage: "timer") {
            // TodayView always shows: usage collection runs locally
            // regardless of auth, so hiding the data behind sign-in helps
            // nobody. The footer adapts based on container.authPhase to
            // show SIWA (unauth) or Sign out (auth).
            TodayView(
                viewModel: container.todayViewModel,
                authViewModel: container.onboardingViewModel,
                authPhase: container.authPhase,
                onSignOut: { Task { await container.signOut() } }
            )
        }
        .menuBarExtraStyle(.window)
        .commands {
            // Tamper-resistance: the agent must keep running on a child's
            // account, so we strip the in-app exit affordances. An empty
            // .appTermination group removes both the menu item *and* the
            // Cmd-Q binding. macOS's force-quit (Activity Monitor / ⌘⌥⎋)
            // still works — §1.14's LaunchAgent KeepAlive respawns those.
            // The parent has the same constraint on their own machine; to
            // actually stop the agent they use Activity Monitor or
            // launchctl. This matches the §1.13 ARCHITECTURE note that
            // launch-at-login is mandatory by design.
            CommandGroup(replacing: .appTermination) { }
        }
    }
}
