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
            lastEventAt: Date(timeIntervalSince1970: 1),
            teamRole: "lead",
            teamTaskTotal: 3,
            teamTaskCompleted: 1
        )
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [agent],
                generatedAt: Date(timeIntervalSince1970: 10),
                attentionCount: 1,
                attentionBreakdown: .init(error: 0, waitingInput: 1, disconnected: 0),
                attentionOrder: ["agent-1"],
                attentionSubtitles: ["agent-1": "needs input · high confidence · Needs confirmation."]
            ),
            events: [],
            agents: [agent]
        )
        let permissions = RecordingNotificationPermissionController(initial: .authorized)
        let viewModel = MenuBarViewModel(client: client, notificationPermissionController: permissions)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.summary?.totalAgents, 1)
        XCTAssertEqual(viewModel.summary?.attentionAgents, 1)
        XCTAssertEqual(viewModel.summary?.attentionBreakdown.waitingInput, 1)
        XCTAssertEqual(viewModel.summary?.attentionBreakdown.error, 0)
        XCTAssertEqual(viewModel.summary?.attentionOrder, ["agent-1"])
        XCTAssertEqual(viewModel.summary?.attentionSubtitles["agent-1"], "needs input · high confidence · Needs confirmation.")
        XCTAssertEqual(viewModel.topSummaryAttentionBreakdownChips.map(\.label), ["Needs Input"])
        XCTAssertEqual(viewModel.topSummaryAttentionBreakdownChips.map(\.count), [1])
        XCTAssertEqual(viewModel.attentionSubtitle(for: agent), "needs input · high confidence · Needs confirmation.")
        XCTAssertEqual(viewModel.agents.count, 1)
        XCTAssertEqual(viewModel.statusLine, "ham 1▶ 0? 0✓")
        XCTAssertNil(viewModel.errorMessage)
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.displayName, "builder")
        XCTAssertEqual(viewModel.notificationPermissionStatus, .authorized)
        XCTAssertEqual(viewModel.confidenceText(for: viewModel.agent(withID: "agent-1")), "100%")
    }

    func testRefreshLoadsAttachableSessions() async {
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
            teamRole: "lead",
            teamTaskTotal: 3,
            teamTaskCompleted: 1
        )
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [agent],
                generatedAt: Date(timeIntervalSince1970: 10)
            ),
            events: [],
            agents: [agent],
            attachableSessions: [
                DaemonAttachableSessionPayload(
                    id: "abc",
                    title: "Claude Review",
                    sessionRef: "iterm2://session/abc",
                    isActive: true
                )
            ]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.attachableSessions.count, 1)
        XCTAssertEqual(viewModel.attachableSessions.first?.id, "abc")
    }

    func testRefreshLoadsTeamsAndFiltersAgentsByTeamAndWorkspace() async {
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
            teamRole: "lead",
            teamTaskTotal: 3,
            teamTaskCompleted: 1
        )
        let otherAgent = Agent(
            id: "agent-2",
            displayName: "reviewer",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/other",
            status: .idle,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2)
        )
        let team = DaemonTeamPayload(id: "team-1", displayName: "alpha", memberAgentIDs: ["agent-1"])
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent, otherAgent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent, otherAgent],
                teams: [team]
            )
        )

        await viewModel.refresh()

        XCTAssertEqual(viewModel.teams.map(\.displayName), ["alpha"])
        XCTAssertEqual(viewModel.workspaceOptions, ["/tmp/app", "/tmp/other"])
        XCTAssertEqual(viewModel.filteredNonAttentionAgents(teamID: "team-1", workspace: nil).map(\.id), ["agent-1"])
        XCTAssertEqual(viewModel.filteredNonAttentionAgents(teamID: nil, workspace: "/tmp/other").map(\.id), ["agent-2"])
        XCTAssertEqual(viewModel.filteredOfficeOccupants(teamID: "team-1", workspace: nil).map(\.area), [.desk])
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.teamRole, "lead")
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.teamTaskTotal, 3)
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.teamTaskCompleted, 1)
    }

    func testRefreshRecordsNotificationHistory() async {
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
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Build failed."
        )
        let historyStore = InMemoryNotificationHistoryStore()
        let viewModel = MenuBarViewModel(
            client: TransitioningClient(initialAgents: [previous], nextAgents: [current]),
            notificationSink: RecordingNotificationSink(),
            notificationHistoryStore: historyStore
        )

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertEqual(viewModel.notificationHistory.last?.title, "builder hit an error")
        XCTAssertEqual(historyStore.load().last?.title, "builder hit an error")
    }

    func testRefreshSurfacesDisconnectedAttachedAgent() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "ops",
            provider: "iterm2",
            host: "localhost",
            mode: .attached,
            projectPath: "/tmp/app",
            status: .disconnected,
            statusConfidence: 0.75,
            lastEventAt: Date(timeIntervalSince1970: 1),
            sessionRef: "iterm2://session/abc"
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()

        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.status, .disconnected)
    }

    func testRefreshSurfacesAttachedSessionMetadata() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "ops",
            provider: "iterm2",
            host: "localhost",
            mode: .attached,
            projectPath: "/tmp/app",
            status: .idle,
            statusConfidence: 0.6,
            statusReason: "Attached to an existing iTerm session.",
            lastEventAt: Date(timeIntervalSince1970: 1),
            sessionRef: "iterm2://session/abc",
            sessionTitle: "Claude Review",
            sessionIsActive: true,
            sessionTTY: "ttys001",
            sessionWorkingDirectory: "/tmp/app",
            sessionActivity: "claude",
            sessionProcessID: 12345,
            sessionCommand: "/usr/local/bin/claude"
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()

        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.sessionTitle, "Claude Review")
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.sessionIsActive, true)
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.sessionTTY, "ttys001")
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.sessionWorkingDirectory, "/tmp/app")
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.sessionActivity, "claude")
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.sessionProcessID, 12345)
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.sessionCommand, "/usr/local/bin/claude")
        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.statusReason, "Attached to an existing iTerm session.")
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
        let client = TransitioningClient(
            initialAgents: [previous],
            nextAgents: [current],
            initialGeneratedAt: Date(timeIntervalSince1970: 10),
            nextGeneratedAt: Date(timeIntervalSince1970: 360)
        )
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(client: client, notificationSink: sink)

        await viewModel.refresh()
        await viewModel.refresh()

        let sent = sink.candidates
        XCTAssertEqual(sent.count, 1)
        XCTAssertEqual(sent.first?.title, "builder finished")
    }

    func testRefreshSendsTeamDigestWhenTeamNewlyNeedsAttention() async {
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
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Need approval."
        )
        let team = DaemonTeamPayload(id: "team-1", displayName: "alpha", memberAgentIDs: ["agent-1"])
        var settings = DaemonSettingsPayload.default
        settings.notifications.previewText = true
        let client = TransitioningClient(initialAgents: [previous], nextAgents: [current], teams: [team], settings: settings)
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(client: client, notificationSink: sink)

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.contains(where: { $0.title == "alpha needs attention" && $0.body.contains("needs input") }))
    }

    func testRefreshSendsTeamTaskCompletedNotification() async {
        let previous = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1),
            teamRole: "lead",
            teamTaskTotal: 2,
            teamTaskCompleted: 0
        )
        let current = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Task completed: write tests",
            teamRole: "lead",
            teamTaskTotal: 2,
            teamTaskCompleted: 1
        )
        let team = DaemonTeamPayload(id: "team-1", displayName: "alpha", memberAgentIDs: ["agent-1"])
        var settings = DaemonSettingsPayload.default
        settings.notifications.previewText = true
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(
            client: TransitioningClient(initialAgents: [previous], nextAgents: [current], teams: [team], settings: settings),
            notificationSink: sink
        )

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.contains(where: {
            $0.title == "alpha completed a task" && $0.body == "Task completed: write tests"
        }))
    }

    func testRefreshSendsTeamTaskCompletedNotificationsForMultiTeamMembership() async {
        let previous = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1),
            teamRole: "lead",
            teamTaskTotal: 2,
            teamTaskCompleted: 0
        )
        let current = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Task completed: write tests",
            teamRole: "lead",
            teamTaskTotal: 2,
            teamTaskCompleted: 1
        )
        let teams = [
            DaemonTeamPayload(id: "team-1", displayName: "alpha", memberAgentIDs: ["agent-1"]),
            DaemonTeamPayload(id: "team-2", displayName: "beta", memberAgentIDs: ["agent-1"]),
        ]
        var settings = DaemonSettingsPayload.default
        settings.notifications.previewText = true
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(
            client: TransitioningClient(initialAgents: [previous], nextAgents: [current], teams: teams, settings: settings),
            notificationSink: sink
        )

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.contains(where: {
            $0.title == "alpha completed a task" && $0.body == "Task completed: write tests"
        }))
        XCTAssertTrue(sink.candidates.contains(where: {
            $0.title == "beta completed a task" && $0.body == "Task completed: write tests"
        }))
    }

    func testRefreshSuppressesRepeatedAttentionNotificationWithinWindow() async {
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
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Need approval."
        )
        let recentEntry = NotificationHistoryEntry(
            key: "agent:agent-1:attention",
            title: "builder needs input",
            body: "Need approval.",
            createdAt: Date(timeIntervalSince1970: 30)
        )
        let historyStore = InMemoryNotificationHistoryStore(entries: [recentEntry])
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(
            client: TransitioningClient(initialAgents: [previous], nextAgents: [current]),
            notificationSink: sink,
            notificationHistoryStore: historyStore,
            now: { Date(timeIntervalSince1970: 60) }
        )

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.isEmpty)
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
        XCTAssertEqual(viewModel.recentEventSummaryChips(forAgentID: "agent-2").first?.label, "Managed")
        XCTAssertEqual(viewModel.recentEventSummaryChips(forAgentID: nil).first?.label, "Managed")
        XCTAssertEqual(viewModel.recentEventSummaryChips(forAgentID: nil).first?.count, 2)
    }

    func testRecentEventsPrioritizeWarningsOverInformationalRows() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [
                    AgentEventPayload(
                        id: "event-1",
                        agentID: "agent-1",
                        type: "agent.registered",
                        summary: "Registered.",
                        occurredAt: Date(timeIntervalSince1970: 3)
                    ),
                    AgentEventPayload(
                        id: "event-2",
                        agentID: "agent-1",
                        type: "agent.disconnected",
                        summary: "Disconnected.",
                        occurredAt: Date(timeIntervalSince1970: 1)
                    )
                ],
                agents: [agent]
            )
        )

        await viewModel.refresh()

        XCTAssertEqual(viewModel.recentEvents(forAgentID: "agent-1").map(\.id), ["event-2", "event-1"])
    }

    func testRecentEventsPrioritizeWarningStatusUpdatesOverRegistrations() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
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
                        agentID: "agent-1",
                        type: "agent.status_updated",
                        summary: "Status changed to waiting_input. Needs confirmation.",
                        occurredAt: Date(timeIntervalSince1970: 1)
                    )
                ],
                agents: [agent]
            )
        )

        await viewModel.refresh()

        XCTAssertEqual(viewModel.recentEvents(forAgentID: "agent-1").map(\.id), ["event-2", "event-1"])
        XCTAssertEqual(viewModel.recentEventSeverityChips(forAgentID: "agent-1").map(\.label), ["Needs Attention", "Info"])
    }

    func testRecentEventSeverityChipsPrioritizeWarningsBeforeInfo() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [
                    AgentEventPayload(
                        id: "event-1",
                        agentID: "agent-1",
                        type: "agent.registered",
                        summary: "Registered.",
                        occurredAt: Date(timeIntervalSince1970: 3)
                    ),
                    AgentEventPayload(
                        id: "event-2",
                        agentID: "agent-1",
                        type: "agent.disconnected",
                        summary: "Disconnected.",
                        occurredAt: Date(timeIntervalSince1970: 1)
                    )
                ],
                agents: [agent]
            )
        )

        await viewModel.refresh()

        XCTAssertEqual(viewModel.recentEventSeverityChips(forAgentID: "agent-1").map(\.label), ["Needs Attention", "Info"])
        XCTAssertEqual(viewModel.recentEventSeverityChips(forAgentID: "agent-1").map(\.count), [1, 1])
    }

    func testConfidenceTextRoundsPercentage() {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .attached,
            projectPath: "/tmp/app",
            status: .reading,
            statusConfidence: 0.72,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(client: FailingClient())

        XCTAssertEqual(viewModel.confidenceText(for: agent), "72%")
        XCTAssertEqual(viewModel.confidenceLevelText(for: agent), "Medium")
        XCTAssertEqual(viewModel.statusDisplayText(for: agent), "reading")
        XCTAssertEqual(viewModel.confidenceSummaryText(for: agent), "medium confidence (72%)")
    }

    func testLowConfidenceStatusUsesLikelyLanguage() {
        let agent = Agent(
            id: "agent-1",
            displayName: "observer",
            provider: "log",
            host: "localhost",
            mode: .observed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 0.45,
            statusReason: "Question-like output detected.",
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(client: FailingClient())

        XCTAssertEqual(viewModel.confidenceLevelText(for: agent), "Low")
        XCTAssertEqual(viewModel.statusDisplayText(for: agent), "likely needs input")
        XCTAssertEqual(viewModel.confidenceSummaryText(for: agent), "low confidence (45%)")
    }

    func testRunningToolStatusDisplayUsesHumanizedLabel() {
        let agent = Agent(
            id: "agent-rt",
            displayName: "builder",
            provider: "codex",
            host: "localhost",
            mode: .observed,
            projectPath: "/tmp/app",
            status: .runningTool,
            statusConfidence: 0.8,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(client: FailingClient())

        XCTAssertEqual(viewModel.statusDisplayText(for: agent), "running tool")
    }

    func testAttentionAgentsArePrioritizedByStatus() async {
        let errorAgent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .error,
            statusConfidence: 1,
            statusReason: "Build failed.",
            lastEventAt: Date(timeIntervalSince1970: 3)
        )
        let waitingAgent = Agent(
            id: "agent-2",
            displayName: "reviewer",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            statusReason: "Needs confirmation.",
            lastEventAt: Date(timeIntervalSince1970: 2)
        )
        let thinkingAgent = Agent(
            id: "agent-3",
            displayName: "observer",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [thinkingAgent, waitingAgent, errorAgent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [thinkingAgent, waitingAgent, errorAgent]
            )
        )

        await viewModel.refresh()

        XCTAssertEqual(viewModel.attentionAgents.map(\.id), ["agent-1", "agent-2"])
        XCTAssertEqual(viewModel.nonAttentionAgents.map(\.id), ["agent-3"])
        XCTAssertEqual(viewModel.attentionSubtitle(for: errorAgent), "error · high confidence · Build failed.")
        XCTAssertEqual(viewModel.attentionSubtitle(for: waitingAgent), "needs input · high confidence · Needs confirmation.")
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

    func testOpenSessionUsesInjectedOpener() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            role: "reviewer",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1),
            sessionRef: "session-1"
        )
        let opener = RecordingSessionOpener()
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            ),
            sessionOpener: opener
        )

        await viewModel.refresh()
        viewModel.openSession(forAgentID: "agent-1")

        XCTAssertEqual(opener.openedAgentIDs, ["agent-1"])
    }

    func testHandleNotificationInteractionSelectsAgent() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        viewModel.handleNotificationInteraction(.focusAgent("agent-1"))

        XCTAssertEqual(viewModel.selectedAgentID, "agent-1")
    }

    func testHandleNotificationInteractionCanOpenTerminal() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1),
            sessionRef: "iterm2://session/abc"
        )
        let opener = RecordingSessionOpener()
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            ),
            sessionOpener: opener
        )

        await viewModel.refresh()
        viewModel.handleNotificationInteraction(.openTerminal("agent-1"))

        XCTAssertEqual(viewModel.selectedAgentID, "agent-1")
        XCTAssertEqual(opener.openedAgentIDs, ["agent-1"])
    }

    func testOpenSessionDoesNotUseOpenerWhenItermIntegrationDisabled() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "iterm2",
            host: "localhost",
            mode: .attached,
            projectPath: "/tmp/app",
            status: .idle,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let opener = RecordingSessionOpener()
        let settings = DaemonSettingsPayload(
            notifications: .init(
                done: true,
                error: true,
                waitingInput: true,
                quietHoursEnabled: false,
                quietHoursStartHour: 22,
                quietHoursEndHour: 8,
                previewText: false
            ),
            appearance: .default,
            integrations: .init(itermEnabled: false)
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent],
                settings: settings
            ),
            sessionOpener: opener
        )

        await viewModel.refresh()
        viewModel.openSession(forAgentID: "agent-1")

        XCTAssertTrue(opener.openedAgentIDs.isEmpty)
        XCTAssertEqual(viewModel.errorMessage, "Enable iTerm integration in Settings to open sessions.")
    }

    func testRequestNotificationPermissionUpdatesPublishedStatus() async {
        let client = FailingClient()
        let permissions = RecordingNotificationPermissionController(initial: .notDetermined, requestResult: .authorized)
        let viewModel = MenuBarViewModel(client: client, notificationPermissionController: permissions)

        await viewModel.requestNotificationPermission()

        XCTAssertEqual(viewModel.notificationPermissionStatus, .authorized)
    }

    func testUpdateNotificationSettingsChangesPublishedSettings() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateNotificationSetting(done: false, silence: true, previewText: true)

        XCTAssertFalse(viewModel.settings.notifications.done)
        XCTAssertTrue(viewModel.settings.notifications.silence)
        XCTAssertTrue(viewModel.settings.notifications.previewText)
    }

    func testUpdateNotificationSettingsCanSetHeartbeatMinutes() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateNotificationSetting(heartbeatMinutes: 30)

        XCTAssertEqual(viewModel.settings.notifications.heartbeatMinutes, 30)
    }

    func testUpdateNotificationSettingsCanToggleQuietHours() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateNotificationSetting(
            quietHoursEnabled: true,
            quietHoursStartHour: 21,
            quietHoursEndHour: 7
        )

        XCTAssertTrue(viewModel.settings.notifications.quietHoursEnabled)
        XCTAssertEqual(viewModel.settings.notifications.quietHoursStartHour, 21)
        XCTAssertEqual(viewModel.settings.notifications.quietHoursEndHour, 7)
    }

    func testUpdateNotificationSettingsCanToggleSilence() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateNotificationSetting(silence: true)

        XCTAssertTrue(viewModel.settings.notifications.silence)
    }

    func testRefreshSendsHeartbeatNotificationWhenConfigured() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            registeredAt: Date(timeIntervalSince1970: 0),
            lastEventAt: Date(timeIntervalSince1970: 60),
            lastUserVisibleSummary: "Read: spec.md",
            omcMode: "ralph"
        )
        let settings = DaemonSettingsPayload(
            notifications: DaemonNotificationSettingsPayload(
                done: true,
                error: true,
                waitingInput: true,
                silence: false,
                quietHoursEnabled: false,
                quietHoursStartHour: 22,
                quietHoursEndHour: 8,
                previewText: true,
                heartbeatMinutes: 10
            )
        )
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 20 * 60)
                ),
                events: [],
                agents: [agent],
                settings: settings
            ),
            notificationSink: sink
        )

        await viewModel.refresh()

        XCTAssertEqual(sink.candidates.first?.title, "builder is still running")
    }

    func testUpdateAppearanceSettingsChangesPublishedTheme() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateAppearanceSetting(theme: "night")

        XCTAssertEqual(viewModel.settings.appearance.theme, "night")
    }

    func testUpdateAppearanceSettingsChangesAnimationControls() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateAppearanceSetting(animationSpeedMultiplier: 1.5)
        await viewModel.updateAppearanceSetting(reduceMotion: true)

        XCTAssertEqual(viewModel.settings.appearance.animationSpeedMultiplier, 1.5)
        XCTAssertTrue(viewModel.settings.appearance.reduceMotion)
    }

    func testUpdateAppearanceSettingsChangesDecorationChoices() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(agents: [agent], generatedAt: Date(timeIntervalSince1970: 10)),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateAppearanceSetting(hamsterSkin: "golden", hat: "cap", deskTheme: "night-shift")

        XCTAssertEqual(viewModel.settings.appearance.hamsterSkin, "golden")
        XCTAssertEqual(viewModel.settings.appearance.hat, "cap")
        XCTAssertEqual(viewModel.settings.appearance.deskTheme, "night-shift")
    }

    func testUpdateGeneralSettingsChangesPublishedValues() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(agents: [agent], generatedAt: Date(timeIntervalSince1970: 10)),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateGeneralSetting(launchAtLogin: true, compactMode: true, showMenuBarAnimationAlways: true)

        XCTAssertTrue(viewModel.settings.general.launchAtLogin)
        XCTAssertTrue(viewModel.settings.general.compactMode)
        XCTAssertTrue(viewModel.settings.general.showMenuBarAnimationAlways)
    }

    func testUpdateIntegrationSettingsChangesPublishedValue() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateIntegrationSetting(itermEnabled: false)

        XCTAssertFalse(viewModel.settings.integrations.itermEnabled)
    }

    func testUpdatePrivacySettingsChangesPublishedValues() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(agents: [agent], generatedAt: Date(timeIntervalSince1970: 10)),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updatePrivacySetting(localOnlyMode: false, eventHistoryRetentionDays: 14, transcriptExcerptStorage: false)

        XCTAssertFalse(viewModel.settings.privacy.localOnlyMode)
        XCTAssertEqual(viewModel.settings.privacy.eventHistoryRetentionDays, 14)
        XCTAssertFalse(viewModel.settings.privacy.transcriptExcerptStorage)
    }

    func testUpdateIntegrationSettingsChangesTranscriptDirsAndProviderAdapters() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateIntegrationSetting(
            transcriptDirs: ["/tmp/a", "/tmp/b"],
            providerAdapters: ["claude": true, "transcript": false]
        )

        XCTAssertEqual(viewModel.settings.integrations.transcriptDirs, ["/tmp/a", "/tmp/b"])
        XCTAssertEqual(viewModel.settings.integrations.providerAdapters["transcript"], false)
    }

    func testUpdateGeneralAndPrivacySettingsChangePublishedValues() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateGeneralSetting(launchAtLogin: true, compactMode: true, showMenuBarAnimationAlways: true)
        await viewModel.updatePrivacySetting(localOnlyMode: false, eventHistoryRetentionDays: 14, transcriptExcerptStorage: false)

        XCTAssertTrue(viewModel.settings.general.launchAtLogin)
        XCTAssertTrue(viewModel.settings.general.compactMode)
        XCTAssertTrue(viewModel.settings.general.showMenuBarAnimationAlways)
        XCTAssertFalse(viewModel.settings.privacy.localOnlyMode)
        XCTAssertEqual(viewModel.settings.privacy.eventHistoryRetentionDays, 14)
        XCTAssertFalse(viewModel.settings.privacy.transcriptExcerptStorage)
    }

    func testUpdateAppearanceSettingsChangesCustomizationFields() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.updateAppearanceSetting(hamsterSkin: "night", hat: "cap", deskTheme: "forest")

        XCTAssertEqual(viewModel.settings.appearance.hamsterSkin, "night")
        XCTAssertEqual(viewModel.settings.appearance.hat, "cap")
        XCTAssertEqual(viewModel.settings.appearance.deskTheme, "forest")
    }

    func testFollowLatestEventsRefreshesSummaryWhenNewEventsArrive() async {
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
        let initialEvent = AgentEventPayload(
            id: "event-1",
            agentID: "agent-1",
            type: "agent.registered",
            summary: "Registered",
            occurredAt: Date(timeIntervalSince1970: 1)
        )
        let followedEvent = AgentEventPayload(
            id: "event-2",
            agentID: "agent-1",
            type: "agent.registered",
            summary: "Followed",
            occurredAt: Date(timeIntervalSince1970: 2)
        )
        let client = EventFollowingClient(
            agent: agent,
            initialEvents: [initialEvent],
            followedEvents: [followedEvent]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()
        await viewModel.followLatestEvents(eventLimit: 5, waitMilliseconds: 1)

        XCTAssertEqual(viewModel.summary?.recentEvents.last?.id, "event-2")
        XCTAssertEqual(viewModel.summary?.attentionBreakdown.error, 0)
        XCTAssertEqual(viewModel.summary?.attentionBreakdown.waitingInput, 0)
        let counts = await client.callCounts()
        XCTAssertEqual(counts.fetchSnapshot, 1)
        XCTAssertEqual(counts.fetchAgents, 2)
        XCTAssertEqual(counts.fetchSettings, 1)
        XCTAssertEqual(counts.fetchEvents, 1)
        XCTAssertEqual(counts.followEvents, 1)
    }

    func testFollowLatestEventsRebuildsAttentionBreakdownFromFetchedAgents() async {
        var followedAgent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2)
        )
        followedAgent.statusReason = "Needs confirmation."

        let client = TransitioningClient(
            initialAgents: [
                Agent(
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
            ],
            nextAgents: [followedAgent],
            followedEvents: [
                AgentEventPayload(
                    id: "event-2",
                    agentID: "agent-1",
                    type: "agent.status_updated",
                    summary: "Needs confirmation.",
                    occurredAt: Date(timeIntervalSince1970: 2)
                )
            ]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()
        await viewModel.followLatestEvents(eventLimit: 5, waitMilliseconds: 1)

        XCTAssertEqual(viewModel.summary?.attentionAgents, 1)
        XCTAssertEqual(viewModel.summary?.attentionBreakdown.waitingInput, 1)
        XCTAssertEqual(viewModel.summary?.attentionOrder, ["agent-1"])
        XCTAssertEqual(viewModel.attentionAgents.map(\.id), ["agent-1"])
        XCTAssertEqual(viewModel.summary?.attentionSubtitles["agent-1"], "needs input · high confidence · Needs confirmation.")
        XCTAssertEqual(viewModel.attentionSubtitle(for: followedAgent), "needs input · high confidence · Needs confirmation.")
        XCTAssertEqual(viewModel.topSummaryAttentionBreakdownChips.map(\.label), ["Needs Input"])
    }

    func testAttentionAgentsPreferDaemonProvidedOrdering() async {
        let errorAgent = Agent(
            id: "agent-1",
            displayName: "erroring",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 3)
        )
        let waitingAgent = Agent(
            id: "agent-2",
            displayName: "waiting",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [errorAgent, waitingAgent],
                generatedAt: Date(timeIntervalSince1970: 10),
                attentionCount: 2,
                attentionBreakdown: .init(error: 1, waitingInput: 1, disconnected: 0),
                attentionOrder: ["agent-2", "agent-1"]
            ),
            events: [],
            agents: [errorAgent, waitingAgent]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.attentionAgents.map(\.id), ["agent-2", "agent-1"])
    }

    func testAttentionAgentsFallbackUsesDeterministicIDTiebreak() async {
        let firstAgent = Agent(
            id: "agent-1",
            displayName: "same",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let secondAgent = Agent(
            id: "agent-2",
            displayName: "same",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [secondAgent, firstAgent],
                generatedAt: Date(timeIntervalSince1970: 10),
                attentionCount: 2,
                attentionBreakdown: .init(error: 0, waitingInput: 2, disconnected: 0),
                attentionOrder: []
            ),
            events: [],
            agents: [secondAgent, firstAgent]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.attentionAgents.map(\.id), ["agent-1", "agent-2"])
    }

    func testAttentionAgentsUseFallbackForIDsMissingFromDaemonOrder() async {
        let providedAgent = Agent(
            id: "agent-1",
            displayName: "provided",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2)
        )
        let missingAgent = Agent(
            id: "agent-2",
            displayName: "missing",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [providedAgent, missingAgent],
                generatedAt: Date(timeIntervalSince1970: 10),
                attentionCount: 2,
                attentionBreakdown: .init(error: 1, waitingInput: 1, disconnected: 0),
                attentionOrder: ["agent-1"]
            ),
            events: [],
            agents: [providedAgent, missingAgent]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.attentionAgents.map(\.id), ["agent-1", "agent-2"])
    }

    func testFollowLatestEventsRebuildsAttentionOrderFromFetchedAgents() async {
        let initialAgent = Agent(
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
        let errorAgent = Agent(
            id: "agent-2",
            displayName: "erroring",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2)
        )
        let waitingAgent = Agent(
            id: "agent-1",
            displayName: "waiting",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .waitingInput,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 3)
        )
        let client = TransitioningClient(
            initialAgents: [initialAgent],
            nextAgents: [waitingAgent, errorAgent],
            followedEvents: [
                AgentEventPayload(
                    id: "event-2",
                    agentID: "agent-2",
                    type: "agent.disconnected",
                    summary: "Disconnected.",
                    occurredAt: Date(timeIntervalSince1970: 4)
                )
            ]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()
        await viewModel.followLatestEvents(eventLimit: 5, waitMilliseconds: 1)

        XCTAssertEqual(viewModel.summary?.attentionOrder, ["agent-2", "agent-1"])
        XCTAssertEqual(viewModel.attentionAgents.map(\.id), ["agent-2", "agent-1"])
    }

    func testStatusLineReflectsLatestWarningEvent() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "ops",
            provider: "iterm2",
            host: "localhost",
            mode: .attached,
            projectPath: "/tmp/app",
            status: .disconnected,
            statusConfidence: 0.75,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [agent],
                generatedAt: Date(timeIntervalSince1970: 10)
            ),
            events: [
                AgentEventPayload(
                    id: "event-1",
                    agentID: "agent-1",
                    type: "agent.disconnected",
                    summary: "Attached session disappeared from iTerm.",
                    occurredAt: Date(timeIntervalSince1970: 2)
                )
            ],
            agents: [agent]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.latestEventPresentation?.label, "Disconnected")
        XCTAssertEqual(viewModel.latestEventSummary, "Attached session disappeared from iTerm.")
        XCTAssertTrue(viewModel.statusLine.hasPrefix("⚠︎ ham"))
    }

    func testLatestEventSummaryPrefersLifecycleReasonFallback() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "ops",
            provider: "iterm2",
            host: "localhost",
            mode: .attached,
            projectPath: "/tmp/app",
            status: .disconnected,
            statusConfidence: 0.45,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let client = StubClient(
            snapshot: DaemonRuntimeSnapshotPayload(
                agents: [agent],
                generatedAt: Date(timeIntervalSince1970: 10)
            ),
            events: [
                AgentEventPayload(
                    id: "event-1",
                    agentID: "agent-1",
                    type: "agent.status_updated",
                    summary: "Status changed to waiting_input. Needs confirmation.",
                    occurredAt: Date(timeIntervalSince1970: 2),
                    lifecycleReason: "Needs confirmation.",
                    lifecycleConfidence: 0.45
                )
            ],
            agents: [agent]
        )
        let viewModel = MenuBarViewModel(client: client)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.latestEventSummary, "Needs confirmation. (low confidence)")
    }

    func testSaveRoleUpdatesSelectedAgent() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            role: "reviewer",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        viewModel.setRoleDraft(from: "agent-1")
        viewModel.roleDraft = "lead"
        await viewModel.saveRole(forAgentID: "agent-1")

        XCTAssertEqual(viewModel.agent(withID: "agent-1")?.role, "lead")
        XCTAssertEqual(viewModel.roleDraft, "lead")
    }

    func testStopTrackingRemovesSelectedAgent() async {
        let agent = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            role: "reviewer",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        await viewModel.stopTracking(forAgentID: "agent-1")

        XCTAssertTrue(viewModel.agents.isEmpty)
    }

    func testSendQuickMessageUsesInjectedSender() async {
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
        let sender = RecordingQuickMessageSender()
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            ),
            quickMessageSender: sender
        )

        await viewModel.refresh()
        viewModel.sendQuickMessage("  please check logs  ", forAgentID: "agent-1")

        XCTAssertEqual(sender.sentMessages.count, 1)
        XCTAssertEqual(sender.sentMessages.first?.0, "agent-1")
        XCTAssertEqual(sender.sentMessages.first?.1, "please check logs")
        XCTAssertEqual(viewModel.quickMessageFeedback, "Sent.")
    }

    func testSendQuickMessageIgnoresBlankDraft() async {
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
        let sender = RecordingQuickMessageSender()
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            ),
            quickMessageSender: sender
        )

        await viewModel.refresh()
        viewModel.sendQuickMessage("   ", forAgentID: "agent-1")

        XCTAssertTrue(sender.sentMessages.isEmpty)
        XCTAssertNil(viewModel.quickMessageFeedback)
    }

    func testSendQuickMessageWithoutSelectedAgentSetsFailureFeedback() {
        let viewModel = MenuBarViewModel(client: FailingClient())

        viewModel.sendQuickMessage("hello", forAgentID: nil)

        XCTAssertEqual(viewModel.quickMessageFeedback, "No agent selected.")
    }

    func testToggleNotificationPauseUpdatesSelectedAgentPolicy() async {
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
        let viewModel = MenuBarViewModel(
            client: StubClient(
                snapshot: DaemonRuntimeSnapshotPayload(
                    agents: [agent],
                    generatedAt: Date(timeIntervalSince1970: 10)
                ),
                events: [],
                agents: [agent]
            )
        )

        await viewModel.refresh()
        viewModel.toggleNotificationPause(forAgentID: "agent-1")
        try? await Task.sleep(nanoseconds: 100_000_000)

        XCTAssertTrue(viewModel.isNotificationsMuted(forAgentID: "agent-1"))
    }

    func testMutedOverrideSuppressesLaterDoneNotification() async {
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
        viewModel.toggleNotificationPause(forAgentID: "agent-1")
        try? await Task.sleep(nanoseconds: 100_000_000)
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.isEmpty)
        XCTAssertTrue(viewModel.isNotificationsMuted(forAgentID: "agent-1"))
    }

    func testNotificationSettingsCanSuppressDoneNotifications() async {
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
        let settings = DaemonSettingsPayload(
            notifications: DaemonNotificationSettingsPayload(
                done: false,
                error: true,
                waitingInput: true,
                quietHoursEnabled: false,
                quietHoursStartHour: 22,
                quietHoursEndHour: 8,
                previewText: true
            )
        )
        let client = TransitioningClient(initialAgents: [previous], nextAgents: [current], settings: settings)
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(client: client, notificationSink: sink)

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.isEmpty)
    }

    func testDoneNotificationRequiresLongRunningTask() async {
        let previous = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 100)
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
            lastEventAt: Date(timeIntervalSince1970: 101),
            lastUserVisibleSummary: "Build completed."
        )
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(
            client: TransitioningClient(initialAgents: [previous], nextAgents: [current]),
            notificationSink: sink,
            now: { Date(timeIntervalSince1970: 160) }
        )

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.isEmpty)
    }

    func testNotificationSettingsCanMaskPreviewText() async {
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
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Secret failure detail"
        )
        let settings = DaemonSettingsPayload(
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
        let client = TransitioningClient(initialAgents: [previous], nextAgents: [current], settings: settings)
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(client: client, notificationSink: sink)

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertEqual(sink.candidates.first?.body, "Open ham-menubar for details.")
    }

    func testNotificationSettingsCanMaskSilencePreviewText() async {
        let previousObservedAt = Date(timeIntervalSince1970: 1_000)
        let currentObservedAt = Date(timeIntervalSince1970: 1_120)
        let lastEventAt = Date(timeIntervalSince1970: 1_000 - (9 * 60))

        let previous = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: lastEventAt,
            lastUserVisibleSummary: "Observed tool-like activity."
        )
        let current = previous
        let settings = DaemonSettingsPayload(
            notifications: DaemonNotificationSettingsPayload(
                done: true,
                error: true,
                waitingInput: true,
                silence: true,
                quietHoursEnabled: false,
                quietHoursStartHour: 22,
                quietHoursEndHour: 8,
                previewText: false
            )
        )
        let client = TransitioningClient(
            initialAgents: [previous],
            nextAgents: [current],
            settings: settings,
            initialGeneratedAt: previousObservedAt,
            nextGeneratedAt: currentObservedAt
        )
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(client: client, notificationSink: sink)

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertEqual(sink.candidates.count, 1)
        XCTAssertEqual(sink.candidates.first?.title, "builder went quiet")
        XCTAssertEqual(sink.candidates.first?.body, "Open ham-menubar for details.")
    }

    func testQuietHoursSuppressNotificationCandidatesInsideOvernightWindow() async {
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
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Build failed."
        )
        let settings = DaemonSettingsPayload(
            notifications: DaemonNotificationSettingsPayload(
                done: true,
                error: true,
                waitingInput: true,
                quietHoursEnabled: true,
                quietHoursStartHour: 22,
                quietHoursEndHour: 8,
                previewText: true
            )
        )
        let client = TransitioningClient(initialAgents: [previous], nextAgents: [current], settings: settings)
        let sink = RecordingNotificationSink()
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 0)!
        let viewModel = MenuBarViewModel(
            client: client,
            notificationSink: sink,
            now: { Date(timeIntervalSince1970: 23 * 60 * 60) },
            calendar: calendar
        )

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.isEmpty)
    }

    func testQuietHoursAllowsNotificationsOutsideWindow() async {
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
            status: .error,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 2),
            lastUserVisibleSummary: "Build failed."
        )
        let settings = DaemonSettingsPayload(
            notifications: DaemonNotificationSettingsPayload(
                done: true,
                error: true,
                waitingInput: true,
                quietHoursEnabled: true,
                quietHoursStartHour: 22,
                quietHoursEndHour: 8,
                previewText: true
            )
        )
        let client = TransitioningClient(initialAgents: [previous], nextAgents: [current], settings: settings)
        let sink = RecordingNotificationSink()
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 0)!
        let viewModel = MenuBarViewModel(
            client: client,
            notificationSink: sink,
            notificationHistoryStore: InMemoryNotificationHistoryStore(),
            now: { Date(timeIntervalSince1970: 14 * 60 * 60) },
            calendar: calendar
        )

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertEqual(sink.candidates.count, 1)
        XCTAssertEqual(sink.candidates.first?.title, "builder hit an error")
    }

    func testNotificationSettingsCanSuppressSilenceNotifications() async {
        let previousObservedAt = Date(timeIntervalSince1970: 1_000)
        let currentObservedAt = Date(timeIntervalSince1970: 1_120)
        let lastEventAt = Date(timeIntervalSince1970: 1_000 - (9 * 60))

        let previous = Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .thinking,
            statusConfidence: 1,
            lastEventAt: lastEventAt
        )
        let current = previous
        let settings = DaemonSettingsPayload(
            notifications: DaemonNotificationSettingsPayload(
                done: true,
                error: true,
                waitingInput: true,
                silence: false,
                quietHoursEnabled: false,
                quietHoursStartHour: 22,
                quietHoursEndHour: 8,
                previewText: true
            )
        )
        let client = TransitioningClient(
            initialAgents: [previous],
            nextAgents: [current],
            settings: settings,
            initialGeneratedAt: previousObservedAt,
            nextGeneratedAt: currentObservedAt
        )
        let sink = RecordingNotificationSink()
        let viewModel = MenuBarViewModel(client: client, notificationSink: sink)

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertTrue(sink.candidates.isEmpty)
    }
}

private final class StubClient: HamDaemonClientProtocol, @unchecked Sendable {
    let snapshot: DaemonRuntimeSnapshotPayload
    let events: [AgentEventPayload]
    let agents: [Agent]
    let attachableSessions: [DaemonAttachableSessionPayload]
    let teams: [DaemonTeamPayload]
    let settings: DaemonSettingsPayload

    init(
        snapshot: DaemonRuntimeSnapshotPayload,
        events: [AgentEventPayload],
        agents: [Agent],
        attachableSessions: [DaemonAttachableSessionPayload] = [],
        teams: [DaemonTeamPayload] = [],
        settings: DaemonSettingsPayload = .default
    ) {
        self.snapshot = snapshot
        self.events = events
        self.agents = agents
        self.attachableSessions = attachableSessions
        self.teams = teams
        self.settings = settings
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload { snapshot }
    func fetchAgents() async throws -> [Agent] { agents }
    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] { attachableSessions }
    func fetchTeams() async throws -> [DaemonTeamPayload] { teams }
    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] { events }
    func followEvents(afterEventID: String, limit: Int, waitMilliseconds: Int) async throws -> [AgentEventPayload] {
        _ = afterEventID
        _ = limit
        _ = waitMilliseconds
        return []
    }

    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent {
        var agent = agents.first { $0.id == agentID } ?? snapshot.agents.first!
        agent.notificationPolicy = policy
        return agent
    }

    func fetchSettings() async throws -> DaemonSettingsPayload {
        settings
    }

    func updateSettings(_ settings: DaemonSettingsPayload) async throws -> DaemonSettingsPayload {
        settings
    }

    func updateRole(agentID: String, role: String) async throws -> Agent {
        var agent = agents.first { $0.id == agentID } ?? snapshot.agents.first!
        agent.role = role
        return agent
    }

    func removeAgent(agentID: String) async throws {
        _ = agentID
    }
}

private struct FailingClient: HamDaemonClientProtocol, Sendable {
    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func fetchAgents() async throws -> [Agent] {
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] {
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func followEvents(afterEventID: String, limit: Int, waitMilliseconds: Int) async throws -> [AgentEventPayload] {
        _ = afterEventID
        _ = limit
        _ = waitMilliseconds
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func fetchSettings() async throws -> DaemonSettingsPayload {
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func updateSettings(_ settings: DaemonSettingsPayload) async throws -> DaemonSettingsPayload {
        _ = settings
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent {
        _ = agentID
        _ = policy
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func updateRole(agentID: String, role: String) async throws -> Agent {
        _ = agentID
        _ = role
        throw HamDaemonClientError.transportFailed("unavailable")
    }

    func removeAgent(agentID: String) async throws {
        _ = agentID
        throw HamDaemonClientError.transportFailed("unavailable")
    }
}

private actor CyclingClient: HamDaemonClientProtocol {
    private let agent: Agent
    private var snapshotCalls = 0
    private let settings: DaemonSettingsPayload

    init(agent: Agent, settings: DaemonSettingsPayload = .default) {
        self.agent = agent
        self.settings = settings
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

    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] {
        []
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        []
    }

    func followEvents(afterEventID: String, limit: Int, waitMilliseconds: Int) async throws -> [AgentEventPayload] {
        _ = afterEventID
        _ = limit
        _ = waitMilliseconds
        return []
    }

    func fetchSettings() async throws -> DaemonSettingsPayload {
        settings
    }

    func updateSettings(_ settings: DaemonSettingsPayload) async throws -> DaemonSettingsPayload {
        settings
    }

    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent {
        _ = agentID
        var updated = agent
        updated.notificationPolicy = policy
        return updated
    }

    func updateRole(agentID: String, role: String) async throws -> Agent {
        _ = agentID
        var updated = agent
        updated.role = role
        return updated
    }

    func removeAgent(agentID: String) async throws {
        _ = agentID
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
    private let followedEvents: [AgentEventPayload]
    private let teams: [DaemonTeamPayload]
    private var fetchAgentsCalls = 0
    private var policyOverride: NotificationPolicy?
    private let settings: DaemonSettingsPayload
    private let initialGeneratedAt: Date
    private let nextGeneratedAt: Date

    init(
        initialAgents: [Agent],
        nextAgents: [Agent],
        followedEvents: [AgentEventPayload] = [],
        teams: [DaemonTeamPayload] = [],
        settings: DaemonSettingsPayload = .default,
        initialGeneratedAt: Date = Date(timeIntervalSince1970: 10),
        nextGeneratedAt: Date = Date(timeIntervalSince1970: 10)
    ) {
        self.initialAgents = initialAgents
        self.nextAgents = nextAgents
        self.followedEvents = followedEvents
        self.teams = teams
        self.settings = settings
        self.initialGeneratedAt = initialGeneratedAt
        self.nextGeneratedAt = nextGeneratedAt
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        let baseAgents = fetchAgentsCalls == 0 ? initialAgents : nextAgents
        let agents = applyPolicyOverride(to: baseAgents)
        let generatedAt = fetchAgentsCalls == 0 ? initialGeneratedAt : nextGeneratedAt
        return DaemonRuntimeSnapshotPayload(agents: agents, generatedAt: generatedAt)
    }

    func fetchAgents() async throws -> [Agent] {
        defer { fetchAgentsCalls += 1 }
        let baseAgents = fetchAgentsCalls == 0 ? initialAgents : nextAgents
        return applyPolicyOverride(to: baseAgents)
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        []
    }

    func followEvents(afterEventID: String, limit: Int, waitMilliseconds: Int) async throws -> [AgentEventPayload] {
        _ = afterEventID
        _ = limit
        _ = waitMilliseconds
        return followedEvents
    }

    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] {
        []
    }

    func fetchTeams() async throws -> [DaemonTeamPayload] {
        teams
    }

    func fetchSettings() async throws -> DaemonSettingsPayload {
        settings
    }

    func updateSettings(_ settings: DaemonSettingsPayload) async throws -> DaemonSettingsPayload {
        settings
    }

    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent {
        let agent = applyPolicyOverride(to: nextAgents).first { $0.id == agentID }
            ?? applyPolicyOverride(to: initialAgents).first!
        policyOverride = policy
        var updated = agent
        updated.notificationPolicy = policy
        return updated
    }

    func updateRole(agentID: String, role: String) async throws -> Agent {
        _ = agentID
        var updated = applyPolicyOverride(to: nextAgents).first ?? applyPolicyOverride(to: initialAgents).first!
        updated.role = role
        return updated
    }

    func removeAgent(agentID: String) async throws {
        _ = agentID
    }

    private func applyPolicyOverride(to agents: [Agent]) -> [Agent] {
        guard let policyOverride else { return agents }
        return agents.map { agent in
            var updated = agent
            updated.notificationPolicy = policyOverride
            return updated
        }
    }
}

private actor EventFollowingClient: HamDaemonClientProtocol {
    private let agent: Agent
    private let initialEvents: [AgentEventPayload]
    private let followedEvents: [AgentEventPayload]
    private var didFollow = false
    private var fetchSnapshotCalls = 0
    private var fetchAgentsCalls = 0
    private var fetchEventsCalls = 0
    private var fetchSettingsCalls = 0
    private var followEventsCalls = 0

    init(agent: Agent, initialEvents: [AgentEventPayload], followedEvents: [AgentEventPayload]) {
        self.agent = agent
        self.initialEvents = initialEvents
        self.followedEvents = followedEvents
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        fetchSnapshotCalls += 1
        return DaemonRuntimeSnapshotPayload(agents: [agent], generatedAt: Date(timeIntervalSince1970: 10))
    }

    func fetchAgents() async throws -> [Agent] {
        fetchAgentsCalls += 1
        return [agent]
    }

    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] {
        []
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        _ = limit
        fetchEventsCalls += 1
        return didFollow ? followedEvents : initialEvents
    }

    func followEvents(afterEventID: String, limit: Int, waitMilliseconds: Int) async throws -> [AgentEventPayload] {
        _ = afterEventID
        _ = limit
        _ = waitMilliseconds
        followEventsCalls += 1
        didFollow = true
        return followedEvents
    }

    func fetchSettings() async throws -> DaemonSettingsPayload {
        fetchSettingsCalls += 1
        return .default
    }

    func updateSettings(_ settings: DaemonSettingsPayload) async throws -> DaemonSettingsPayload {
        settings
    }

    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent {
        _ = agentID
        var updated = agent
        updated.notificationPolicy = policy
        return updated
    }

    func updateRole(agentID: String, role: String) async throws -> Agent {
        _ = agentID
        var updated = agent
        updated.role = role
        return updated
    }

    func removeAgent(agentID: String) async throws {
        _ = agentID
    }

    func callCounts() -> (fetchSnapshot: Int, fetchAgents: Int, fetchEvents: Int, fetchSettings: Int, followEvents: Int) {
        (fetchSnapshotCalls, fetchAgentsCalls, fetchEventsCalls, fetchSettingsCalls, followEventsCalls)
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

private final class RecordingSessionOpener: SessionOpening, @unchecked Sendable {
    private(set) var openedAgentIDs: [String] = []

    func openSession(for agent: Agent) {
        openedAgentIDs.append(agent.id)
    }
}

private final class RecordingQuickMessageSender: QuickMessageSending, @unchecked Sendable {
    private(set) var sentMessages: [(String, String)] = []

    func send(message: String, to agent: Agent) -> QuickMessageResult {
        sentMessages.append((agent.id, message))
        return .delivered("Sent.")
    }
}

private actor RecordingNotificationPermissionController: NotificationPermissionControlling {
    private var current: NotificationPermissionStatus
    private let requestResult: NotificationPermissionStatus

    init(initial: NotificationPermissionStatus, requestResult: NotificationPermissionStatus? = nil) {
        self.current = initial
        self.requestResult = requestResult ?? initial
    }

    func currentPermissionStatus() async -> NotificationPermissionStatus {
        current
    }

    func requestPermission() async -> NotificationPermissionStatus {
        current = requestResult
        return current
    }
}
