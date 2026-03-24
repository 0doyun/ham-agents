import Foundation
import XCTest
@testable import HamAppServices
@testable import HamCore

final class MenuBarSummaryServiceTests: XCTestCase {
    func testRefreshBuildsSummaryFromSnapshotAndEvents() async throws {
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [
                    Agent(
                        id: "a1",
                        displayName: "builder",
                        provider: "claude",
                        host: "localhost",
                        mode: .managed,
                        projectPath: "/tmp/app",
                        status: .thinking,
                        statusConfidence: 1,
                        lastEventAt: Date(timeIntervalSince1970: 1)
                    ),
                    Agent(
                        id: "a2",
                        displayName: "reviewer",
                        provider: "claude",
                        host: "localhost",
                        mode: .managed,
                        projectPath: "/tmp/app",
                        status: .done,
                        statusConfidence: 1,
                        lastEventAt: Date(timeIntervalSince1970: 2)
                    ),
                ],
                generatedAt: Date(timeIntervalSince1970: 10)
            ),
            events: [
                AgentEventPayload(
                    id: "event-1",
                    agentID: "a1",
                    type: "agent.registered",
                    summary: "Managed session registered.",
                    occurredAt: Date(timeIntervalSince1970: 3)
                )
            ]
        )
        let service = MenuBarSummaryService(client: client)

        let summary = try await service.refresh(eventLimit: 5)

        XCTAssertEqual(summary.totalAgents, 2)
        XCTAssertEqual(summary.runningAgents, 1)
        XCTAssertEqual(summary.doneAgents, 1)
        XCTAssertEqual(summary.waitingAgents, 0)
        XCTAssertEqual(summary.recentEvents.count, 1)
        XCTAssertEqual(client.requestedEventLimit, 5)
    }

    func testDefaultSocketPathPrefersEnvironmentOverrides() throws {
        let explicit = try DaemonEnvironment.defaultSocketPath(
            env: ["HAM_AGENTS_SOCKET": "/tmp/custom.sock"]
        )
        XCTAssertEqual(explicit, "/tmp/custom.sock")

        let derived = try DaemonEnvironment.defaultSocketPath(
            env: ["HAM_AGENTS_HOME": "/tmp/ham-home"]
        )
        XCTAssertEqual(derived, "/tmp/ham-home/hamd.sock")
    }
}

private final class StubClient: HamDaemonClientProtocol, @unchecked Sendable {
    let snapshot: DaemonRuntimeSnapshotPayload
    let events: [AgentEventPayload]
    private(set) var requestedEventLimit: Int?

    init(snapshot: DaemonRuntimeSnapshotPayload, events: [AgentEventPayload]) {
        self.snapshot = snapshot
        self.events = events
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        snapshot
    }

    func fetchAgents() async throws -> [Agent] {
        snapshot.agents
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        requestedEventLimit = limit
        return events
    }

    func fetchSettings() async throws -> DaemonSettingsPayload {
        DaemonSettingsPayload(
            notifications: DaemonNotificationSettingsPayload(
                done: true,
                error: true,
                waitingInput: true,
                quietHoursEnabled: false,
                quietHoursStartHour: 22,
                quietHoursEndHour: 8,
                previewText: false
            )
        )
    }

    func updateSettings(_ settings: DaemonSettingsPayload) async throws -> DaemonSettingsPayload {
        settings
    }

    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent {
        var agent = snapshot.agents.first { $0.id == agentID }!
        agent.notificationPolicy = policy
        return agent
    }

    func updateRole(agentID: String, role: String) async throws -> Agent {
        var agent = snapshot.agents.first { $0.id == agentID }!
        agent.role = role
        return agent
    }

    func removeAgent(agentID: String) async throws {
        _ = agentID
    }
}
