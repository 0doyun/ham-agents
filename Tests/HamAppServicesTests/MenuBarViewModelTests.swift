import Foundation
import XCTest
@testable import HamAppServices
@testable import HamCore

@MainActor
final class MenuBarViewModelTests: XCTestCase {
    func testRefreshLoadsSummaryAndAgents() async {
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
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [agent],
                generatedAt: Date(timeIntervalSince1970: 10)
            ),
            events: [],
            agents: [agent]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.summary?.totalAgents, 1)
        XCTAssertEqual(viewModel.agents.count, 1)
        XCTAssertEqual(viewModel.statusLine, "ham 1▶ 0? 0✓")
        XCTAssertNil(viewModel.errorMessage)
    }

    func testRefreshCapturesErrors() async {
        let client = FailingClient()
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.statusLine, "ham offline")
        XCTAssertNotNil(viewModel.errorMessage)
    }

    func testStartTriggersInitialRefresh() async {
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
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [agent],
                generatedAt: Date(timeIntervalSince1970: 10)
            ),
            events: [],
            agents: [agent]
        )
        let viewModel = MenuBarViewModel(client: client)

        viewModel.start()
        try? await Task.sleep(nanoseconds: 100_000_000)

        XCTAssertEqual(viewModel.summary?.totalAgents, 1)
    }
}

private final class StubClient: HamDaemonClientProtocol, @unchecked Sendable {
    let snapshot: DaemonRuntimeSnapshotPayload
    let events: [AgentEventPayload]
    let agents: [Agent]

    init(snapshot: DaemonRuntimeSnapshotPayload, events: [AgentEventPayload], agents: [Agent]) {
        self.snapshot = snapshot
        self.events = events
        self.agents = agents
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload { snapshot }
    func fetchAgents() async throws -> [Agent] { agents }
    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] { events }
}

private struct FailingClient: HamDaemonClientProtocol, Sendable {
    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func fetchAgents() async throws -> [Agent] {
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        throw HamDaemonClientError.transportFailed("unavailable")
    }
}
