import Foundation

public enum DaemonCommand: String, Codable, Sendable {
    case runManaged = "run.managed"
    case attachSession = "attach.session"
    case observeSource = "observe.source"
    case createTeam = "teams.create"
    case addTeamMember = "teams.add_member"
    case listTeams = "teams.list"
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

public struct DaemonTeamPayload: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var displayName: String
    public var memberAgentIDs: [String]

    public init(id: String, displayName: String, memberAgentIDs: [String]) {
        self.id = id
        self.displayName = displayName
        self.memberAgentIDs = memberAgentIDs
    }

    enum CodingKeys: String, CodingKey {
        case id
        case displayName = "display_name"
        case memberAgentIDs = "member_agent_ids"
    }
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
    public var teamRef: String?
    public var memberAgentID: String?
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
        teamRef: String? = nil,
        memberAgentID: String? = nil,
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
        self.teamRef = teamRef
        self.memberAgentID = memberAgentID
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
        case teamRef = "team_ref"
        case memberAgentID = "member_agent_id"
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

public struct DaemonGeneralSettingsPayload: Codable, Equatable, Sendable {
    public var launchAtLogin: Bool
    public var compactMode: Bool
    public var showMenuBarAnimationAlways: Bool

    public init(launchAtLogin: Bool = false, compactMode: Bool = false, showMenuBarAnimationAlways: Bool = false) {
        self.launchAtLogin = launchAtLogin
        self.compactMode = compactMode
        self.showMenuBarAnimationAlways = showMenuBarAnimationAlways
    }

    enum CodingKeys: String, CodingKey {
        case launchAtLogin = "launch_at_login"
        case compactMode = "compact_mode"
        case showMenuBarAnimationAlways = "show_menu_bar_animation_always"
    }
}

public struct DaemonSettingsPayload: Codable, Equatable, Sendable {
    public var general: DaemonGeneralSettingsPayload
    public var notifications: DaemonNotificationSettingsPayload
    public var appearance: DaemonAppearanceSettingsPayload
    public var integrations: DaemonIntegrationSettingsPayload
    public var privacy: DaemonPrivacySettingsPayload

    public init(
        general: DaemonGeneralSettingsPayload = .init(),
        notifications: DaemonNotificationSettingsPayload,
        appearance: DaemonAppearanceSettingsPayload = .default,
        integrations: DaemonIntegrationSettingsPayload = .default,
        privacy: DaemonPrivacySettingsPayload = .default
    ) {
        self.general = general
        self.notifications = notifications
        self.appearance = appearance
        self.integrations = integrations
        self.privacy = privacy
    }

    public static let `default` = DaemonSettingsPayload(
        general: .init(),
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
        integrations: .default,
        privacy: .default
    )

    enum CodingKeys: String, CodingKey {
        case general
        case notifications
        case appearance
        case integrations
        case privacy
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        let defaults = DaemonSettingsPayload.default
        general = try container.decodeIfPresent(DaemonGeneralSettingsPayload.self, forKey: .general) ?? defaults.general
        notifications = try container.decodeIfPresent(DaemonNotificationSettingsPayload.self, forKey: .notifications) ?? defaults.notifications
        appearance = try container.decodeIfPresent(DaemonAppearanceSettingsPayload.self, forKey: .appearance) ?? defaults.appearance
        integrations = try container.decodeIfPresent(DaemonIntegrationSettingsPayload.self, forKey: .integrations) ?? defaults.integrations
        privacy = try container.decodeIfPresent(DaemonPrivacySettingsPayload.self, forKey: .privacy) ?? defaults.privacy
    }
}

public struct DaemonAppearanceSettingsPayload: Codable, Equatable, Sendable {
    public var theme: String
    public var animationSpeedMultiplier: Double
    public var reduceMotion: Bool
    public var hamsterSkin: String
    public var hat: String
    public var deskTheme: String

    public init(theme: String, animationSpeedMultiplier: Double = 1, reduceMotion: Bool = false, hamsterSkin: String = "default", hat: String = "none", deskTheme: String = "classic") {
        self.theme = theme
        self.animationSpeedMultiplier = animationSpeedMultiplier
        self.reduceMotion = reduceMotion
        self.hamsterSkin = hamsterSkin
        self.hat = hat
        self.deskTheme = deskTheme
    }

    public static let `default` = DaemonAppearanceSettingsPayload(theme: "auto", animationSpeedMultiplier: 1, reduceMotion: false, hamsterSkin: "default", hat: "none", deskTheme: "classic")

    enum CodingKeys: String, CodingKey {
        case theme
        case animationSpeedMultiplier = "animation_speed_multiplier"
        case reduceMotion = "reduce_motion"
        case hamsterSkin = "hamster_skin"
        case hat
        case deskTheme = "desk_theme"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        theme = try container.decodeIfPresent(String.self, forKey: .theme) ?? "auto"
        animationSpeedMultiplier = try container.decodeIfPresent(Double.self, forKey: .animationSpeedMultiplier) ?? 1
        reduceMotion = try container.decodeIfPresent(Bool.self, forKey: .reduceMotion) ?? false
        hamsterSkin = try container.decodeIfPresent(String.self, forKey: .hamsterSkin) ?? "default"
        hat = try container.decodeIfPresent(String.self, forKey: .hat) ?? "none"
        deskTheme = try container.decodeIfPresent(String.self, forKey: .deskTheme) ?? "classic"
    }
}

public struct DaemonIntegrationSettingsPayload: Codable, Equatable, Sendable {
    public var itermEnabled: Bool
    public var transcriptDirs: [String]
    public var providerAdapters: [String: Bool]

    public init(itermEnabled: Bool, transcriptDirs: [String] = [], providerAdapters: [String: Bool] = [:]) {
        self.itermEnabled = itermEnabled
        self.transcriptDirs = transcriptDirs
        self.providerAdapters = providerAdapters
    }

    public static let `default` = DaemonIntegrationSettingsPayload(itermEnabled: true, transcriptDirs: [], providerAdapters: ["claude": true, "generic_process": true, "transcript": true])

    enum CodingKeys: String, CodingKey {
        case itermEnabled = "iterm_enabled"
        case transcriptDirs = "transcript_dirs"
        case providerAdapters = "provider_adapters"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        itermEnabled = try container.decodeIfPresent(Bool.self, forKey: .itermEnabled) ?? true
        transcriptDirs = try container.decodeIfPresent([String].self, forKey: .transcriptDirs) ?? []
        providerAdapters = try container.decodeIfPresent([String: Bool].self, forKey: .providerAdapters) ?? DaemonIntegrationSettingsPayload.default.providerAdapters
    }
}

public struct DaemonPrivacySettingsPayload: Codable, Equatable, Sendable {
    public var localOnlyMode: Bool
    public var eventHistoryRetentionDays: Int
    public var transcriptExcerptStorage: Bool

    public init(localOnlyMode: Bool, eventHistoryRetentionDays: Int, transcriptExcerptStorage: Bool) {
        self.localOnlyMode = localOnlyMode
        self.eventHistoryRetentionDays = eventHistoryRetentionDays
        self.transcriptExcerptStorage = transcriptExcerptStorage
    }

    public static let `default` = DaemonPrivacySettingsPayload(localOnlyMode: true, eventHistoryRetentionDays: 30, transcriptExcerptStorage: true)

    enum CodingKeys: String, CodingKey {
        case localOnlyMode = "local_only_mode"
        case eventHistoryRetentionDays = "event_history_retention_days"
        case transcriptExcerptStorage = "transcript_excerpt_storage"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        localOnlyMode = try container.decodeIfPresent(Bool.self, forKey: .localOnlyMode) ?? DaemonPrivacySettingsPayload.default.localOnlyMode
        eventHistoryRetentionDays = try container.decodeIfPresent(Int.self, forKey: .eventHistoryRetentionDays) ?? DaemonPrivacySettingsPayload.default.eventHistoryRetentionDays
        transcriptExcerptStorage = try container.decodeIfPresent(Bool.self, forKey: .transcriptExcerptStorage) ?? DaemonPrivacySettingsPayload.default.transcriptExcerptStorage
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
        agents.filter { $0.status.isRunningActivity }.count
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
    public var team: DaemonTeamPayload?
    public var agents: [Agent]?
    public var teams: [DaemonTeamPayload]?
    public var attachableSessions: [DaemonAttachableSessionPayload]?
    public var events: [AgentEventPayload]?
    public var snapshot: DaemonRuntimeSnapshotPayload?
    public var settings: DaemonSettingsPayload?
    public var error: String?

    public init(
        agent: Agent? = nil,
        team: DaemonTeamPayload? = nil,
        agents: [Agent]? = nil,
        teams: [DaemonTeamPayload]? = nil,
        attachableSessions: [DaemonAttachableSessionPayload]? = nil,
        events: [AgentEventPayload]? = nil,
        snapshot: DaemonRuntimeSnapshotPayload? = nil,
        settings: DaemonSettingsPayload? = nil,
        error: String? = nil
    ) {
        self.agent = agent
        self.team = team
        self.agents = agents
        self.teams = teams
        self.attachableSessions = attachableSessions
        self.events = events
        self.snapshot = snapshot
        self.settings = settings
        self.error = error
    }

    enum CodingKeys: String, CodingKey {
        case agent
        case team
        case agents
        case teams
        case attachableSessions = "attachable_sessions"
        case events
        case snapshot
        case settings
        case error
    }
}
