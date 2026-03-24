import Foundation
import XCTest
@testable import HamAppServices
@testable import HamCore

final class SessionTargetPlannerTests: XCTestCase {
    func testUsesExternalURLWhenSessionRefLooksLikeURL() {
        let planner = SessionTargetPlanner()
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1),
            sessionRef: "iterm2://session/abc"
        )

        let target = planner.target(for: agent)

        XCTAssertEqual(target, .externalURL(URL(string: "iterm2://session/abc")!))
    }

    func testFallsBackToWorkspaceWhenSessionRefMissing() {
        let planner = SessionTargetPlanner()
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )

        let target = planner.target(for: agent)

        XCTAssertEqual(target, .workspace(path: "/tmp/app"))
    }
}
