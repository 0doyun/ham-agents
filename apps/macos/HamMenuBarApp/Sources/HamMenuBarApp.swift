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

                NotificationPermissionRow(
                    status: viewModel.notificationPermissionStatus,
                    requestPermission: {
                        Task { await viewModel.requestNotificationPermission() }
                    }
                )

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
                                Text("\(agent.status.rawValue) · \(agent.projectPath)")
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
                notificationsMuted: viewModel.isNotificationsMuted(forAgentID: selectedAgentID),
                quickMessageFeedback: viewModel.quickMessageFeedback,
                quickMessage: $quickMessage,
                openProject: {
                    viewModel.openProject(forAgentID: selectedAgentID)
                },
                openSession: {
                    viewModel.openSession(forAgentID: selectedAgentID)
                },
                toggleNotifications: {
                    viewModel.toggleNotificationPause(forAgentID: selectedAgentID)
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
        }
        .onChange(of: viewModel.agents.map(\.id)) { ids in
            if selectedAgentID == nil || !ids.contains(selectedAgentID ?? "") {
                selectedAgentID = ids.first
            }
        }
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

private struct AgentDetailView: View {
    let agent: Agent?
    let recentEvents: [AgentEventPayload]
    let notificationsMuted: Bool
    let quickMessageFeedback: String?
    @Binding var quickMessage: String
    let openProject: () -> Void
    let openSession: () -> Void
    let toggleNotifications: () -> Void
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
                Text(agent.projectPath)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)

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

                if recentEvents.isEmpty {
                    Text("No recent events for this agent.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(recentEvents.prefix(3)) { event in
                        VStack(alignment: .leading, spacing: 2) {
                            Text(event.type)
                                .font(.caption.weight(.medium))
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

    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent {
        let agents = try await fetchAgents()
        var agent = agents.first { $0.id == agentID } ?? agents.first!
        agent.notificationPolicy = policy
        return agent
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
        case .externalURL(let url):
            workspace.open(url)
        case .workspace(let path):
            guard let appURL = workspace.urlForApplication(withBundleIdentifier: "com.googlecode.iterm2") else {
                return false
            }
            let configuration = NSWorkspace.OpenConfiguration()
            workspace.open([URL(fileURLWithPath: path)], withApplicationAt: appURL, configuration: configuration) { _, _ in }
        }

        let source = """
        tell application "iTerm"
            activate
            tell current window
                tell current session
                    write text "\(appleScriptEscaped(message))"
                end tell
            end tell
        end tell
        """

        guard let script = NSAppleScript(source: source) else {
            return false
        }

        var error: NSDictionary?
        script.executeAndReturnError(&error)
        return error == nil
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
}
