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
        viewModel.stop()
    }

    func testPollingRecoversAfterInitialFailure() async {
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
        let client = CyclingClient(agent: agent)
        let sleepController = SleepController()
        let viewModel = MenuBarViewModel(
            client: client,
            pollIntervalNanoseconds: 1,
            sleep: { nanoseconds in
                try await sleepController.sleep(nanoseconds: nanoseconds)
            }
        )

        viewModel.start()
        try? await Task.sleep(nanoseconds: 100_000_000)

        XCTAssertEqual(viewModel.summary?.totalAgents, 1)
        XCTAssertNil(viewModel.errorMessage)
        viewModel.stop()
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

private actor CyclingClient: HamDaemonClientProtocol {
    private let agent: Agent
    private var snapshotCalls = 0

    init(agent: Agent) {
        self.agent = agent
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        snapshotCalls += 1
        if snapshotCalls == 1 {
            throw HamDaemonClientError.transportFailed("unavailable")
        }
        return DaemonRuntimeSnapshotPayload(
            agents: [agent],
            generatedAt: Date(timeIntervalSince1970: 10)
        )
    }

    func fetchAgents() async throws -> [Agent] {
        [agent]
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        []
    }
}

private actor SleepController {
    private var calls = 0

    func sleep(nanoseconds: UInt64) async throws {
        _ = nanoseconds
        calls += 1
        if calls == 1 {
            return
        }
        try await Task.sleep(nanoseconds: 50_000_000)
    }
}
