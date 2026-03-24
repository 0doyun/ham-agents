import Foundation
import XCTest
@testable import HamAppServices
@testable import HamCore
@testable import HamNotifications

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
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.displayName, "builder")
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
        let sleepController = CancellingSleepController()
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

    func testRefreshSendsNotificationForObservedDoneTransition() async {
        let previous = Agent(
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
        let current = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .done,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Build completed."
        )
        let client = TransitioningClient(initialAgents: [previous], nextAgents: [current])
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(client: client, notificationSink: sink)

        await viewModel.refresh()
        await viewModel.refresh()

        let sent = sink.candidates
        XCTAssertEqual(sent.count, 1)
        XCTAssertEqual(sent.first?.title, "builder finished")
    }

    func testRecentEventsFiltersBySelectedAgent() async {
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
        let otherAgent = Agent(
            id: "agent-2",
            displayName: "reviewer",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .done,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2)
        )
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [agent, otherAgent],
                generatedAt: Date(timeIntervalSince1970: 10)
            ),
            events: [
                AgentEventPayload(
                    id: "event-1",
                    agentID: "agent-1",
                    type: "agent.registered",
                    summary: "Managed session registered.",
                    occurredAt: Date(timeIntervalSince1970: 3)
                ),
                AgentEventPayload(
                    id: "event-2",
                    agentID: "agent-2",
                    type: "agent.registered",
                    summary: "Other agent registered.",
                    occurredAt: Date(timeIntervalSince1970: 4)
                ),
            ],
            agents: [agent, otherAgent]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.agent(withID: "agent-2")?.displayName, "reviewer")
        XCTAssertEqual(viewModel.recentEvents(forAgentID: "agent-2").count, 1)
        XCTAssertEqual(viewModel.recentEvents(forAgentID: "agent-2").first?.summary, "Other agent registered.")
    }

    func testOpenProjectUsesInjectedOpener() async {
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
        let opener = RecordingProjectOpener()
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            ),
            projectOpener: opener
        )

        await viewModel.refresh()
        viewModel.openProject(forAgentID: "agent-1")

        XCTAssertEqual(opener.openedPaths, ["/tmp/app"])
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

private actor CancellingSleepController {
    func sleep(nanoseconds: UInt64) async throws {
        _ = nanoseconds
        throw CancellationError()
    }
}

private actor TransitioningClient: HamDaemonClientProtocol {
    private let initialAgents: [Agent]
    private let nextAgents: [Agent]
    private var fetchAgentsCalls = 0

    init(initialAgents: [Agent], nextAgents: [Agent]) {
        self.initialAgents = initialAgents
        self.nextAgents = nextAgents
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        let agents = fetchAgentsCalls == 0 ? initialAgents : nextAgents
        return DaemonRuntimeSnapshotPayload(agents: agents, generatedAt: Date(timeIntervalSince1970: 10))
    }

    func fetchAgents() async throws -> [Agent] {
        defer { fetchAgentsCalls += 1 }
        return fetchAgentsCalls == 0 ? initialAgents : nextAgents
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        []
    }
}

private final class RecordingNotificationSink: NotificationSink, @unchecked Sendable {
    private let lock = NSLock()
    private(set) var candidates: [NotificationCandidate] = []

    func send(_ candidate: NotificationCandidate) {
        lock.lock()
        defer { lock.unlock() }
        candidates.append(candidate)
    }
}

private final class RecordingProjectOpener: ProjectOpening, @unchecked Sendable {
    private(set) var openedPaths: [String] = []

    func openProject(at path: String) {
        openedPaths.append(path)
    }
}
