import XCTest
@testable import LoginItem

final class LoginItemControllerTests: XCTestCase {

    func testFirstLaunchAutoRegisters() throws {
        let registry = InMemoryLoginItemRegistry(initialStatus: .notRegistered)
        let prefs = InMemoryLoginItemPreferenceStore()
        let controller = LoginItemController(registry: registry, preferences: prefs)

        try controller.enableOnFirstLaunchIfNeeded()

        XCTAssertEqual(registry.registerCalls, 1)
        XCTAssertEqual(registry.currentStatus(), .enabled)
        XCTAssertTrue(prefs.didAutoRegister())
    }

    func testSecondLaunchDoesNotReRegister() throws {
        let registry = InMemoryLoginItemRegistry(initialStatus: .notRegistered)
        let prefs = InMemoryLoginItemPreferenceStore(autoRegistered: true)
        let controller = LoginItemController(registry: registry, preferences: prefs)

        try controller.enableOnFirstLaunchIfNeeded()

        XCTAssertEqual(registry.registerCalls, 0,
                       "auto-register must run only on the very first launch")
    }

    func testAutoRegisterRespectsUserDisabled() throws {
        let registry = InMemoryLoginItemRegistry(initialStatus: .notRegistered)
        let prefs = InMemoryLoginItemPreferenceStore(autoRegistered: false, userDisabled: true)
        let controller = LoginItemController(registry: registry, preferences: prefs)

        try controller.enableOnFirstLaunchIfNeeded()

        XCTAssertEqual(registry.registerCalls, 0,
                       "must not silently re-enable after user opted out")
        XCTAssertEqual(registry.currentStatus(), .notRegistered)
    }

    func testFailedAutoRegisterStillFlipsTheFlag() {
        struct StubError: Error {}
        let registry = InMemoryLoginItemRegistry(initialStatus: .notRegistered)
        registry.registerError = StubError()
        let prefs = InMemoryLoginItemPreferenceStore()
        let controller = LoginItemController(registry: registry, preferences: prefs)

        XCTAssertThrowsError(try controller.enableOnFirstLaunchIfNeeded())
        XCTAssertTrue(prefs.didAutoRegister(),
                      "must not retry forever when the registry is broken")
    }

    func testSetEnabledTrueRegistersAndClearsUserDisabled() throws {
        let registry = InMemoryLoginItemRegistry(initialStatus: .notRegistered)
        let prefs = InMemoryLoginItemPreferenceStore(userDisabled: true)
        let controller = LoginItemController(registry: registry, preferences: prefs)

        try controller.setEnabled(true)

        XCTAssertEqual(registry.registerCalls, 1)
        XCTAssertEqual(registry.currentStatus(), .enabled)
        XCTAssertFalse(prefs.userDisabled(),
                       "re-enabling clears the user-disabled flag")
    }

    func testSetEnabledFalseUnregistersAndRecordsUserDisabled() throws {
        let registry = InMemoryLoginItemRegistry(initialStatus: .enabled)
        let prefs = InMemoryLoginItemPreferenceStore(autoRegistered: true)
        let controller = LoginItemController(registry: registry, preferences: prefs)

        try controller.setEnabled(false)

        XCTAssertEqual(registry.unregisterCalls, 1)
        XCTAssertEqual(registry.currentStatus(), .notRegistered)
        XCTAssertTrue(prefs.userDisabled())
    }

    func testStatusPassesThrough() {
        let registry = InMemoryLoginItemRegistry(initialStatus: .requiresApproval)
        let controller = LoginItemController(
            registry: registry,
            preferences: InMemoryLoginItemPreferenceStore()
        )
        XCTAssertEqual(controller.status(), .requiresApproval)
    }
}
