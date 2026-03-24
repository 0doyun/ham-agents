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
    @Published public var roleDraft = ""
    @Published public private(set) var settings = DaemonSettingsPayload(
        notifications: DaemonNotificationSettingsPayload(
            done: true,
            error: true,
            waitingInput: true,
            quietHoursEnabled: false,
            previewText: false
        )
    )

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

    public func setRoleDraft(from agentID: Agent.ID?) {
        roleDraft = agent(withID: agentID)?.role ?? ""
    }

    public func recentEvents(forAgentID id: Agent.ID?) -> [AgentEventPayload] {
        let events = summary?.recentEvents ?? []
        guard let id else { return events }
        return events.filter { $0.agentID == id }
    }

    public func confidenceText(for agent: Agent?) -> String {
        guard let agent else { return "—" }
        return "\(Int((agent.statusConfidence * 100).rounded()))%"
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
        guard let id, let agent = agent(withID: id) else { return }
        let nextPolicy: NotificationPolicy = agent.notificationPolicy == .muted ? .default : .muted
        Task { [weak self] in
            guard let self else { return }
            do {
                let updated = try await client.updateNotificationPolicy(agentID: id, policy: nextPolicy)
                if let index = agents.firstIndex(where: { $0.id == updated.id }) {
                    agents[index] = updated
                }
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    public func requestNotificationPermission() async {
        notificationPermissionStatus = await notificationPermissionController.requestPermission()
    }

    public func updateNotificationSetting(
        done: Bool? = nil,
        error: Bool? = nil,
        waitingInput: Bool? = nil,
        previewText: Bool? = nil
    ) async {
        var updated = settings
        if let done { updated.notifications.done = done }
        if let error { updated.notifications.error = error }
        if let waitingInput { updated.notifications.waitingInput = waitingInput }
        if let previewText { updated.notifications.previewText = previewText }

        do {
            settings = try await client.updateSettings(updated)
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    public func saveRole(forAgentID id: Agent.ID?) async {
        guard let id else {
            errorMessage = "No agent selected."
            return
        }

        do {
            let updated = try await client.updateRole(agentID: id, role: roleDraft)
            if let index = agents.firstIndex(where: { $0.id == updated.id }) {
                agents[index] = updated
            }
            roleDraft = updated.role ?? ""
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    public func stopTracking(forAgentID id: Agent.ID?) async {
        guard let id else {
            errorMessage = "No agent selected."
            return
        }

        do {
            try await client.removeAgent(agentID: id)
            agents.removeAll { $0.id == id }
            roleDraft = agent(withID: nil)?.role ?? ""
            quickMessageFeedback = nil
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
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
            async let loadedSettings = client.fetchSettings()
            async let permissionStatus = notificationPermissionController.currentPermissionStatus()

            let summaryValue = try await loadedSummary
            let loadedAgentsValue = try await loadedAgents
            let loadedSettingsValue = try await loadedSettings
            let candidates = filteredNotificationCandidates(
                notificationEngine.candidates(previous: previousAgents, current: loadedAgentsValue),
                settings: loadedSettingsValue
            )

            summary = summaryValue
            agents = loadedAgentsValue
            settings = loadedSettingsValue
            notificationPermissionStatus = await permissionStatus
            if roleDraft.isEmpty, let firstAgent = agents.first {
                roleDraft = firstAgent.role ?? ""
            }
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

    private func filteredNotificationCandidates(
        _ candidates: [NotificationCandidate],
        settings: DaemonSettingsPayload
    ) -> [NotificationCandidate] {
        candidates.compactMap { candidate in
            switch candidate.event {
            case .done:
                guard settings.notifications.done else { return nil }
            case .error:
                guard settings.notifications.error else { return nil }
            case .waitingInput:
                guard settings.notifications.waitingInput else { return nil }
            }

            guard settings.notifications.previewText else {
                return NotificationCandidate(
                    event: candidate.event,
                    title: candidate.title,
                    body: "Open ham-menubar for details."
                )
            }

            return candidate
        }
    }
}
