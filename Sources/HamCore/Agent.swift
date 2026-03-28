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
    public var lastEventAt: Date
    public var lastUserVisibleSummary: String?
    public var notificationPolicy: NotificationPolicy
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
        lastEventAt: Date,
        lastUserVisibleSummary: String? = nil,
        notificationPolicy: NotificationPolicy = .default,
        sessionRef: String? = nil,
        sessionTitle: String? = nil,
        sessionIsActive: Bool = false,
        sessionTTY: String? = nil,
        sessionWorkingDirectory: String? = nil,
        sessionActivity: String? = nil,
        sessionProcessID: Int? = nil,
        sessionCommand: String? = nil,
        avatarVariant: String = "default",
        subAgentCount: Int = 0
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
        self.lastEventAt = lastEventAt
        self.lastUserVisibleSummary = lastUserVisibleSummary
        self.notificationPolicy = notificationPolicy
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
        case lastEventAt = "last_event_at"
        case lastUserVisibleSummary = "last_user_visible_summary"
        case notificationPolicy = "notification_policy"
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
        lastEventAt = try c.decodeIfPresent(Date.self, forKey: .lastEventAt) ?? Date()
        lastUserVisibleSummary = try c.decodeIfPresent(String.self, forKey: .lastUserVisibleSummary)
        notificationPolicy = try c.decodeIfPresent(NotificationPolicy.self, forKey: .notificationPolicy) ?? .default
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
    }
}
