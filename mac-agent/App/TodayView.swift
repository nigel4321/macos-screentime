import SwiftUI

struct TodayView: View {
    var viewModel: TodayViewModel

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
            quitButton
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
