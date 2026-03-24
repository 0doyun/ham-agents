import SwiftUI
import HamAppServices
import HamCore
import HamNotifications
import AppKit

@main
struct HamMenuBarApp: App {
    @StateObject private var viewModel = HamMenuBarApp.makeViewModel()

    var body: some Scene {
        MenuBarExtra {
            MenuBarContentView(viewModel: viewModel)
                .frame(minWidth: 320, minHeight: 220)
                .task {
                    await viewModel.refresh()
                }
        } label: {
            Text(viewModel.statusLine)
        }
        .menuBarExtraStyle(.window)
    }

    private static func makeViewModel() -> MenuBarViewModel {
        let client: HamDaemonClientProtocol
        if let transport = try? UnixSocketDaemonTransport() {
            client = HamDaemonClient(transport: transport)
        } else {
            client = PreviewDaemonClient()
        }
        let notificationSink = UserNotificationSink()
        let projectOpener = WorkspaceProjectOpener()
        let sessionOpener = ItermSessionOpener(projectOpener: projectOpener)
        let viewModel = MenuBarViewModel(
            client: client,
            notificationSink: notificationSink,
            notificationPermissionController: notificationSink,
            projectOpener: projectOpener,
            sessionOpener: sessionOpener,
            quickMessageSender: ItermQuickMessageSender(
                sessionOpener: sessionOpener,
                projectOpener: projectOpener
            )
        )
        viewModel.start()
        return viewModel
    }
}

private struct MenuBarContentView: View {
    @ObservedObject var viewModel: MenuBarViewModel
    @State private var selectedAgentID: Agent.ID?
    @State private var quickMessage = ""

    var body: some View {
        HStack(alignment: .top, spacing: 14) {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Text("Ham Office")
                        .font(.headline)
                    Spacer()
                    if viewModel.isRefreshing {
                        ProgressView()
                            .controlSize(.small)
                    }
                    Button("Refresh") {
                        Task { await viewModel.refresh() }
                    }
                }

                if let summary = viewModel.summary {
                    HStack {
                        SummaryBadge(title: "Total", value: summary.totalAgents)
                        SummaryBadge(title: "Run", value: summary.runningAgents)
                        SummaryBadge(title: "Wait", value: summary.waitingAgents)
                        SummaryBadge(title: "Done", value: summary.doneAgents)
                    }
                }

                if let errorMessage = viewModel.errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                if let presentation = viewModel.latestEventPresentation,
                   let summary = viewModel.latestEventSummary {
                    LatestEventBanner(presentation: presentation, summary: summary)
                }

                let recentActivityChips = viewModel.recentEventSummaryChips(forAgentID: nil)
                if !recentActivityChips.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Recent Activity")
                            .font(.caption.weight(.semibold))
                        EventSummaryChipsView(chips: recentActivityChips)
                    }
                }

                NotificationPermissionRow(
                    status: viewModel.notificationPermissionStatus,
                    requestPermission: {
                        Task { await viewModel.requestNotificationPermission() }
                    }
                )

                NotificationSettingsSection(
                    settings: viewModel.settings.notifications,
                    updateDone: { value in
                        Task { await viewModel.updateNotificationSetting(done: value) }
                    },
                    updateError: { value in
                        Task { await viewModel.updateNotificationSetting(error: value) }
                    },
                    updateWaiting: { value in
                        Task { await viewModel.updateNotificationSetting(waitingInput: value) }
                    },
                    updateQuietHours: { value in
                        Task { await viewModel.updateNotificationSetting(quietHoursEnabled: value) }
                    },
                    updateQuietStartHour: { value in
                        Task { await viewModel.updateNotificationSetting(quietHoursStartHour: value) }
                    },
                    updateQuietEndHour: { value in
                        Task { await viewModel.updateNotificationSetting(quietHoursEndHour: value) }
                    },
                    updatePreviewText: { value in
                        Task { await viewModel.updateNotificationSetting(previewText: value) }
                    }
                )

                AppearanceSettingsSection(
                    settings: viewModel.settings.appearance,
                    updateTheme: { value in
                        Task { await viewModel.updateAppearanceSetting(theme: value) }
                    }
                )

                IntegrationSettingsSection(
                    settings: viewModel.settings.integrations,
                    updateItermEnabled: { value in
                        Task { await viewModel.updateIntegrationSetting(itermEnabled: value) }
                    }
                )

                if viewModel.settings.integrations.itermEnabled && !viewModel.attachableSessions.isEmpty {
                    AttachableSessionsSection(sessions: viewModel.attachableSessions)
                }

                Text("Agents")
                    .font(.subheadline.weight(.semibold))

                if viewModel.agents.isEmpty {
                    Text("No tracked agents")
                        .foregroundStyle(.secondary)
                } else {
                    List(selection: $selectedAgentID) {
                        ForEach(viewModel.agents) { agent in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(agent.displayName)
                                    .font(.body.weight(.medium))
                                Text("\(viewModel.statusDisplayText(for: agent)) · \(agent.mode.rawValue) · \(viewModel.confidenceLevelText(for: agent)) \(viewModel.confidenceText(for: agent))")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .lineLimit(1)
                            }
                            .tag(agent.id)
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .frame(minWidth: 190)

            Divider()

            AgentDetailView(
                agent: viewModel.agent(withID: selectedAgentID),
                recentEvents: viewModel.recentEvents(forAgentID: selectedAgentID),
                recentEventSummaryChips: viewModel.recentEventSummaryChips(forAgentID: selectedAgentID),
                notificationsMuted: viewModel.isNotificationsMuted(forAgentID: selectedAgentID),
                quickMessageFeedback: viewModel.quickMessageFeedback,
                confidenceText: viewModel.confidenceSummaryText(for: viewModel.agent(withID: selectedAgentID)),
                roleDraft: Binding(
                    get: { viewModel.roleDraft },
                    set: { viewModel.roleDraft = $0 }
                ),
                quickMessage: $quickMessage,
                openProject: {
                    viewModel.openProject(forAgentID: selectedAgentID)
                },
                openSession: {
                    viewModel.openSession(forAgentID: selectedAgentID)
                },
                canOpenSession: viewModel.settings.integrations.itermEnabled,
                toggleNotifications: {
                    viewModel.toggleNotificationPause(forAgentID: selectedAgentID)
                },
                saveRole: {
                    await viewModel.saveRole(forAgentID: selectedAgentID)
                },
                stopTracking: {
                    await viewModel.stopTracking(forAgentID: selectedAgentID)
                    selectedAgentID = viewModel.agents.first?.id
                },
                sendQuickMessage: {
                    viewModel.sendQuickMessage(quickMessage, forAgentID: selectedAgentID)
                    quickMessage = ""
                }
            )
            .frame(minWidth: 140, maxWidth: .infinity, alignment: .topLeading)
        }
        .padding(14)
        .onAppear {
            if selectedAgentID == nil {
                selectedAgentID = viewModel.agents.first?.id
            }
            viewModel.setRoleDraft(from: selectedAgentID)
        }
        .onChange(of: viewModel.agents.map(\.id)) { ids in
            if selectedAgentID == nil || !ids.contains(selectedAgentID ?? "") {
                selectedAgentID = ids.first
            }
            viewModel.setRoleDraft(from: selectedAgentID)
        }
    }
}

private struct NotificationSettingsSection: View {
    let settings: DaemonNotificationSettingsPayload
    let updateDone: (Bool) -> Void
    let updateError: (Bool) -> Void
    let updateWaiting: (Bool) -> Void
    let updateQuietHours: (Bool) -> Void
    let updateQuietStartHour: (Int) -> Void
    let updateQuietEndHour: (Int) -> Void
    let updatePreviewText: (Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Notification Settings")
                .font(.caption.weight(.semibold))

            Toggle("Done", isOn: Binding(get: { settings.done }, set: updateDone))
            Toggle("Error", isOn: Binding(get: { settings.error }, set: updateError))
            Toggle("Waiting Input", isOn: Binding(get: { settings.waitingInput }, set: updateWaiting))
            Toggle("Quiet Hours", isOn: Binding(get: { settings.quietHoursEnabled }, set: updateQuietHours))
            if settings.quietHoursEnabled {
                HStack {
                    Text("Start")
                    Spacer()
                    Text(hourLabel(settings.quietHoursStartHour))
                        .foregroundStyle(.secondary)
                    Stepper(
                        "",
                        value: Binding(
                            get: { settings.quietHoursStartHour },
                            set: updateQuietStartHour
                        ),
                        in: 0...23
                    )
                    .labelsHidden()
                }
                HStack {
                    Text("End")
                    Spacer()
                    Text(hourLabel(settings.quietHoursEndHour))
                        .foregroundStyle(.secondary)
                    Stepper(
                        "",
                        value: Binding(
                            get: { settings.quietHoursEndHour },
                            set: updateQuietEndHour
                        ),
                        in: 0...23
                    )
                    .labelsHidden()
                }
                Text("Current window \(hourLabel(settings.quietHoursStartHour)) → \(hourLabel(settings.quietHoursEndHour))")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            Toggle("Preview Text", isOn: Binding(get: { settings.previewText }, set: updatePreviewText))
        }
        .toggleStyle(.checkbox)
    }

    private func hourLabel(_ hour: Int) -> String {
        String(format: "%02d:00", hour)
    }
}

private struct NotificationPermissionRow: View {
    let status: NotificationPermissionStatus
    let requestPermission: () -> Void

    var body: some View {
        HStack {
            Text("Notifications")
                .font(.caption.weight(.semibold))
            Spacer()
            Text(statusLabel)
                .font(.caption)
                .foregroundStyle(.secondary)
            if status != .authorized {
                Button("Enable") {
                    requestPermission()
                }
                .buttonStyle(.borderless)
            }
        }
    }

    private var statusLabel: String {
        switch status {
        case .authorized:
            "Enabled"
        case .denied:
            "Denied"
        case .notDetermined:
            "Not requested"
        }
    }
}

private struct AttachableSessionsSection: View {
    let sessions: [DaemonAttachableSessionPayload]

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Attachable iTerm Sessions")
                .font(.caption.weight(.semibold))

            ForEach(sessions.prefix(3)) { session in
                HStack(spacing: 6) {
                    Text(session.title)
                        .lineLimit(1)
                    if session.isActive {
                        Text("Current")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
                .font(.caption)
            }
        }
    }
}

private struct IntegrationSettingsSection: View {
    let settings: DaemonIntegrationSettingsPayload
    let updateItermEnabled: (Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Integrations")
                .font(.caption.weight(.semibold))

            Toggle(
                "iTerm2 Access",
                isOn: Binding(
                    get: { settings.itermEnabled },
                    set: updateItermEnabled
                )
            )
        }
        .toggleStyle(.checkbox)
    }
}

private struct LatestEventBanner: View {
    let presentation: AgentEventPresentation
    let summary: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(presentation.label)
                .font(.caption.weight(.semibold))
            Text(summary)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(2)
        }
        .padding(8)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(eventBadgeBackground(for: presentation.emphasis))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}

private struct AppearanceSettingsSection: View {
    let settings: DaemonAppearanceSettingsPayload
    let updateTheme: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Appearance")
                .font(.caption.weight(.semibold))

            Picker(
                "Theme",
                selection: Binding(
                    get: { settings.theme },
                    set: updateTheme
                )
            ) {
                Text("Auto").tag("auto")
                Text("Day").tag("day")
                Text("Night").tag("night")
            }
            .labelsHidden()
        }
    }
}

private struct AgentDetailView: View {
    let agent: Agent?
    let recentEvents: [AgentEventPayload]
    let recentEventSummaryChips: [AgentEventSummaryChip]
    let notificationsMuted: Bool
    let quickMessageFeedback: String?
    let confidenceText: String
    @Binding var roleDraft: String
    @Binding var quickMessage: String
    let openProject: () -> Void
    let openSession: () -> Void
    let canOpenSession: Bool
    let toggleNotifications: () -> Void
    let saveRole: () async -> Void
    let stopTracking: () async -> Void
    let sendQuickMessage: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Details")
                .font(.subheadline.weight(.semibold))

            if let agent {
                Text(agent.displayName)
                    .font(.headline)
                Text("\(agent.provider) · \(agent.mode.rawValue)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                if let sessionTitle = agent.sessionTitle, !sessionTitle.isEmpty {
                    Text(sessionTitle)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                if agent.mode == .attached {
                    Text(agent.sessionIsActive ? "Current iTerm session" : "Background iTerm session")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    if let sessionTTY = agent.sessionTTY, !sessionTTY.isEmpty {
                        Text("tty \(sessionTTY)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let sessionWorkingDirectory = agent.sessionWorkingDirectory, !sessionWorkingDirectory.isEmpty {
                        Text(sessionWorkingDirectory)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }
                    if let sessionActivity = agent.sessionActivity, !sessionActivity.isEmpty {
                        Text("activity \(sessionActivity)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let sessionProcessID = agent.sessionProcessID {
                        Text("pid \(sessionProcessID)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let sessionCommand = agent.sessionCommand, !sessionCommand.isEmpty {
                        Text(sessionCommand)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }
                }
                Text(agent.projectPath)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
                Text(confidenceText)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                if let statusReason = agent.statusReason, !statusReason.isEmpty {
                    Text("Reason: \(statusReason)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                }

                Text("Role")
                    .font(.caption.weight(.semibold))
                HStack {
                    TextField("Role…", text: $roleDraft)
                        .textFieldStyle(.roundedBorder)
                    Button("Save") {
                        Task { await saveRole() }
                    }
                    .buttonStyle(.bordered)
                }

                if let summary = agent.lastUserVisibleSummary {
                    Text(summary)
                        .font(.caption)
                }

                Button("Open Project Folder") {
                    openProject()
                }
                .buttonStyle(.bordered)

                Button("Open in iTerm") {
                    openSession()
                }
                .buttonStyle(.borderedProminent)
                .disabled(!canOpenSession)

                Button("Stop Tracking") {
                    Task { await stopTracking() }
                }
                .buttonStyle(.bordered)
                .tint(.red)

                Button(notificationsMuted ? "Resume Notifications" : "Pause Notifications") {
                    toggleNotifications()
                }
                .buttonStyle(.bordered)

                Text("Quick Message")
                    .font(.caption.weight(.semibold))
                    .padding(.top, 4)
                TextField("Draft message…", text: $quickMessage, axis: .vertical)
                    .textFieldStyle(.roundedBorder)
                    .lineLimit(2 ... 4)
                Button("Send Message") {
                    sendQuickMessage()
                }
                .buttonStyle(.borderedProminent)
                .disabled(quickMessage.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                if let quickMessageFeedback {
                    Text(quickMessageFeedback)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Text("Recent Events")
                    .font(.caption.weight(.semibold))
                    .padding(.top, 4)

                if !recentEventSummaryChips.isEmpty {
                    EventSummaryChipsView(chips: recentEventSummaryChips)
                }

                if recentEvents.isEmpty {
                    Text("No recent events for this agent.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(recentEvents.prefix(3)) { event in
                        let presentation = AgentEventPresenter.present(event)
                        VStack(alignment: .leading, spacing: 2) {
                            HStack(spacing: 6) {
                                Text(presentation.label)
                                    .font(.caption2.weight(.semibold))
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 2)
                                    .background(eventBadgeBackground(for: presentation.emphasis))
                                    .clipShape(Capsule())
                                if presentation.showsTechnicalType {
                                    Text(event.type)
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                } else {
                                    Text(event.occurredAt.formatted(.relative(presentation: .named)))
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                            }
                            Text(event.summary)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                        .padding(.vertical, 4)
                    }
                }
            } else {
                Text("Select an agent to inspect status, project, and recent events.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Spacer()
        }
    }
}

private struct EventSummaryChipsView: View {
    let chips: [AgentEventSummaryChip]

    var body: some View {
        FlexibleChipRow(items: chips) { chip in
            HStack(spacing: 4) {
                Text(chip.label)
                Text("\(chip.count)")
                    .foregroundStyle(.secondary)
            }
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background(eventBadgeBackground(for: chip.emphasis))
            .clipShape(Capsule())
        }
    }
}

private struct FlexibleChipRow<Item, Content: View>: View {
    let items: [Item]
    let content: (Item) -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            ForEach(Array(items.enumerated()), id: \.offset) { _, item in
                content(item)
            }
        }
    }
}

private func eventBadgeBackground(for emphasis: AgentEventEmphasis) -> Color {
    switch emphasis {
    case .positive:
        return .green.opacity(0.18)
    case .warning:
        return .orange.opacity(0.18)
    case .info:
        return .blue.opacity(0.18)
    case .neutral:
        return .gray.opacity(0.18)
    }
}

private struct SummaryBadge: View {
    let title: String
    let value: Int

    var body: some View {
        VStack(spacing: 4) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(String(value))
                .font(.headline.monospacedDigit())
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .background(Color.gray.opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}

private struct PreviewDaemonClient: HamDaemonClientProtocol {
    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        DaemonRuntimeSnapshotPayload(
            agents: [
                Agent(
                    id: "preview-1",
                    displayName: "preview-reviewer",
                    provider: "claude",
                    host: "localhost",
                    mode: .managed,
                    projectPath: "/tmp/demo",
                    status: .thinking,
                    statusConfidence: 1,
                    lastEventAt: .now
                )
            ],
            generatedAt: .now
        )
    }

    func fetchAgents() async throws -> [Agent] {
        (try await fetchSnapshot()).agents
    }

    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] {
        [
            DaemonAttachableSessionPayload(
                id: "preview-iterm-1",
                title: "Claude Review",
                sessionRef: "iterm2://session/preview-iterm-1",
                isActive: true
            ),
            DaemonAttachableSessionPayload(
                id: "preview-iterm-2",
                title: "Shell",
                sessionRef: "iterm2://session/preview-iterm-2",
                isActive: false
            ),
        ]
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        [
            AgentEventPayload(
                id: "preview-event-1",
                agentID: "preview-1",
                type: "agent.registered",
                summary: "Preview mode active.",
                occurredAt: .now
            )
        ]
    }

    func followEvents(afterEventID: String, limit: Int, waitMilliseconds: Int) async throws -> [AgentEventPayload] {
        _ = afterEventID
        _ = limit
        _ = waitMilliseconds
        return []
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
        let agents = try await fetchAgents()
        var agent = agents.first { $0.id == agentID } ?? agents.first!
        agent.notificationPolicy = policy
        return agent
    }

    func updateRole(agentID: String, role: String) async throws -> Agent {
        let agents = try await fetchAgents()
        var agent = agents.first { $0.id == agentID } ?? agents.first!
        agent.role = role
        return agent
    }

    func removeAgent(agentID: String) async throws {
        _ = agentID
    }
}

private struct WorkspaceProjectOpener: ProjectOpening {
    func openProject(at path: String) {
        NSWorkspace.shared.open(URL(fileURLWithPath: path))
    }
}

private struct ItermSessionOpener: SessionOpening {
    let projectOpener: ProjectOpening
    private let planner = SessionTargetPlanner()

    func openSession(for agent: Agent) {
        let workspace = NSWorkspace.shared
        switch planner.target(for: agent) {
        case .itermSession(let sessionID, let url):
            if focusSession(id: sessionID) {
                return
            }
            workspace.open(url)
        case .externalURL(let url):
            workspace.open(url)
        case .workspace(let path):
            let projectURL = URL(fileURLWithPath: path)

            if let appURL = workspace.urlForApplication(withBundleIdentifier: "com.googlecode.iterm2") {
                let configuration = NSWorkspace.OpenConfiguration()
                workspace.open([projectURL], withApplicationAt: appURL, configuration: configuration) { _, _ in }
                return
            }

            projectOpener.openProject(at: path)
        }
    }

    private func focusSession(id sessionID: String) -> Bool {
        let source = """
        tell application "iTerm"
            activate
            repeat with aWindow in windows
                repeat with aTab in tabs of aWindow
                    repeat with aSession in sessions of aTab
                        if id of aSession is "\(appleScriptEscaped(sessionID))" then
                            select aSession
                            return
                        end if
                    end repeat
                end repeat
            end repeat
        end tell
        """

        return executeAppleScript(source)
    }
}

private struct ItermQuickMessageSender: QuickMessageSending {
    let sessionOpener: SessionOpening
    let projectOpener: ProjectOpening
    private let planner = QuickMessagePlanner()

    func send(message: String, to agent: Agent) -> QuickMessageResult {
        switch planner.plan(message: message, for: agent, supportsTerminalAutomation: true) {
        case .terminalWrite(let target, let message):
            if tryTerminalWrite(message: message, target: target) {
                return .delivered("Sent to iTerm.")
            }
        case .clipboardHandoff(let message):
            copyToClipboard(message)
            sessionOpener.openSession(for: agent)
            return .handoff("Copied to clipboard and opened the session.")
        }

        copyToClipboard(message)
        projectOpener.openProject(at: agent.projectPath)
        return .handoff("Copied to clipboard and opened the project folder.")
    }

    private func tryTerminalWrite(message: String, target: SessionTarget) -> Bool {
        let workspace = NSWorkspace.shared

        switch target {
        case .itermSession(let sessionID, let url):
            workspace.open(url)
            return executeAppleScript(targetedWriteSource(message: message, sessionID: sessionID))
        case .externalURL(let url):
            workspace.open(url)
        case .workspace(let path):
            guard let appURL = workspace.urlForApplication(withBundleIdentifier: "com.googlecode.iterm2") else {
                return false
            }
            let configuration = NSWorkspace.OpenConfiguration()
            workspace.open([URL(fileURLWithPath: path)], withApplicationAt: appURL, configuration: configuration) { _, _ in }
        }

        return executeAppleScript(defaultWriteSource(message: message))
    }

    private func copyToClipboard(_ message: String) {
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()
        pasteboard.setString(message, forType: .string)
    }

    private func appleScriptEscaped(_ message: String) -> String {
        message
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "\"", with: "\\\"")
    }

    private func defaultWriteSource(message: String) -> String {
        """
        tell application "iTerm"
            activate
            tell current window
                tell current session
                    write text "\(appleScriptEscaped(message))"
                end tell
            end tell
        end tell
        """
    }

    private func targetedWriteSource(message: String, sessionID: String) -> String {
        """
        tell application "iTerm"
            activate
            repeat with aWindow in windows
                repeat with aTab in tabs of aWindow
                    repeat with aSession in sessions of aTab
                        if id of aSession is "\(appleScriptEscaped(sessionID))" then
                            tell aSession to write text "\(appleScriptEscaped(message))"
                            return
                        end if
                    end repeat
                end repeat
            end repeat
            tell current window
                tell current session
                    write text "\(appleScriptEscaped(message))"
                end tell
            end tell
        end tell
        """
    }
}

@discardableResult
private func executeAppleScript(_ source: String) -> Bool {
    guard let script = NSAppleScript(source: source) else {
        return false
    }

    var error: NSDictionary?
    script.executeAndReturnError(&error)
    return error == nil
}

private func appleScriptEscaped(_ text: String) -> String {
    text
        .replacingOccurrences(of: "\\", with: "\\\\")
        .replacingOccurrences(of: "\"", with: "\\\"")
}
