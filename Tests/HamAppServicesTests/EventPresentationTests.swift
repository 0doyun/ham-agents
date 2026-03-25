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

    func testStatusUpdatedRunningToolGetsInfoPresentation() {
        let event = AgentEventPayload(
            id: "event-4c",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to running_tool. Observed tool-like activity.",
            occurredAt: Date(timeIntervalSince1970: 4)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Running Tool")
        XCTAssertEqual(presentation.emphasis, .info)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testStatusUpdatedReadingGetsInfoPresentation() {
        let event = AgentEventPayload(
            id: "event-4d",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to reading. Observed reading-like activity.",
            occurredAt: Date(timeIntervalSince1970: 4)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Reading")
        XCTAssertEqual(presentation.emphasis, .info)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testStatusUpdatedBootingGetsInfoPresentation() {
        let event = AgentEventPayload(
            id: "event-4g",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to booting. Observed booting-like activity.",
            occurredAt: Date(timeIntervalSince1970: 4)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Booting")
        XCTAssertEqual(presentation.emphasis, .info)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testStatusUpdatedThinkingGetsInfoPresentation() {
        let event = AgentEventPayload(
            id: "event-4e",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to thinking. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 4)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Thinking")
        XCTAssertEqual(presentation.emphasis, .info)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testStatusUpdatedSleepingGetsNeutralPresentation() {
        let event = AgentEventPayload(
            id: "event-4f",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to sleeping. Observed source idle for 10m.",
            occurredAt: Date(timeIntervalSince1970: 4)
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Sleeping")
        XCTAssertEqual(presentation.emphasis, .neutral)
        XCTAssertFalse(presentation.showsTechnicalType)
    }

    func testLowConfidenceLifecyclePresentationGetsLikelyPrefix() {
        let event = AgentEventPayload(
            id: "event-4b",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to waiting_input. Needs confirmation.",
            occurredAt: Date(timeIntervalSince1970: 4),
            lifecycleStatus: "waiting_input",
            lifecycleConfidence: 0.45
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Likely Needs Input")
        XCTAssertEqual(presentation.emphasis, .warning)
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

    func testLowConfidencePresentationHintAlsoGetsLikelyPrefix() {
        let event = AgentEventPayload(
            id: "event-7b",
            agentID: "agent-1",
            type: "agent.registered",
            summary: "Managed session registered.",
            occurredAt: Date(timeIntervalSince1970: 7),
            presentationLabel: "Managed",
            presentationEmphasis: "info",
            lifecycleConfidence: 0.4
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Likely Managed")
        XCTAssertEqual(presentation.emphasis, .info)
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

    func testLifecycleMetadataMapsRunningToolPresentation() {
        let event = AgentEventPayload(
            id: "event-9b",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to idle. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 9),
            lifecycleStatus: "running_tool"
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Running Tool")
        XCTAssertEqual(presentation.emphasis, .info)
    }

    func testLifecycleMetadataMapsReadingPresentation() {
        let event = AgentEventPayload(
            id: "event-9c",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to idle. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 9),
            lifecycleStatus: "reading"
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Reading")
        XCTAssertEqual(presentation.emphasis, .info)
    }

    func testLifecycleMetadataMapsBootingPresentation() {
        let event = AgentEventPayload(
            id: "event-9f",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to idle. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 9),
            lifecycleStatus: "booting"
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Booting")
        XCTAssertEqual(presentation.emphasis, .info)
    }

    func testLifecycleMetadataMapsThinkingPresentation() {
        let event = AgentEventPayload(
            id: "event-9d",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to idle. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 9),
            lifecycleStatus: "thinking"
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Thinking")
        XCTAssertEqual(presentation.emphasis, .info)
    }

    func testLifecycleMetadataMapsSleepingPresentation() {
        let event = AgentEventPayload(
            id: "event-9e",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to idle. Observed recent output.",
            occurredAt: Date(timeIntervalSince1970: 9),
            lifecycleStatus: "sleeping"
        )

        let presentation = AgentEventPresenter.present(event)

        XCTAssertEqual(presentation.label, "Sleeping")
        XCTAssertEqual(presentation.emphasis, .neutral)
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

    func testDisplaySummaryUsesLifecycleReasonWhenNoPresentationSummaryExists() {
        let event = AgentEventPayload(
            id: "event-10",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to waiting_input. Needs confirmation.",
            occurredAt: Date(timeIntervalSince1970: 10),
            lifecycleReason: "Needs confirmation.",
            lifecycleConfidence: 0.9
        )

        XCTAssertEqual(AgentEventPresenter.displaySummary(for: event), "Needs confirmation.")
    }

    func testDisplaySummarySoftensLowConfidenceLifecycleReason() {
        let event = AgentEventPayload(
            id: "event-11",
            agentID: "agent-1",
            type: "agent.status_updated",
            summary: "Status changed to waiting_input. Needs confirmation.",
            occurredAt: Date(timeIntervalSince1970: 11),
            lifecycleReason: "Needs confirmation.",
            lifecycleConfidence: 0.45
        )

        XCTAssertEqual(AgentEventPresenter.displaySummary(for: event), "Needs confirmation. (low confidence)")
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
