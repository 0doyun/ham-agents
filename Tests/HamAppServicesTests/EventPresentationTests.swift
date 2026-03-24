import Foundation
import XCTest
@testable import HamAppServices
@testable import HamCore

final class EventPresentationTests: XCTestCase {
    func testDisconnectedEventGetsWarningPresentation() {
        let event = AgentEventPayload(
            id: "event-1",
            agentID: "agent-1",
            type: "agent.disconnected",
            summary: "Attached session disappeared from iTerm.",
            occurredAt: Date(timeIntervalSince1970: 1)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Disconnected")
        XCTAssertEqual(presentation.emphasis, .warning)
    }

    func testReconnectedEventGetsPositivePresentation() {
        let event = AgentEventPayload(
            id: "event-2",
            agentID: "agent-1",
            type: "agent.reconnected",
            summary: "Attached session became reachable again.",
            occurredAt: Date(timeIntervalSince1970: 2)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Reconnected")
        XCTAssertEqual(presentation.emphasis, .positive)
    }
}
