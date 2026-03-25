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
        XCTAssertFalse(presentation.showsTechnicalType)
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
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testStatusUpdatedEventGetsInfoPresentation() {
        let event = AgentEventPayload(
            id: "event-3",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to idle. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 3)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Idle")
        XCTAssertEqual(presentation.emphasis, .info)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testStatusUpdatedWaitingInputGetsWarningPresentation() {
        let event = AgentEventPayload(
            id: "event-4",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to waiting_input. Needs confirmation.",
            occurredAt: Date(timeIntervalSince1970: 4)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Needs Input")
        XCTAssertEqual(presentation.emphasis, .warning)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testManagedRegisteredEventGetsManagedPresentation() {
        let event = AgentEventPayload(
            id: "event-5",
            agentID: "agent-1",
            type: "agent.registered",
            summary: "Managed session registered.",
            occurredAt: Date(timeIntervalSince1970: 5)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Managed")
        XCTAssertEqual(presentation.emphasis, .info)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testAttachedRegisteredEventGetsAttachedPresentation() {
        let event = AgentEventPayload(
            id: "event-6",
            agentID: "agent-1",
            type: "agent.registered",
            summary: "Attached session registered.",
            occurredAt: Date(timeIntervalSince1970: 6)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Attached")
        XCTAssertEqual(presentation.emphasis, .info)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testUnknownEventKeepsTechnicalTypeVisible() {
        let event = AgentEventPayload(
            id: "event-5",
            agentID: "agent-1",
            type: "agent.custom_event",
            summary: "Custom event.",
            occurredAt: Date(timeIntervalSince1970: 4)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "agent.custom_event")
        XCTAssertTrue(presentation.showsTechnicalType)
    }

    func testPresentationHintOverridesSummaryInference() {
        let event = AgentEventPayload(
            id: "event-7",
            agentID: "agent-1",
            type: "agent.registered",
            summary: "Managed session registered.",
            occurredAt: Date(timeIntervalSince1970: 7),
            presentationLabel: "Observed",
            presentationEmphasis: "info",
            presentationSummary: "Observed source registered."
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Observed")
        XCTAssertEqual(presentation.emphasis, .info)
        XCTAssertFalse(presentation.showsTechnicalType)
        XCTAssertEqual(AgentEventPresenter.displaySummary(for: event), "Observed source registered.")
    }

    func testLifecycleMetadataOverridesSummaryInference() {
        let event = AgentEventPayload(
            id: "event-9",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to idle. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 9),
            lifecycleStatus: "error"
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Error")
        XCTAssertEqual(presentation.emphasis, .warning)
    }

    func testDisplaySummaryFallsBackToRawSummary() {
        let event = AgentEventPayload(
            id: "event-8",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to idle. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 8)
        )

        XCTAssertEqual(AgentEventPresenter.displaySummary(for: event), "Status changed to idle. Observed recent output.")
    }

    func testSummarizeGroupsEventsByPresentation() {
        let events = [
            AgentEventPayload(
                id: "event-1",
                agentID: "agent-1",
                type: "agent.disconnected",
                summary: "Attached session disappeared from iTerm.",
                occurredAt: Date(timeIntervalSince1970: 1)
            ),
            AgentEventPayload(
                id: "event-2",
                agentID: "agent-1",
                type: "agent.disconnected",
                summary: "Attached session disappeared from iTerm.",
                occurredAt: Date(timeIntervalSince1970: 2)
            ),
            AgentEventPayload(
                id: "event-3",
                agentID: "agent-1",
                type: "agent.reconnected",
                summary: "Attached session became reachable again.",
                occurredAt: Date(timeIntervalSince1970: 3)
            ),
        ]

        let summary = AgentEventPresenter.summarize(events)

        XCTAssertEqual(summary.count, 2)
        XCTAssertEqual(summary.first?.label, "Disconnected")
        XCTAssertEqual(summary.first?.count, 2)
        XCTAssertEqual(summary.last?.label, "Reconnected")
        XCTAssertEqual(summary.last?.count, 1)
    }

    func testSummarizeBySeverityGroupsWarningsBeforePositiveThenInfo() {
        let events = [
            AgentEventPayload(
                id: "event-1",
                agentID: "agent-1",
                type: "agent.registered",
                summary: "Registered.",
                occurredAt: Date(timeIntervalSince1970: 1)
            ),
            AgentEventPayload(
                id: "event-2",
                agentID: "agent-1",
                type: "agent.disconnected",
                summary: "Disconnected.",
                occurredAt: Date(timeIntervalSince1970: 2)
            ),
            AgentEventPayload(
                id: "event-3",
                agentID: "agent-1",
                type: "agent.disconnected",
                summary: "Disconnected.",
                occurredAt: Date(timeIntervalSince1970: 3)
            ),
            AgentEventPayload(
                id: "event-4",
                agentID: "agent-1",
                type: "agent.reconnected",
                summary: "Reconnected.",
                occurredAt: Date(timeIntervalSince1970: 4)
            ),
        ]

        let summary = AgentEventPresenter.summarizeBySeverity(events)

        XCTAssertEqual(summary.map(\.label), ["Needs Attention", "Positive", "Info"])
        XCTAssertEqual(summary.map(\.count), [2, 1, 1])
    }

    func testOrderedPrioritizesWarningsBeforeInfoThenRecency() {
        let infoEvent = AgentEventPayload(
            id: "event-1",
            agentID: "agent-1",
            type: "agent.registered",
            summary: "Registered.",
            occurredAt: Date(timeIntervalSince1970: 3)
        )
        let warningEvent = AgentEventPayload(
            id: "event-2",
            agentID: "agent-1",
            type: "agent.disconnected",
            summary: "Disconnected.",
            occurredAt: Date(timeIntervalSince1970: 1)
        )
        let positiveEvent = AgentEventPayload(
            id: "event-3",
            agentID: "agent-1",
            type: "agent.reconnected",
            summary: "Reconnected.",
            occurredAt: Date(timeIntervalSince1970: 2)
        )

        let ordered = AgentEventPresenter.ordered([infoEvent, positiveEvent, warningEvent])

        XCTAssertEqual(ordered.map(\.id), ["event-2", "event-3", "event-1"])
    }
}
