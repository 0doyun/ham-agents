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

    private func makeAgent(
        status: AgentStatus,
        summary: String? = nil,
        policy: NotificationPolicy = .default
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
            lastEventAt: Date(timeIntervalSince1970: 1),
            lastUserVisibleSummary: summary,
            notificationPolicy: policy
        )
    }
}
