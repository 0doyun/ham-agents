import XCTest
@testable import HamAppServices
@testable import HamCore

final class PixelOfficeModelTests: XCTestCase {
    func testMapperRoutesStatusesToExpectedAreasAndSprites() {
        XCTAssertEqual(PixelOfficeMapper.area(for: .thinking), .desk)
        XCTAssertEqual(PixelOfficeMapper.area(for: .booting), .desk)
        XCTAssertEqual(PixelOfficeMapper.area(for: .runningTool), .desk)
        XCTAssertEqual(PixelOfficeMapper.area(for: .reading), .bookshelf)
        XCTAssertEqual(PixelOfficeMapper.area(for: .waitingInput), .alertLight)
        XCTAssertEqual(PixelOfficeMapper.area(for: .error), .alertLight)
        XCTAssertEqual(PixelOfficeMapper.area(for: .disconnected), .alertLight)
        XCTAssertEqual(PixelOfficeMapper.area(for: .done), .sofa)
        XCTAssertEqual(PixelOfficeMapper.area(for: .idle), .sofa)
        XCTAssertEqual(PixelOfficeMapper.area(for: .sleeping), .sofa)

        XCTAssertEqual(PixelOfficeMapper.sprite(for: .runningTool), .type)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .waitingInput), .alert)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .error), .error)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .done), .celebrate)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .reading), .read)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .thinking), .think)
    }

    func testMenuBarStatePrioritizesErrorThenWaitingThenRunningThenDone() {
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: [makeAgent(status: .error)]), .error)
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: [makeAgent(status: .waitingInput)]), .waiting)
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: [makeAgent(status: .thinking)]), .running)
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: [makeAgent(status: .done)]), .done)
        XCTAssertEqual(PixelOfficeMapper.menuBarState(summary: nil, agents: []), .idle)
    }

    func testStatusIconMapping() {
        XCTAssertEqual(PixelOfficeMapper.statusIcon(for: .waitingInput), .question)
        XCTAssertEqual(PixelOfficeMapper.statusIcon(for: .error), .warning)
        XCTAssertEqual(PixelOfficeMapper.statusIcon(for: .disconnected), .warning)
        XCTAssertEqual(PixelOfficeMapper.statusIcon(for: .done), .check)
        XCTAssertNil(PixelOfficeMapper.statusIcon(for: .thinking))
        XCTAssertNil(PixelOfficeMapper.statusIcon(for: .idle))
        XCTAssertNil(PixelOfficeMapper.statusIcon(for: .reading))
    }

    func testOccupantIncludesSubAgentCount() {
        let agent = makeAgent(status: .thinking, subAgentCount: 3)
        let occupant = PixelOfficeMapper.occupant(for: agent)
        XCTAssertEqual(occupant.subAgentCount, 3)
        XCTAssertEqual(occupant.area, .desk)
    }

    private func makeAgent(status: AgentStatus, subAgentCount: Int = 0) -> Agent {
        Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: status,
            statusConfidence: 1,
            lastEventAt: .init(timeIntervalSince1970: 1),
            subAgentCount: subAgentCount
        )
    }
}
