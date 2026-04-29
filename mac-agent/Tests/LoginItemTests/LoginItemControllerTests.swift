import XCTest
@testable import LoginItem

final class LoginItemControllerTests: XCTestCase {

    func testEnsureEnabledRegistersWhenNotRegistered() throws {
        let registry = InMemoryLoginItemRegistry(initialStatus: .notRegistered)
        let controller = LoginItemController(registry: registry)

        try controller.ensureEnabled()

        XCTAssertEqual(registry.registerCalls, 1)
        XCTAssertEqual(registry.currentStatus(), .enabled)
    }

    func testEnsureEnabledIsNoopWhenAlreadyEnabled() throws {
        let registry = InMemoryLoginItemRegistry(initialStatus: .enabled)
        let controller = LoginItemController(registry: registry)

        try controller.ensureEnabled()

        XCTAssertEqual(registry.registerCalls, 0,
                       "must not re-register when SMAppService already reports enabled")
    }

    func testEnsureEnabledRetriesOnRequiresApproval() throws {
        // System Settings → Login Items can disable the item out from under
        // us. We still call register() — macOS keeps the user's choice
        // authoritative, so the status will simply remain .requiresApproval
        // afterward, but attempting on every launch is the correct policy.
        let registry = InMemoryLoginItemRegistry(initialStatus: .requiresApproval)
        let controller = LoginItemController(registry: registry)

        try controller.ensureEnabled()

        XCTAssertEqual(registry.registerCalls, 1)
    }

    func testEnsureEnabledWrapsRegistryError() {
        struct StubError: Error {}
        let registry = InMemoryLoginItemRegistry(initialStatus: .notRegistered)
        registry.registerError = StubError()
        let controller = LoginItemController(registry: registry)

        XCTAssertThrowsError(try controller.ensureEnabled()) { error in
            guard case LoginItemError.registryFailed = error else {
                XCTFail("expected LoginItemError.registryFailed, got \(error)")
                return
            }
        }
    }

    func testStatusPassesThrough() {
        let registry = InMemoryLoginItemRegistry(initialStatus: .requiresApproval)
        let controller = LoginItemController(registry: registry)
        XCTAssertEqual(controller.status(), .requiresApproval)
    }
}
