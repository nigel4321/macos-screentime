/// Abstraction over `NSWorkspace` notifications, making `UsageCollector`
/// testable without requiring a live macOS session.
public protocol WorkspaceSource: AnyObject {
    /// Subscribe to workspace events. Only one subscriber is supported.
    func subscribe(handler: @escaping (WorkspaceEvent) -> Void)
}
