import Foundation
import LocalStore
import UsageCollector

/// Owns every long-lived dependency and wires them together.
/// Created once at app launch and torn down on quit.
@MainActor
final class AppContainer {
    let todayViewModel: TodayViewModel

    private let usageEventDAO: UsageEventDAO
    private let source: NSWorkspaceSource
    private let collector: UsageCollector

    init() {
        // swiftlint:disable:next force_try
        let db = try! AppDatabase(path: Self.databasePath())
        usageEventDAO = UsageEventDAO(database: db)
        source = NSWorkspaceSource()
        let dao = usageEventDAO
        let vm = TodayViewModel(dao: dao)
        todayViewModel = vm
        collector = UsageCollector(source: source) { event in
            try? dao.insert(event)
            Task { @MainActor in vm.refresh() }
        }
    }

    func flush() {
        collector.flush()
    }

    private static func databasePath() -> String {
        // swiftlint:disable:next force_unwrapping
        let support = FileManager.default.urls(
            for: .applicationSupportDirectory,
            in: .userDomainMask
        ).first!
        let dir = support.appendingPathComponent("MacAgent", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir.appendingPathComponent("data.db").path
    }
}
