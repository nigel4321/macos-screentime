import Foundation
@testable import UsageCollector

final class FakeWorkspaceSource: WorkspaceSource {
    private var handler: ((WorkspaceEvent) -> Void)?

    func subscribe(handler: @escaping (WorkspaceEvent) -> Void) {
        self.handler = handler
    }

    func emit(_ event: WorkspaceEvent) {
        handler?(event)
    }
}
