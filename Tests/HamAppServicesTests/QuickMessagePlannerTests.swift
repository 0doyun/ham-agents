import Foundation
import XCTest
@testable import HamAppServices
@testable import HamCore

final class QuickMessagePlannerTests: XCTestCase {
    func testTerminalWriteUsesSessionRefTargetWhenAutomationAvailable() {
        let planner = QuickMessagePlanner()
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

        let plan = planner.plan(message: "hello", for: agent, supportsTerminalAutomation: true)

        XCTAssertEqual(
            plan,
            .terminalWrite(
                target: .itermSession(id: "abc", url: URL(string: "iterm2://session/abc")!),
                message: "hello"
            )
        )
    }

    func testClipboardFallbackWhenAutomationUnavailable() {
        let planner = QuickMessagePlanner()
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

        let plan = planner.plan(message: "hello", for: agent, supportsTerminalAutomation: false)

        XCTAssertEqual(plan, .clipboardHandoff(message: "hello"))
    }
}
