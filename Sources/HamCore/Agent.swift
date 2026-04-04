import Foundation

public enum AgentMode: String, Codable, CaseIterable, Sendable {
    case managed
    case attached
    case observed
}

public enum AgentStatus: String, Codable, CaseIterable, Sendable {
    case booting
    case idle
    case thinking
    case reading
    case runningTool = "running_tool"
    case waitingInput = "waiting_input"
    case done
    case error
    case disconnected
    case sleeping
}

public enum NotificationPolicy: String, Codable, CaseIterable, Sendable {
    case `default`
    case muted
    case priorityOnly
}

public struct ToolActivity: Codable, Equatable, Sendable {
    public var toolName: String
    public var inputPreview: String?
    public var activityDesc: String?
    public var toolType: String?
    public var startedAt: Date
    public var completedAt: Date?
    public var durationMs: Int?

    enum CodingKeys: String, CodingKey {
        case toolName = "tool_name"
        case inputPreview = "input_preview"
        case activityDesc = "activity_desc"
        case toolType = "tool_type"
        case startedAt = "started_at"
        case completedAt = "completed_at"
        case durationMs = "duration_ms"
    }
}

public struct Agent: Codable, Equatable, Identifiable, Sendable {
    public let id: String
    public var displayName: String
    public var provider: String
    public var host: String
    public var mode: AgentMode
    public var projectPath: String
    public var role: String?
    public var status: AgentStatus
    public var statusConfidence: Double
    public var statusReason: String?
    public var errorType: String?
    public var registeredAt: Date?
    public var lastEventAt: Date
    public var lastUserVisibleSummary: String?
    public var recentTools: [String]
    public var recentToolsDetailed: [ToolActivity]
    public var omcMode: String?
    public var notificationPolicy: NotificationPolicy
    public var sessionID: String?
    public var sessionRef: String?
    public var sessionTitle: String?
    public var sessionIsActive: Bool
    public var sessionTTY: String?
    public var sessionWorkingDirectory: String?
    public var sessionActivity: String?
    public var sessionProcessID: Int?
    public var sessionCommand: String?
    public var avatarVariant: String
    public var subAgentCount: Int
    public var teamRole: String?
    public var teamTaskTotal: Int
    public var teamTaskCompleted: Int

    public init(
        id: String,
        displayName: String,
        provider: String,
        host: String,
        mode: AgentMode,
        projectPath: String,
        role: String? = nil,
        status: AgentStatus,
        statusConfidence: Double,
        statusReason: String? = nil,
        errorType: String? = nil,
        registeredAt: Date? = nil,
        lastEventAt: Date,
        lastUserVisibleSummary: String? = nil,
        recentTools: [String] = [],
        recentToolsDetailed: [ToolActivity] = [],
        omcMode: String? = nil,
        notificationPolicy: NotificationPolicy = .default,
        sessionID: String? = nil,
        sessionRef: String? = nil,
        sessionTitle: String? = nil,
        sessionIsActive: Bool = false,
        sessionTTY: String? = nil,
        sessionWorkingDirectory: String? = nil,
        sessionActivity: String? = nil,
        sessionProcessID: Int? = nil,
        sessionCommand: String? = nil,
        avatarVariant: String = "default",
        subAgentCount: Int = 0,
        teamRole: String? = nil,
        teamTaskTotal: Int = 0,
        teamTaskCompleted: Int = 0
    ) {
        self.id = id
        self.displayName = displayName
        self.provider = provider
        self.host = host
        self.mode = mode
        self.projectPath = projectPath
        self.role = role
        self.status = status
        self.statusConfidence = statusConfidence
        self.statusReason = statusReason
        self.errorType = errorType
        self.registeredAt = registeredAt
        self.lastEventAt = lastEventAt
        self.lastUserVisibleSummary = lastUserVisibleSummary
        self.recentTools = recentTools
        self.recentToolsDetailed = recentToolsDetailed
        self.omcMode = omcMode
        self.notificationPolicy = notificationPolicy
        self.sessionID = sessionID
        self.sessionRef = sessionRef
        self.sessionTitle = sessionTitle
        self.sessionIsActive = sessionIsActive
        self.sessionTTY = sessionTTY
        self.sessionWorkingDirectory = sessionWorkingDirectory
        self.sessionActivity = sessionActivity
        self.sessionProcessID = sessionProcessID
        self.sessionCommand = sessionCommand
        self.avatarVariant = avatarVariant
        self.subAgentCount = subAgentCount
        self.teamRole = teamRole
        self.teamTaskTotal = teamTaskTotal
        self.teamTaskCompleted = teamTaskCompleted
    }

    enum CodingKeys: String, CodingKey {
        case id
        case displayName = "display_name"
        case provider
        case host
        case mode
        case projectPath = "project_path"
        case role
        case status
        case statusConfidence = "status_confidence"
        case statusReason = "status_reason"
        case errorType = "error_type"
        case registeredAt = "registered_at"
        case lastEventAt = "last_event_at"
        case lastUserVisibleSummary = "last_user_visible_summary"
        case recentTools = "recent_tools"
        case recentToolsDetailed = "recent_tools_detailed"
        case omcMode = "omc_mode"
        case notificationPolicy = "notification_policy"
        case sessionID = "session_id"
        case sessionRef = "session_ref"
        case sessionTitle = "session_title"
        case sessionIsActive = "session_is_active"
        case sessionTTY = "session_tty"
        case sessionWorkingDirectory = "session_working_directory"
        case sessionActivity = "session_activity"
        case sessionProcessID = "session_process_id"
        case sessionCommand = "session_command"
        case avatarVariant = "avatar_variant"
        case subAgentCount = "sub_agent_count"
        case teamRole = "team_role"
        case teamTaskTotal = "team_task_total"
        case teamTaskCompleted = "team_task_completed"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        displayName = try c.decode(String.self, forKey: .displayName)
        provider = try c.decode(String.self, forKey: .provider)
        host = try c.decodeIfPresent(String.self, forKey: .host) ?? ""
        mode = try c.decode(AgentMode.self, forKey: .mode)
        projectPath = try c.decode(String.self, forKey: .projectPath)
        role = try c.decodeIfPresent(String.self, forKey: .role)
        status = try c.decode(AgentStatus.self, forKey: .status)
        statusConfidence = try c.decodeIfPresent(Double.self, forKey: .statusConfidence) ?? 0
        statusReason = try c.decodeIfPresent(String.self, forKey: .statusReason)
        errorType = try c.decodeIfPresent(String.self, forKey: .errorType)
        registeredAt = try c.decodeIfPresent(Date.self, forKey: .registeredAt)
        lastEventAt = try c.decodeIfPresent(Date.self, forKey: .lastEventAt) ?? Date()
        lastUserVisibleSummary = try c.decodeIfPresent(String.self, forKey: .lastUserVisibleSummary)
        recentTools = try c.decodeIfPresent([String].self, forKey: .recentTools) ?? []
        recentToolsDetailed = try c.decodeIfPresent([ToolActivity].self, forKey: .recentToolsDetailed) ?? []
        omcMode = try c.decodeIfPresent(String.self, forKey: .omcMode)
        notificationPolicy = try c.decodeIfPresent(NotificationPolicy.self, forKey: .notificationPolicy) ?? .default
        sessionID = try c.decodeIfPresent(String.self, forKey: .sessionID)
        sessionRef = try c.decodeIfPresent(String.self, forKey: .sessionRef)
        sessionTitle = try c.decodeIfPresent(String.self, forKey: .sessionTitle)
        sessionIsActive = try c.decodeIfPresent(Bool.self, forKey: .sessionIsActive) ?? false
        sessionTTY = try c.decodeIfPresent(String.self, forKey: .sessionTTY)
        sessionWorkingDirectory = try c.decodeIfPresent(String.self, forKey: .sessionWorkingDirectory)
        sessionActivity = try c.decodeIfPresent(String.self, forKey: .sessionActivity)
        sessionProcessID = try c.decodeIfPresent(Int.self, forKey: .sessionProcessID)
        sessionCommand = try c.decodeIfPresent(String.self, forKey: .sessionCommand)
        avatarVariant = try c.decodeIfPresent(String.self, forKey: .avatarVariant) ?? "default"
        subAgentCount = try c.decodeIfPresent(Int.self, forKey: .subAgentCount) ?? 0
        teamRole = try c.decodeIfPresent(String.self, forKey: .teamRole)
        teamTaskTotal = try c.decodeIfPresent(Int.self, forKey: .teamTaskTotal) ?? 0
        teamTaskCompleted = try c.decodeIfPresent(Int.self, forKey: .teamTaskCompleted) ?? 0
    }
}
