import AppKit
import PolicyEngine

/// Production `WorkspaceSource` backed by `NSWorkspace` notifications.
public final class NSWorkspaceSource: WorkspaceSource {
    private var handler: ((WorkspaceEvent) -> Void)?
    private var observers: [NSObjectProtocol] = []

    public init() {}

    public func subscribe(handler: @escaping (WorkspaceEvent) -> Void) {
        self.handler = handler
        let nc = NSWorkspace.shared.notificationCenter

        observers.append(nc.addObserver(
            forName: NSWorkspace.didActivateApplicationNotification,
            object: nil,
            queue: .main
        ) { [weak self] notification in
            guard
                let app = notification.userInfo?[NSWorkspace.applicationUserInfoKey] as? NSRunningApplication,
                let id = app.bundleIdentifier
            else { return }
            self?.handler?(.appActivated(bundleID: BundleID(id), at: Date()))
        })

        observers.append(nc.addObserver(
            forName: NSWorkspace.willSleepNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in self?.handler?(.systemWillSleep(at: Date())) })

        observers.append(nc.addObserver(
            forName: NSWorkspace.didWakeNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in self?.handler?(.systemDidWake(at: Date())) })

        observers.append(nc.addObserver(
            forName: NSWorkspace.screensDidSleepNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in self?.handler?(.screensDidSleep(at: Date())) })
    }

    deinit {
        let nc = NSWorkspace.shared.notificationCenter
        observers.forEach { nc.removeObserver($0) }
    }
}
