import Foundation

public enum DaemonCommand: String, Codable, Sendable {
    case runManaged = "run.managed"
    case attachSession = "attach.session"
    case observeSource = "observe.source"
    case listItermSessions = "iterm.sessions"
    case listAgents = "agents.list"
    case status = "agents.status"
    case events = "events.list"
    case followEvents = "events.follow"
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
    public var afterEventID: String?
    public var waitMillis: Int?
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
        afterEventID: String? = nil,
        waitMillis: Int? = nil,
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
        self.afterEventID = afterEventID
        self.waitMillis = waitMillis
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
        case afterEventID = "after_event_id"
        case waitMillis = "wait_millis"
        case policy
        case settings
    }
}

public struct DaemonNotificationSettingsPayload: Codable, Equatable, Sendable {
    public var done: Bool
    public var error: Bool
    public var waitingInput: Bool
    public var silence: Bool
    public var quietHoursEnabled: Bool
    public var quietHoursStartHour: Int
    public var quietHoursEndHour: Int
    public var previewText: Bool

    public init(
        done: Bool,
        error: Bool,
        waitingInput: Bool,
        silence: Bool = false,
        quietHoursEnabled: Bool,
        quietHoursStartHour: Int,
        quietHoursEndHour: Int,
        previewText: Bool
    ) {
        self.done = done
        self.error = error
        self.waitingInput = waitingInput
        self.silence = silence
        self.quietHoursEnabled = quietHoursEnabled
        self.quietHoursStartHour = quietHoursStartHour
        self.quietHoursEndHour = quietHoursEndHour
        self.previewText = previewText
    }

    enum CodingKeys: String, CodingKey {
        case done
        case error
        case waitingInput = "waiting_input"
        case silence
        case quietHoursEnabled = "quiet_hours_enabled"
        case quietHoursStartHour = "quiet_hours_start_hour"
        case quietHoursEndHour = "quiet_hours_end_hour"
        case previewText = "preview_text"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        done = try container.decode(Bool.self, forKey: .done)
        error = try container.decode(Bool.self, forKey: .error)
        waitingInput = try container.decode(Bool.self, forKey: .waitingInput)
        silence = try container.decodeIfPresent(Bool.self, forKey: .silence) ?? false
        quietHoursEnabled = try container.decode(Bool.self, forKey: .quietHoursEnabled)
        quietHoursStartHour = try container.decode(Int.self, forKey: .quietHoursStartHour)
        quietHoursEndHour = try container.decode(Int.self, forKey: .quietHoursEndHour)
        previewText = try container.decode(Bool.self, forKey: .previewText)
    }
}

public struct DaemonSettingsPayload: Codable, Equatable, Sendable {
    public var notifications: DaemonNotificationSettingsPayload
    public var appearance: DaemonAppearanceSettingsPayload
    public var integrations: DaemonIntegrationSettingsPayload

    public init(
        notifications: DaemonNotificationSettingsPayload,
        appearance: DaemonAppearanceSettingsPayload = .default,
        integrations: DaemonIntegrationSettingsPayload = .default
    ) {
        self.notifications = notifications
        self.appearance = appearance
        self.integrations = integrations
    }

    public static let `default` = DaemonSettingsPayload(
        notifications: DaemonNotificationSettingsPayload(
            done: true,
            error: true,
            waitingInput: true,
            silence: false,
            quietHoursEnabled: false,
            quietHoursStartHour: 22,
            quietHoursEndHour: 8,
            previewText: false
        ),
        appearance: .default,
        integrations: .default
    )
}

public struct DaemonAppearanceSettingsPayload: Codable, Equatable, Sendable {
    public var theme: String

    public init(theme: String) {
        self.theme = theme
    }

    public static let `default` = DaemonAppearanceSettingsPayload(theme: "auto")
}

public struct DaemonIntegrationSettingsPayload: Codable, Equatable, Sendable {
    public var itermEnabled: Bool

    public init(itermEnabled: Bool) {
        self.itermEnabled = itermEnabled
    }

    public static let `default` = DaemonIntegrationSettingsPayload(itermEnabled: true)

    enum CodingKeys: String, CodingKey {
        case itermEnabled = "iterm_enabled"
    }
}

public struct DaemonRuntimeSnapshotPayload: Codable, Equatable, Sendable {
    public var agents: [Agent]
    public var generatedAt: Date
    public var attentionCount: Int
    public var attentionBreakdown: DaemonAttentionBreakdownPayload
    public var attentionOrder: [String]
    public var attentionSubtitles: [String: String]

    public init(
        agents: [Agent],
        generatedAt: Date,
        attentionCount: Int = 0,
        attentionBreakdown: DaemonAttentionBreakdownPayload = .init(),
        attentionOrder: [String] = [],
        attentionSubtitles: [String: String] = [:]
    ) {
        self.agents = agents
        self.generatedAt = generatedAt
        self.attentionCount = attentionCount
        self.attentionBreakdown = attentionBreakdown
        self.attentionOrder = attentionOrder
        self.attentionSubtitles = attentionSubtitles
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
        case attentionCount = "attention_count"
        case attentionBreakdown = "attention_breakdown"
        case attentionOrder = "attention_order"
        case attentionSubtitles = "attention_subtitles"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        agents = try container.decode([Agent].self, forKey: .agents)
        generatedAt = try container.decode(Date.self, forKey: .generatedAt)
        attentionCount = try container.decodeIfPresent(Int.self, forKey: .attentionCount) ?? 0
        attentionBreakdown = try container.decodeIfPresent(DaemonAttentionBreakdownPayload.self, forKey: .attentionBreakdown) ?? .init()
        attentionOrder = try container.decodeIfPresent([String].self, forKey: .attentionOrder) ?? []
        attentionSubtitles = try container.decodeIfPresent([String: String].self, forKey: .attentionSubtitles) ?? [:]
    }
}

public struct DaemonAttentionBreakdownPayload: Codable, Equatable, Sendable {
    public var error: Int
    public var waitingInput: Int
    public var disconnected: Int

    public init(error: Int = 0, waitingInput: Int = 0, disconnected: Int = 0) {
        self.error = error
        self.waitingInput = waitingInput
        self.disconnected = disconnected
    }

    enum CodingKeys: String, CodingKey {
        case error
        case waitingInput = "waiting_input"
        case disconnected
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
