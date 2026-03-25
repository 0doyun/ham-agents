import XCTest
@testable import HamCore

final class AgentStatusPresentationTests: XCTestCase {
    func testHumanizedLabelSpecialCases() {
        XCTAssertEqual(AgentStatus.waitingInput.humanizedLabel, "needs input")
        XCTAssertEqual(AgentStatus.runningTool.humanizedLabel, "running tool")
        XCTAssertEqual(AgentStatus.disconnected.humanizedLabel, "disconnected")
    }

    func testRunningActivityMatchesExpectedStatuses() {
        XCTAssertTrue(AgentStatus.booting.isRunningActivity)
        XCTAssertTrue(AgentStatus.thinking.isRunningActivity)
        XCTAssertTrue(AgentStatus.reading.isRunningActivity)
        XCTAssertTrue(AgentStatus.runningTool.isRunningActivity)
        XCTAssertFalse(AgentStatus.idle.isRunningActivity)
        XCTAssertFalse(AgentStatus.done.isRunningActivity)
    }
}
