import LoginItem
import SwiftUI

struct TodayView: View {
    var viewModel: TodayViewModel
    var loginItem: LoginItemController?
    @State private var launchAtLogin: Bool = false

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
            launchAtLoginToggle
            Divider()
            quitButton
        }
        .frame(width: 280)
        .onAppear { syncLaunchAtLoginState() }
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

    @ViewBuilder
    private var launchAtLoginToggle: some View {
        if let loginItem {
            Toggle("Launch at Login", isOn: Binding(
                get: { launchAtLogin },
                set: { newValue in
                    do {
                        try loginItem.setEnabled(newValue)
                        launchAtLogin = newValue
                    } catch {
                        // Re-sync from the registry on failure so the
                        // toggle reflects the OS's actual state, not the
                        // stale UI state.
                        syncLaunchAtLoginState()
                    }
                }
            ))
            .toggleStyle(.switch)
            .padding(.horizontal, 12)
            .padding(.vertical, 6)
        }
    }

    private func syncLaunchAtLoginState() {
        guard let loginItem else { return }
        launchAtLogin = (loginItem.status() == .enabled)
    }

    private var quitButton: some View {
        Button("Quit MacAgent") {
            NSApplication.shared.terminate(nil)
        }
        .buttonStyle(.plain)
        .foregroundStyle(.secondary)
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
    }
}
