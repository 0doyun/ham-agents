import Combine
import Foundation
import HamCore
import HamNotifications

@MainActor
public final class MenuBarViewModel: ObservableObject {
    @Published public private(set) var summary: HamMenuBarSummary?
    @Published public private(set) var agents: [Agent] = []
    @Published public private(set) var isRefreshing = false
    @Published public private(set) var errorMessage: String?
    @Published public private(set) var notificationPermissionStatus: NotificationPermissionStatus = .notDetermined
    @Published public private(set) var quickMessageFeedback: String?

    private let client: HamDaemonClientProtocol
    private let summaryService: MenuBarSummaryService
    private let notificationEngine: StatusChangeNotificationEngine
    private let notificationSink: NotificationSink
    private let notificationPermissionController: NotificationPermissionControlling
    private let projectOpener: ProjectOpening
    private let sessionOpener: SessionOpening
    private let quickMessageSender: QuickMessageSending
    private let pollIntervalNanoseconds: UInt64
    private let sleep: @Sendable (UInt64) async throws -> Void
    private var notificationOverrides: [Agent.ID: NotificationPolicy] = [:]
    private var hasStarted = false
    private var refreshTask: Task<Void, Never>?

    public init(
        client: HamDaemonClientProtocol,
        notificationEngine: StatusChangeNotificationEngine = StatusChangeNotificationEngine(),
        notificationSink: NotificationSink = NoopNotificationSink(),
        notificationPermissionController: NotificationPermissionControlling = NoopNotificationPermissionController(),
        projectOpener: ProjectOpening = NoopProjectOpener(),
        sessionOpener: SessionOpening = NoopSessionOpener(),
        quickMessageSender: QuickMessageSending = NoopQuickMessageSender(),
        pollIntervalNanoseconds: UInt64 = 15_000_000_000,
        sleep: @escaping @Sendable (UInt64) async throws -> Void = { nanoseconds in
            try await Task.sleep(nanoseconds: nanoseconds)
        }
    ) {
        self.client = client
        self.summaryService = MenuBarSummaryService(client: client)
        self.notificationEngine = notificationEngine
        self.notificationSink = notificationSink
        self.notificationPermissionController = notificationPermissionController
        self.projectOpener = projectOpener
        self.sessionOpener = sessionOpener
        self.quickMessageSender = quickMessageSender
        self.pollIntervalNanoseconds = pollIntervalNanoseconds
        self.sleep = sleep
    }

    public var statusLine: String {
        guard let summary else {
            return errorMessage == nil ? "ham idle" : "ham offline"
        }
        return "ham \(summary.runningAgents)▶ \(summary.waitingAgents)? \(summary.doneAgents)✓"
    }

    public func agent(withID id: Agent.ID?) -> Agent? {
        guard let id else { return agents.first }
        return agents.first(where: { $0.id == id }) ?? agents.first
    }

    public func recentEvents(forAgentID id: Agent.ID?) -> [AgentEventPayload] {
        let events = summary?.recentEvents ?? []
        guard let id else { return events }
        return events.filter { $0.agentID == id }
    }

    public func openProject(forAgentID id: Agent.ID?) {
        guard let agent = agent(withID: id) else { return }
        projectOpener.openProject(at: agent.projectPath)
    }

    public func openSession(forAgentID id: Agent.ID?) {
        guard let agent = agent(withID: id) else { return }
        sessionOpener.openSession(for: agent)
    }

    public func sendQuickMessage(_ message: String, forAgentID id: Agent.ID?) {
        guard let agent = agent(withID: id) else {
            quickMessageFeedback = "No agent selected."
            return
        }
        let trimmed = message.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            quickMessageFeedback = nil
            return
        }
        let result = quickMessageSender.send(message: trimmed, to: agent)
        switch result {
        case .delivered(let message), .handoff(let message), .failed(let message):
            quickMessageFeedback = message
        }
    }

    public func isNotificationsMuted(forAgentID id: Agent.ID?) -> Bool {
        agent(withID: id)?.notificationPolicy == .muted
    }

    public func toggleNotificationPause(forAgentID id: Agent.ID?) {
        guard let agent = agent(withID: id) else { return }
        let nextPolicy: NotificationPolicy = agent.notificationPolicy == .muted ? .default : .muted
        notificationOverrides[agent.id] = nextPolicy
        agents = applyNotificationOverrides(to: agents)
    }

    public func requestNotificationPermission() async {
        notificationPermissionStatus = await notificationPermissionController.requestPermission()
    }

    public func start() {
        guard !hasStarted else { return }
        hasStarted = true

        refreshTask = Task { [weak self] in
            guard let self else { return }
            await self.refresh()

            while !Task.isCancelled {
                do {
                    try await self.sleep(self.pollIntervalNanoseconds)
                } catch {
                    break
                }

                if Task.isCancelled {
                    break
                }

                await self.refresh()
            }
        }
    }

    public func stop() {
        refreshTask?.cancel()
        refreshTask = nil
        hasStarted = false
    }

    public func refresh(eventLimit: Int = 5) async {
        isRefreshing = true
        defer { isRefreshing = false }
        let previousAgents = agents

        do {
            async let loadedSummary = summaryService.refresh(eventLimit: eventLimit)
            async let loadedAgents = client.fetchAgents()
            async let permissionStatus = notificationPermissionController.currentPermissionStatus()

            let summaryValue = try await loadedSummary
            let loadedAgentsValue = applyNotificationOverrides(to: try await loadedAgents)
            let candidates = notificationEngine.candidates(previous: previousAgents, current: loadedAgentsValue)

            summary = summaryValue
            agents = loadedAgentsValue
            notificationPermissionStatus = await permissionStatus
            errorMessage = nil

            for candidate in candidates {
                notificationSink.send(candidate)
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    deinit {
        refreshTask?.cancel()
    }

    private func applyNotificationOverrides(to agents: [Agent]) -> [Agent] {
        agents.map { agent in
            guard let override = notificationOverrides[agent.id] else { return agent }
            var updated = agent
            updated.notificationPolicy = override
            return updated
        }
    }
}
