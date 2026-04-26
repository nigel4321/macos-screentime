import SwiftUI

@main
struct MacAgentApp: App {
    @State private var container = AppContainer()

    var body: some Scene {
        MenuBarExtra("Screen Time", systemImage: "timer") {
            TodayView(viewModel: container.todayViewModel)
        }
        .menuBarExtraStyle(.window)
    }
}
