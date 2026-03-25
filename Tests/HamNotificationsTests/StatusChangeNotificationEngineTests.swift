import Foundation
import XCTest
@testable import HamCore
@testable import HamNotifications

final class StatusChangeNotificationEngineTests: XCTestCase {
    func testCandidateProducedForObservedDoneTransition() {
        let previous = [makeAgent(status: .thinking)]
        let current = [makeAgent(status: .done, summary: "Finished.")]
        let engine = StatusChangeNotificationEngine()

        let candidates = engine.candidates(previous: previous, current: current)

        XCTAssertEqual(candidates.count, 1)
        XCTAssertEqual(candidates[0].title, "builder finished")
    }

    func testNoCandidateWhenStatusDidNotChange() {
        let previous = [makeAgent(status: .done)]
        let current = [makeAgent(status: .done)]
        let engine = StatusChangeNotificationEngine()

        XCTAssertTrue(engine.candidates(previous: previous, current: current).isEmpty)
    }

    func testNoCandidateForMutedAgent() {
        let previous = [makeAgent(status: .thinking, policy: .muted)]
        let current = [makeAgent(status: .error, policy: .muted)]
        let engine = StatusChangeNotificationEngine()

        XCTAssertTrue(engine.candidates(previous: previous, current: current).isEmpty)
    }

    func testWaitingInputFallbackBodyUsesHumanizedStatus() {
        let previous = [makeAgent(status: .thinking)]
        let current = [makeAgent(status: .waitingInput, summary: nil)]
        let engine = StatusChangeNotificationEngine()

        let candidates = engine.candidates(previous: previous, current: current)

        XCTAssertEqual(candidates.count, 1)
        XCTAssertEqual(candidates[0].body, "needs input at /tmp/app")
    }

    func testCandidateProducedWhenActiveAgentCrossesSilenceThreshold() {
        let previousObservedAt = Date(timeIntervalSince1970: 1_000)
        let currentObservedAt = Date(timeIntervalSince1970: 1_120)
        let lastEventAt = Date(timeIntervalSince1970: 1_000 - (9 * 60))
        let previous = [makeAgent(status: .thinking, lastEventAt: lastEventAt)]
        let current = [makeAgent(status: .thinking, lastEventAt: lastEventAt)]
        let engine = StatusChangeNotificationEngine()

        let candidates = engine.candidates(
            previous: previous,
            current: current,
            previousObservedAt: previousObservedAt,
            currentObservedAt: currentObservedAt
        )

        XCTAssertEqual(candidates.count, 1)
        XCTAssertEqual(candidates[0].title, "builder went quiet")
        XCTAssertEqual(candidates[0].body, "No activity for 11m at /tmp/app")
    }

    func testNoRepeatedSilenceCandidateAfterThresholdAlreadyCrossed() {
        let previousObservedAt = Date(timeIntervalSince1970: 1_000)
        let currentObservedAt = Date(timeIntervalSince1970: 1_120)
        let lastEventAt = Date(timeIntervalSince1970: 1_000 - (11 * 60))
        let previous = [makeAgent(status: .reading, lastEventAt: lastEventAt)]
        let current = [makeAgent(status: .reading, lastEventAt: lastEventAt)]
        let engine = StatusChangeNotificationEngine()

        XCTAssertTrue(
            engine.candidates(
                previous: previous,
                current: current,
                previousObservedAt: previousObservedAt,
                currentObservedAt: currentObservedAt
            ).isEmpty
        )
    }

    func testSilenceCandidateUsesLastSeenSummaryWhenAvailable() {
        let previousObservedAt = Date(timeIntervalSince1970: 1_000)
        let currentObservedAt = Date(timeIntervalSince1970: 1_120)
        let lastEventAt = Date(timeIntervalSince1970: 1_000 - (9 * 60))
        let previous = [makeAgent(status: .runningTool, summary: "Observed tool-like activity.", lastEventAt: lastEventAt)]
        let current = [makeAgent(status: .runningTool, summary: "Observed tool-like activity.", lastEventAt: lastEventAt)]
        let engine = StatusChangeNotificationEngine()

        let candidates = engine.candidates(
            previous: previous,
            current: current,
            previousObservedAt: previousObservedAt,
            currentObservedAt: currentObservedAt
        )

        XCTAssertEqual(candidates.count, 1)
        XCTAssertEqual(candidates[0].body, "No activity for 11m. Last seen: Observed tool-like activity.")
    }

    private func makeAgent(
        status: AgentStatus,
        summary: String? = nil,
        policy: NotificationPolicy = .default,
        lastEventAt: Date = Date(timeIntervalSince1970: 1)
    ) -> Agent {
        Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: status,
            statusConfidence: 1,
            lastEventAt: lastEventAt,
            lastUserVisibleSummary: summary,
            notificationPolicy: policy
        )
    }
}
