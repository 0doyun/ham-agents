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
        let permissions = RecordingNotificationPermissionController(initial: .authorized)
        let viewModel = MenuBarViewModel(client: client, notificationPermissionController: permissions)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.summary?.totalAgents, 1)
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
            lastEventAt: Date(timeIntervalSince1970: 1)
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
            lastEventAt: Date(timeIntervalSince1970: 1),
            sessionRef: "iterm2://session/abc",
            sessionTitle: "Claude Review",
            sessionIsActive: true,
            sessionTTY: "ttys001",
            sessionWorkingDirectory: "/tmp/app",
            sessionActivity: "claude"
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
        await viewModel.updateNotificationSetting(done: false, previewText: true)

        XCTAssertFalse(viewModel.settings.notifications.done)
        XCTAssertTrue(viewModel.settings.notifications.previewText)
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
        let counts = await client.callCounts()
        XCTAssertEqual(counts.fetchSettings, 1)
        XCTAssertEqual(counts.fetchEvents, 1)
        XCTAssertEqual(counts.followEvents, 1)
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
            now: { Date(timeIntervalSince1970: 14 * 60 * 60) },
            calendar: calendar
        )

        await viewModel.refresh()
        await viewModel.refresh()

        XCTAssertEqual(sink.candidates.count, 1)
        XCTAssertEqual(sink.candidates.first?.title, "builder hit an error")
    }
}

private final class StubClient: HamDaemonClientProtocol, @unchecked Sendable {
    let snapshot: DaemonRuntimeSnapshotPayload
    let events: [AgentEventPayload]
    let agents: [Agent]
    let attachableSessions: [DaemonAttachableSessionPayload]
    let settings: DaemonSettingsPayload

    init(
        snapshot: DaemonRuntimeSnapshotPayload,
        events: [AgentEventPayload],
        agents: [Agent],
        attachableSessions: [DaemonAttachableSessionPayload] = [],
        settings: DaemonSettingsPayload = .default
    ) {
        self.snapshot = snapshot
        self.events = events
        self.agents = agents
        self.attachableSessions = attachableSessions
        self.settings = settings
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload { snapshot }
    func fetchAgents() async throws -> [Agent] { agents }
    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] { attachableSessions }
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
    private var fetchAgentsCalls = 0
    private var policyOverride: NotificationPolicy?
    private let settings: DaemonSettingsPayload

    init(initialAgents: [Agent], nextAgents: [Agent], settings: DaemonSettingsPayload = .default) {
        self.initialAgents = initialAgents
        self.nextAgents = nextAgents
        self.settings = settings
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        let baseAgents = fetchAgentsCalls == 0 ? initialAgents : nextAgents
        let agents = applyPolicyOverride(to: baseAgents)
        return DaemonRuntimeSnapshotPayload(agents: agents, generatedAt: Date(timeIntervalSince1970: 10))
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
        return []
    }

    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] {
        []
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
    private var fetchEventsCalls = 0
    private var fetchSettingsCalls = 0
    private var followEventsCalls = 0

    init(agent: Agent, initialEvents: [AgentEventPayload], followedEvents: [AgentEventPayload]) {
        self.agent = agent
        self.initialEvents = initialEvents
        self.followedEvents = followedEvents
    }

    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        DaemonRuntimeSnapshotPayload(agents: [agent], generatedAt: Date(timeIntervalSince1970: 10))
    }

    func fetchAgents() async throws -> [Agent] {
        [agent]
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

    func callCounts() -> (fetchEvents: Int, fetchSettings: Int, followEvents: Int) {
        (fetchEventsCalls, fetchSettingsCalls, followEventsCalls)
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
