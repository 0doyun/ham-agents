import Foundation
import XCTest
@testable import HamCore
@testable import HamPersistence
@testable import HamRuntime

final class RuntimeTests: XCTestCase {
    func testRegisterAddsAgentToSnapshot() {
        let store = InMemoryAgentStore()
        let runtime = RuntimeRegistry(store: store)
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "codex",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/project",
            status: .thinking,
            statusConfidence: 1.0,
            lastEventAt: Date()
        )

        runtime.register(agent)

        let snapshot = runtime.snapshot()
        XCTAssertEqual(snapshot.totalCount, 1)
        XCTAssertEqual(snapshot.runningCount, 1)
        XCTAssertEqual(snapshot.agents.first?.displayName, "builder")
    }
}
