import SwiftUI

/// Inline "About" section shown at the bottom of the menubar window when
/// the user expands the disclosure. Reads the version straight off the
/// running bundle so it can't drift from what xcodebuild produced.
struct AboutView: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Text("MacAgent")
                    .font(.subheadline.bold())
                Text(Self.versionString)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .monospacedDigit()
            }
            Text(
                "MacAgent runs quietly in your menu bar, recording which " +
                "apps you use throughout the day. Sign in with Apple to " +
                "sync your usage to an Android dashboard for parental review."
            )
            .font(.caption)
            .foregroundStyle(.secondary)
            .fixedSize(horizontal: false, vertical: true)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    /// "v1.2.3 (45)" if both keys are set, "v1.2.3" otherwise, "—" as last
    /// resort. CFBundleVersion drifts from CFBundleShortVersionString in
    /// CI tag builds (versionCode vs versionName), which is why the build
    /// number is worth showing alongside.
    private static var versionString: String {
        let info = Bundle.main.infoDictionary
        let short = info?["CFBundleShortVersionString"] as? String
        let build = info?["CFBundleVersion"] as? String
        switch (short, build) {
        case let (s?, b?) where s != b: return "v\(s) (\(b))"
        case let (s?, _): return "v\(s)"
        case let (_, b?): return "build \(b)"
        default: return "—"
        }
    }
}
