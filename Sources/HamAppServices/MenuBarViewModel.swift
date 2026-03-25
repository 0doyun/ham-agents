import Combine
import Foundation
import HamCore
import HamNotifications

@MainActor
public final class MenuBarViewModel: ObservableObject {
    @Published public private(set) var summary: HamMenuBarSummary?
    @Published public private(set) var agents: [Agent] = []
    @Published public private(set) var attachableSessions: [DaemonAttachableSessionPayload] = []
    @Published public private(set) var teams: [DaemonTeamPayload] = []
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
            silence: false,
            quietHoursEnabled: false,
            quietHoursStartHour: 22,
            quietHoursEndHour: 8,
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
    private let eventFollowWaitMilliseconds: Int
    private let sleep: @Sendable (UInt64) async throws -> Void
    private let now: @Sendable () -> Date
    private let calendar: Calendar
    private var hasStarted = false
    private var refreshTask: Task<Void, Never>?
    private var eventFollowTask: Task<Void, Never>?

    public init(
        client: HamDaemonClientProtocol,
        notificationEngine: StatusChangeNotificationEngine = StatusChangeNotificationEngine(),
        notificationSink: NotificationSink = NoopNotificationSink(),
        notificationPermissionController: NotificationPermissionControlling = NoopNotificationPermissionController(),
        projectOpener: ProjectOpening = NoopProjectOpener(),
        sessionOpener: SessionOpening = NoopSessionOpener(),
        quickMessageSender: QuickMessageSending = NoopQuickMessageSender(),
        pollIntervalNanoseconds: UInt64 = 15_000_000_000,
        eventFollowWaitMilliseconds: Int = 15_000,
        now: @escaping @Sendable () -> Date = { Date() },
        calendar: Calendar = .autoupdatingCurrent,
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
        self.eventFollowWaitMilliseconds = eventFollowWaitMilliseconds
        self.now = now
        self.calendar = calendar
        self.sleep = sleep
    }

    public var statusLine: String {
        guard let summary else {
            return errorMessage == nil ? "ham idle" : "ham offline"
        }
        let base = "ham \(summary.runningAgents)▶ \(summary.waitingAgents)? \(summary.doneAgents)✓"
        guard let indicator = latestEventIndicator else { return base }
        return "\(indicator) \(base)"
    }

    public var latestEventPresentation: AgentEventPresentation? {
        guard let event = summary?.recentEvents.last else { return nil }
        return AgentEventPresenter.present(event)
    }

    public var latestEventSummary: String? {
        guard let event = summary?.recentEvents.last else { return nil }
        return AgentEventPresenter.displaySummary(for: event)
    }

    public func agent(withID id: Agent.ID?) -> Agent? {
        guard let id else { return agents.first }
        return agents.first(where: { $0.id == id }) ?? agents.first
    }

    public func setRoleDraft(from agentID: Agent.ID?) {
        roleDraft = agent(withID: agentID)?.role ?? ""
    }

    public func recentEvents(forAgentID id: Agent.ID?) -> [AgentEventPayload] {
        let events = AgentEventPresenter.ordered(summary?.recentEvents ?? [])
        guard let id else { return events }
        return events.filter { $0.agentID == id }
    }

    public func recentEventSummaryChips(forAgentID id: Agent.ID?) -> [AgentEventSummaryChip] {
        AgentEventPresenter.summarize(recentEvents(forAgentID: id))
    }

    public func recentEventSeverityChips(forAgentID id: Agent.ID?) -> [AgentEventSummaryChip] {
        AgentEventPresenter.summarizeBySeverity(recentEvents(forAgentID: id))
    }

    public var topSummaryAttentionBreakdownChips: [AgentEventSummaryChip] {
        guard let summary, summary.attentionAgents > 0 else { return [] }

        let breakdown = summary.attentionBreakdown
        return [
            breakdown.error > 0 ? AgentEventSummaryChip(label: "Errors", emphasis: .warning, count: breakdown.error) : nil,
            breakdown.waitingInput > 0 ? AgentEventSummaryChip(label: "Needs Input", emphasis: .info, count: breakdown.waitingInput) : nil,
            breakdown.disconnected > 0 ? AgentEventSummaryChip(label: "Disconnected", emphasis: .neutral, count: breakdown.disconnected) : nil,
        ].compactMap { $0 }
    }

    public var attentionAgents: [Agent] {
        let filtered = agents.filter { attentionPriority(for: $0) != nil }
        guard let summary else {
            return sortAttentionAgents(filtered)
        }

        let orderIndex = Dictionary(uniqueKeysWithValues: summary.attentionOrder.enumerated().map { ($1, $0) })
        return filtered.sorted { lhs, rhs in
            let lhsOrder = orderIndex[lhs.id]
            let rhsOrder = orderIndex[rhs.id]
            switch (lhsOrder, rhsOrder) {
            case let (.some(left), .some(right)):
                if left != right { return left < right }
            case (.some, .none):
                return true
            case (.none, .some):
                return false
            case (.none, .none):
                break
            }
            return compareAttentionAgents(lhs, rhs)
        }
    }

    public var nonAttentionAgents: [Agent] {
        let attentionIDs = Set(attentionAgents.map(\.id))
        return agents.filter { !attentionIDs.contains($0.id) }
    }

    public func attentionSubtitle(for agent: Agent) -> String {
        if let subtitle = summary?.attentionSubtitles[agent.id], !subtitle.isEmpty {
            return subtitle
        }

        let status = statusDisplayText(for: agent)
        let confidence = confidenceLevelText(for: agent).lowercased()
        if let reason = agent.statusReason, !reason.isEmpty {
            return "\(status) · \(confidence) confidence · \(reason)"
        }
        return "\(status) · \(confidence) confidence"
    }

    public func confidenceText(for agent: Agent?) -> String {
        guard let agent else { return "—" }
        return "\(Int((agent.statusConfidence * 100).rounded()))%"
    }

    public func confidenceLevelText(for agent: Agent?) -> String {
        guard let agent else { return "Unknown" }
        switch agent.statusConfidence {
        case 0.8...:
            return "High"
        case 0.5..<0.8:
            return "Medium"
        default:
            return "Low"
        }
    }

    public func statusDisplayText(for agent: Agent?) -> String {
        guard let agent else { return "unknown" }
        let label = agent.status.humanizedLabel
        if agent.statusConfidence < 0.5 {
            return "likely \(label)"
        }
        return label
    }

    public func confidenceSummaryText(for agent: Agent?) -> String {
        guard let agent else { return "unknown confidence" }
        return "\(confidenceLevelText(for: agent).lowercased()) confidence (\(confidenceText(for: agent)))"
    }

    public func openProject(forAgentID id: Agent.ID?) {
        guard let agent = agent(withID: id) else { return }
        projectOpener.openProject(at: agent.projectPath)
    }

    public func openSession(forAgentID id: Agent.ID?) {
        guard let agent = agent(withID: id) else { return }
        guard settings.integrations.itermEnabled else {
            errorMessage = "Enable iTerm integration in Settings to open sessions."
            return
        }
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
        silence: Bool? = nil,
        quietHoursEnabled: Bool? = nil,
        quietHoursStartHour: Int? = nil,
        quietHoursEndHour: Int? = nil,
        previewText: Bool? = nil
    ) async {
        var updated = settings
        if let done { updated.notifications.done = done }
        if let error { updated.notifications.error = error }
        if let waitingInput { updated.notifications.waitingInput = waitingInput }
        if let silence { updated.notifications.silence = silence }
        if let quietHoursEnabled { updated.notifications.quietHoursEnabled = quietHoursEnabled }
        if let quietHoursStartHour { updated.notifications.quietHoursStartHour = quietHoursStartHour }
        if let quietHoursEndHour { updated.notifications.quietHoursEndHour = quietHoursEndHour }
        if let previewText { updated.notifications.previewText = previewText }

        do {
            settings = try await client.updateSettings(updated)
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    public func updateAppearanceSetting(theme: String) async {
        var updated = settings
        updated.appearance.theme = theme

        do {
            settings = try await client.updateSettings(updated)
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    public func updateIntegrationSetting(itermEnabled: Bool) async {
        var updated = settings
        updated.integrations.itermEnabled = itermEnabled

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

        eventFollowTask = Task { [weak self] in
            guard let self else { return }
            while !Task.isCancelled {
                await self.followLatestEvents(waitMilliseconds: self.eventFollowWaitMilliseconds)
            }
        }
    }

    public func stop() {
        refreshTask?.cancel()
        refreshTask = nil
        eventFollowTask?.cancel()
        eventFollowTask = nil
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
            let loadedAttachableSessionsValue = (try? await client.fetchAttachableSessions()) ?? []
            let loadedTeamsValue = (try? await client.fetchTeams()) ?? []
            applyRefreshedState(
                summary: summaryValue,
                agents: loadedAgentsValue,
                previousAgents: previousAgents,
                settings: loadedSettingsValue,
                teams: loadedTeamsValue
            )
            attachableSessions = loadedAttachableSessionsValue
            teams = loadedTeamsValue
            settings = loadedSettingsValue
            notificationPermissionStatus = await permissionStatus
            if roleDraft.isEmpty, let firstAgent = agents.first {
                roleDraft = firstAgent.role ?? ""
            }
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    public func followLatestEvents(eventLimit: Int = 5, waitMilliseconds: Int) async {
        let afterEventID = summary?.recentEvents.last?.id ?? ""

        do {
            let followedEvents = try await client.followEvents(
                afterEventID: afterEventID,
                limit: eventLimit,
                waitMilliseconds: waitMilliseconds
            )
            guard !followedEvents.isEmpty else { return }

            let previousAgents = agents
            async let loadedAgents = client.fetchAgents()

            let loadedAgentsValue = try await loadedAgents
            let mergedEvents = mergedRecentEvents(current: summary?.recentEvents ?? [], followed: followedEvents, limit: eventLimit)
            let summaryValue = makeSummary(agents: loadedAgentsValue, recentEvents: mergedEvents, generatedAt: now())

            applyRefreshedState(
                summary: summaryValue,
                agents: loadedAgentsValue,
                previousAgents: previousAgents,
                settings: settings,
                teams: teams
            )
            errorMessage = nil
        } catch {
            return
        }
    }

    deinit {
        refreshTask?.cancel()
        eventFollowTask?.cancel()
    }

    private func filteredNotificationCandidates(
        _ candidates: [NotificationCandidate],
        settings: DaemonSettingsPayload
    ) -> [NotificationCandidate] {
        if isQuietHoursActive(settings.notifications, at: now()) {
            return []
        }

        return candidates.compactMap { candidate -> NotificationCandidate? in
            switch candidate.event {
            case .done:
                guard settings.notifications.done else { return nil }
            case .error:
                guard settings.notifications.error else { return nil }
            case .waitingInput:
                guard settings.notifications.waitingInput else { return nil }
            case .silence:
                guard settings.notifications.silence else { return nil }
            case .teamDigest:
                guard settings.notifications.error || settings.notifications.waitingInput else { return nil }
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

    private func applyRefreshedState(
        summary: HamMenuBarSummary,
        agents: [Agent],
        previousAgents: [Agent],
        settings: DaemonSettingsPayload,
        teams: [DaemonTeamPayload]
    ) {
        let candidates = filteredNotificationCandidates(
            notificationEngine.candidates(
                previous: previousAgents,
                current: agents,
                previousObservedAt: self.summary?.generatedAt,
                currentObservedAt: summary.generatedAt
            ),
            settings: settings
        ) + filteredNotificationCandidates(
            teamDigestCandidates(previousAgents: previousAgents, currentAgents: agents, teams: teams),
            settings: settings
        )

        self.summary = summary
        self.agents = agents
        self.teams = teams

        for candidate in candidates {
            notificationSink.send(candidate)
        }
    }

    private func teamDigestCandidates(
        previousAgents: [Agent],
        currentAgents: [Agent],
        teams: [DaemonTeamPayload]
    ) -> [NotificationCandidate] {
        let previousByID = Dictionary(uniqueKeysWithValues: previousAgents.map { ($0.id, $0) })

        return teams.compactMap { team in
            let currentMembers = currentAgents.filter { team.memberAgentIDs.contains($0.id) }
            guard !currentMembers.isEmpty else { return nil }

            let currentAttention = currentMembers.filter { attentionPriority(for: $0) != nil }
            guard !currentAttention.isEmpty else { return nil }

            let previousAttentionCount = team.memberAgentIDs.reduce(into: 0) { result, memberID in
                if let previousAgent = previousByID[memberID], attentionPriority(for: previousAgent) != nil {
                    result += 1
                }
            }
            guard previousAttentionCount == 0 else { return nil }

            let errorCount = currentAttention.filter { $0.status == .error }.count
            let needsInputCount = currentAttention.filter { $0.status == .waitingInput }.count
            let disconnectedCount = currentAttention.filter { $0.status == .disconnected }.count

            var parts: [String] = []
            if errorCount > 0 { parts.append("\(errorCount) error") }
            if needsInputCount > 0 { parts.append("\(needsInputCount) needs input") }
            if disconnectedCount > 0 { parts.append("\(disconnectedCount) disconnected") }
            let body = parts.isEmpty ? "Team requires attention." : parts.joined(separator: ", ")

            return NotificationCandidate(
                event: .teamDigest(team.displayName),
                title: "\(team.displayName) needs attention",
                body: body
            )
        }
    }

    private func makeSummary(snapshot: DaemonRuntimeSnapshotPayload, recentEvents: [AgentEventPayload]) -> HamMenuBarSummary {
        HamMenuBarSummary(
            generatedAt: snapshot.generatedAt,
            totalAgents: snapshot.totalCount,
            attentionAgents: snapshot.attentionCount,
            attentionBreakdown: snapshot.attentionBreakdown,
            attentionOrder: snapshot.attentionOrder,
            attentionSubtitles: snapshot.attentionSubtitles,
            runningAgents: snapshot.runningCount,
            waitingAgents: snapshot.waitingCount,
            doneAgents: snapshot.doneCount,
            recentEvents: recentEvents
        )
    }

    private func makeSummary(agents: [Agent], recentEvents: [AgentEventPayload], generatedAt: Date) -> HamMenuBarSummary {
        let totalAgents = agents.count
        let attentionAgents = agents.filter { attentionPriority(for: $0) != nil }.count
        let runningAgents = agents.filter { $0.status.isRunningActivity }.count
        let waitingAgents = agents.filter { $0.status == .waitingInput }.count
        let doneAgents = agents.filter { $0.status == .done }.count

        return HamMenuBarSummary(
            generatedAt: generatedAt,
            totalAgents: totalAgents,
            attentionAgents: attentionAgents,
            attentionBreakdown: .init(
                error: agents.filter { $0.status == .error }.count,
                waitingInput: agents.filter { $0.status == .waitingInput }.count,
                disconnected: agents.filter { $0.status == .disconnected }.count
            ),
            attentionOrder: sortAttentionAgents(agents.filter { attentionPriority(for: $0) != nil }).map(\.id),
            attentionSubtitles: Dictionary(
                uniqueKeysWithValues: agents
                    .filter { attentionPriority(for: $0) != nil }
                    .map { ($0.id, attentionSubtitleFallback(for: $0)) }
            ),
            runningAgents: runningAgents,
            waitingAgents: waitingAgents,
            doneAgents: doneAgents,
            recentEvents: recentEvents
        )
    }

    private func mergedRecentEvents(
        current: [AgentEventPayload],
        followed: [AgentEventPayload],
        limit: Int
    ) -> [AgentEventPayload] {
        var merged = current
        for event in followed where !merged.contains(where: { $0.id == event.id }) {
            merged.append(event)
        }
        if limit > 0 && merged.count > limit {
            return Array(merged.suffix(limit))
        }
        return merged
    }

    private func isQuietHoursActive(_ settings: DaemonNotificationSettingsPayload, at date: Date) -> Bool {
        guard settings.quietHoursEnabled else { return false }
        guard let hour = calendar.dateComponents([.hour], from: date).hour else { return false }

        let startHour = settings.quietHoursStartHour
        let endHour = settings.quietHoursEndHour

        if startHour == endHour {
            return true
        }
        if startHour < endHour {
            return hour >= startHour && hour < endHour
        }
        return hour >= startHour || hour < endHour
    }

    private var latestEventIndicator: String? {
        guard let presentation = latestEventPresentation else { return nil }
        switch presentation.emphasis {
        case .warning:
            return "⚠︎"
        case .positive:
            return "✓"
        case .info:
            return "•"
        case .neutral:
            return nil
        }
    }

    private func attentionPriority(for agent: Agent) -> Int? {
        switch agent.status {
        case .error:
            return 0
        case .waitingInput:
            return 1
        case .disconnected:
            return 2
        default:
            return nil
        }
    }

    private func sortAttentionAgents(_ agents: [Agent]) -> [Agent] {
        agents.sorted(by: compareAttentionAgents)
    }

    private func compareAttentionAgents(_ lhs: Agent, _ rhs: Agent) -> Bool {
        let lhsPriority = attentionPriority(for: lhs) ?? .max
        let rhsPriority = attentionPriority(for: rhs) ?? .max
        if lhsPriority == rhsPriority {
            if lhs.lastEventAt == rhs.lastEventAt {
                if lhs.displayName == rhs.displayName {
                    return lhs.id < rhs.id
                }
                return lhs.displayName < rhs.displayName
            }
            return lhs.lastEventAt > rhs.lastEventAt
        }
        return lhsPriority < rhsPriority
    }

    private func attentionSubtitleFallback(for agent: Agent) -> String {
        let status = statusDisplayText(for: agent)
        let confidence = confidenceLevelText(for: agent).lowercased()
        if let reason = agent.statusReason, !reason.isEmpty {
            return "\(status) · \(confidence) confidence · \(reason)"
        }
        return "\(status) · \(confidence) confidence"
    }


    public var workspaceOptions: [String] {
        Array(Set(agents.map(\.projectPath))).sorted()
    }

    public func teamName(for agent: Agent) -> String? {
        teams.first(where: { $0.memberAgentIDs.contains(agent.id) })?.displayName
    }

    public func filteredAttentionAgents(teamID: String?, workspace: String?) -> [Agent] {
        attentionAgents.filter { agentMatchesFilters($0, teamID: teamID, workspace: workspace) }
    }

    public func filteredNonAttentionAgents(teamID: String?, workspace: String?) -> [Agent] {
        nonAttentionAgents.filter { agentMatchesFilters($0, teamID: teamID, workspace: workspace) }
    }

    private func agentMatchesFilters(_ agent: Agent, teamID: String?, workspace: String?) -> Bool {
        if let workspace, !workspace.isEmpty, agent.projectPath != workspace {
            return false
        }
        if let teamID, !teamID.isEmpty {
            guard let team = teams.first(where: { $0.id == teamID }) else { return false }
            return team.memberAgentIDs.contains(agent.id)
        }
        return true
    }

}
