import XCTest
@testable import HamAppServices
@testable import HamCore

final class PixelOfficeModelTests: XCTestCase {
    func testMapperRoutesStatusesToExpectedAreasAndSprites() {
        // Desk area: working statuses
        XCTAssertEqual(PixelOfficeMapper.area(for: .thinking), .desk)
        XCTAssertEqual(PixelOfficeMapper.area(for: .booting), .desk)
        XCTAssertEqual(PixelOfficeMapper.area(for: .runningTool), .desk)

        // Bookshelf area: reading
        XCTAssertEqual(PixelOfficeMapper.area(for: .reading), .bookshelf)

        // Alert area: attention needed
        XCTAssertEqual(PixelOfficeMapper.area(for: .waitingInput), .alertLight)
        XCTAssertEqual(PixelOfficeMapper.area(for: .error), .alertLight)
        XCTAssertEqual(PixelOfficeMapper.area(for: .disconnected), .alertLight)

        // Desk area: idle/sleeping stay at desk
        XCTAssertEqual(PixelOfficeMapper.area(for: .idle), .desk)
        XCTAssertEqual(PixelOfficeMapper.area(for: .sleeping), .desk)
        XCTAssertEqual(PixelOfficeMapper.area(for: .done), .desk)

        // Sprite mappings
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .runningTool), .type)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .waitingInput), .alert)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .error), .error)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .reading), .read)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .thinking), .think)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .sleeping), .sleep)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .idle), .idle)
        XCTAssertEqual(PixelOfficeMapper.sprite(for: .done), .idle)  // done falls back to idle
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
        XCTAssertNil(PixelOfficeMapper.statusIcon(for: .done))  // done no longer shows icon
        XCTAssertNil(PixelOfficeMapper.statusIcon(for: .thinking))
        XCTAssertNil(PixelOfficeMapper.statusIcon(for: .idle))
        XCTAssertNil(PixelOfficeMapper.statusIcon(for: .reading))
    }

    func testOccupantsFiltersDoneAgents() {
        let agents = [
            makeAgent(id: "a1", status: .thinking),
            makeAgent(id: "a2", status: .done),
            makeAgent(id: "a3", status: .reading),
        ]
        let occupants = PixelOfficeMapper.occupants(from: agents)
        XCTAssertEqual(occupants.count, 2)
        XCTAssertEqual(occupants.map(\.id), ["a1", "a3"])
    }

    func testOccupantIncludesSubAgentCount() {
        let agent = makeAgent(status: .thinking, subAgentCount: 3)
        let occupant = PixelOfficeMapper.occupant(for: agent)
        XCTAssertEqual(occupant.subAgentCount, 3)
        XCTAssertEqual(occupant.area, .desk)
    }

    func testThreeAreasOnly() {
        // Verify OfficeArea has exactly 3 cases
        XCTAssertEqual(OfficeArea.allCases.count, 3)
        XCTAssertTrue(OfficeArea.allCases.contains(.desk))
        XCTAssertTrue(OfficeArea.allCases.contains(.bookshelf))
        XCTAssertTrue(OfficeArea.allCases.contains(.alertLight))
    }

    private func makeAgent(id: String = "agent-1", status: AgentStatus, subAgentCount: Int = 0) -> Agent {
        Agent(
            id: id,
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
