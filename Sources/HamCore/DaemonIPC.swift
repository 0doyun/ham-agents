import Foundation

public enum DaemonCommand: String, Codable, Sendable {
    case runManaged = "run.managed"
    case attachSession = "attach.session"
    case observeSource = "observe.source"
    case listItermSessions = "iterm.sessions"
    case listAgents = "agents.list"
    case status = "agents.status"
    case events = "events.list"
    case setNotificationPolicy = "agents.set_notification_policy"
    case setRole = "agents.set_role"
    case removeAgent = "agents.remove"
    case getSettings = "settings.get"
    case updateSettings = "settings.update"
}

public struct DaemonAttachableSessionPayload: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var title: String
    public var sessionRef: String
    public var isActive: Bool

    public init(id: String, title: String, sessionRef: String, isActive: Bool) {
        self.id = id
        self.title = title
        self.sessionRef = sessionRef
        self.isActive = isActive
    }

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case sessionRef = "session_ref"
        case isActive = "is_active"
    }
}

public struct DaemonRequest: Codable, Equatable, Sendable {
    public var command: DaemonCommand
    public var agentID: String?
    public var provider: String?
    public var displayName: String?
    public var projectPath: String?
    public var role: String?
    public var limit: Int?
    public var policy: String?
    public var settings: DaemonSettingsPayload?

    public init(
        command: DaemonCommand,
        agentID: String? = nil,
        provider: String? = nil,
        displayName: String? = nil,
        projectPath: String? = nil,
        role: String? = nil,
        limit: Int? = nil,
        policy: String? = nil,
        settings: DaemonSettingsPayload? = nil
    ) {
        self.command = command
        self.agentID = agentID
        self.provider = provider
        self.displayName = displayName
        self.projectPath = projectPath
        self.role = role
        self.limit = limit
        self.policy = policy
        self.settings = settings
    }

    enum CodingKeys: String, CodingKey {
        case command
        case agentID = "agent_id"
        case provider
        case displayName = "display_name"
        case projectPath = "project_path"
        case role
        case limit
        case policy
        case settings
    }
}

public struct DaemonNotificationSettingsPayload: Codable, Equatable, Sendable {
    public var done: Bool
    public var error: Bool
    public var waitingInput: Bool
    public var quietHoursEnabled: Bool
    public var quietHoursStartHour: Int
    public var quietHoursEndHour: Int
    public var previewText: Bool

    public init(
        done: Bool,
        error: Bool,
        waitingInput: Bool,
        quietHoursEnabled: Bool,
        quietHoursStartHour: Int,
        quietHoursEndHour: Int,
        previewText: Bool
    ) {
        self.done = done
        self.error = error
        self.waitingInput = waitingInput
        self.quietHoursEnabled = quietHoursEnabled
        self.quietHoursStartHour = quietHoursStartHour
        self.quietHoursEndHour = quietHoursEndHour
        self.previewText = previewText
    }

    enum CodingKeys: String, CodingKey {
        case done
        case error
        case waitingInput = "waiting_input"
        case quietHoursEnabled = "quiet_hours_enabled"
        case quietHoursStartHour = "quiet_hours_start_hour"
        case quietHoursEndHour = "quiet_hours_end_hour"
        case previewText = "preview_text"
    }
}

public struct DaemonSettingsPayload: Codable, Equatable, Sendable {
    public var notifications: DaemonNotificationSettingsPayload

    public init(notifications: DaemonNotificationSettingsPayload) {
        self.notifications = notifications
    }

    public static let `default` = DaemonSettingsPayload(
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

public struct DaemonRuntimeSnapshotPayload: Codable, Equatable, Sendable {
    public var agents: [Agent]
    public var generatedAt: Date

    public init(agents: [Agent], generatedAt: Date) {
        self.agents = agents
        self.generatedAt = generatedAt
    }

    public var totalCount: Int { agents.count }
    public var runningCount: Int {
        agents.filter { [.booting, .thinking, .reading, .runningTool].contains($0.status) }.count
    }
    public var waitingCount: Int {
        agents.filter { $0.status == .waitingInput }.count
    }
    public var doneCount: Int {
        agents.filter { $0.status == .done }.count
    }

    enum CodingKeys: String, CodingKey {
        case agents
        case generatedAt = "generated_at"
    }
}

public struct DaemonResponse: Codable, Equatable, Sendable {
    public var agent: Agent?
    public var agents: [Agent]?
    public var attachableSessions: [DaemonAttachableSessionPayload]?
    public var events: [AgentEventPayload]?
    public var snapshot: DaemonRuntimeSnapshotPayload?
    public var settings: DaemonSettingsPayload?
    public var error: String?

    public init(
        agent: Agent? = nil,
        agents: [Agent]? = nil,
        attachableSessions: [DaemonAttachableSessionPayload]? = nil,
        events: [AgentEventPayload]? = nil,
        snapshot: DaemonRuntimeSnapshotPayload? = nil,
        settings: DaemonSettingsPayload? = nil,
        error: String? = nil
    ) {
        self.agent = agent
        self.agents = agents
        self.attachableSessions = attachableSessions
        self.events = events
        self.snapshot = snapshot
        self.settings = settings
        self.error = error
    }

    enum CodingKeys: String, CodingKey {
        case agent
        case agents
        case attachableSessions = "attachable_sessions"
        case events
        case snapshot
        case settings
        case error
    }
}
