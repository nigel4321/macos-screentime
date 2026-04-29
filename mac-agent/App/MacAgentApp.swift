import SwiftUI

@main
struct MacAgentApp: App {
    @State private var container = AppContainer()

    var body: some Scene {
        MenuBarExtra("Screen Time", systemImage: "timer") {
            TodayView(viewModel: container.todayViewModel)
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
