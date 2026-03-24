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
    public var lastEventAt: Date
    public var lastUserVisibleSummary: String?
    public var notificationPolicy: NotificationPolicy
    public var sessionRef: String?
    public var avatarVariant: String

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
        lastEventAt: Date,
        lastUserVisibleSummary: String? = nil,
        notificationPolicy: NotificationPolicy = .default,
        sessionRef: String? = nil,
        avatarVariant: String = "default"
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
        self.lastEventAt = lastEventAt
        self.lastUserVisibleSummary = lastUserVisibleSummary
        self.notificationPolicy = notificationPolicy
        self.sessionRef = sessionRef
        self.avatarVariant = avatarVariant
    }
}
