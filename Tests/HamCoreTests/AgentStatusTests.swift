import XCTest
@testable import HamCore

final class AgentStatusTests: XCTestCase {
    func testAllExpectedStatusesExist() {
        XCTAssertTrue(AgentStatus.allCases.contains(.booting))
        XCTAssertTrue(AgentStatus.allCases.contains(.waitingInput))
        XCTAssertTrue(AgentStatus.allCases.contains(.done))
    }

    func testManagedAgentModeExists() {
        XCTAssertEqual(AgentMode.managed.rawValue, "managed")
    }
}
