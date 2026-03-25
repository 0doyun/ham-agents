import AppKit
import Foundation
import HamAppServices
import HamCore

struct PreviewDaemonClient: HamDaemonClientProtocol {
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

    func fetchTeams() async throws -> [DaemonTeamPayload] {
        [DaemonTeamPayload(id: "preview-team-1", displayName: "preview-squad", memberAgentIDs: ["preview-1"])]
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

struct WorkspaceProjectOpener: ProjectOpening {
    func openProject(at path: String) {
        NSWorkspace.shared.open(URL(fileURLWithPath: path))
    }
}

struct ItermSessionOpener: SessionOpening {
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

struct ItermQuickMessageSender: QuickMessageSending {
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
