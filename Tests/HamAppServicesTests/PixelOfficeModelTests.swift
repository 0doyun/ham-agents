import XCTest
@testable import HamAppServices
@testable import HamCore

final class PixelOfficeModelTests: XCTestCase {
    func testMapperRoutesStatusesToExpectedZonesAndSprites() {
        XCTAssertEqual(PixelOfficeMapper.zone(for: .thinking), .desk)
        XCTAssertEqual(PixelOfficeMapper.zone(for: .reading), .library)
        XCTAssertEqual(PixelOfficeMapper.zone(for: .waitingInput), .alertCorner)
        XCTAssertEqual(PixelOfficeMapper.zone(for: .done), .kitchen)

        XCTAssertEqual(PixelOfficeMapper.sprite(for: .runningTool), .type)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .waitingInput), .alert)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .error), .error)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .done), .celebrate)
    }

    func testMenuBarStatePrioritizesErrorThenWaitingThenRunningThenDone() {
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: [makeAgent(status: .error)]), .error)
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: [makeAgent(status: .waitingInput)]), .waiting)
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: [makeAgent(status: .thinking)]), .running)
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: [makeAgent(status: .done)]), .done)
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: []), .idle)
    }

    private func makeAgent(status: AgentStatus) -> Agent {
        Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: status,
            statusConfidence: 1,
            lastEventAt: .init(timeIntervalSince1970: 1)
        )
    }
}
