import Foundation

public enum DaemonCommand: String, Codable, Sendable {
    case runManaged = "run.managed"
    case listAgents = "agents.list"
    case status = "agents.status"
    case events = "events.list"
}

public struct DaemonRequest: Codable, Equatable, Sendable {
    public var command: DaemonCommand
    public var provider: String?
    public var displayName: String?
    public var projectPath: String?
    public var role: String?
    public var limit: Int?

    public init(
        command: DaemonCommand,
        provider: String? = nil,
        displayName: String? = nil,
        projectPath: String? = nil,
        role: String? = nil,
        limit: Int? = nil
    ) {
        self.command = command
        self.provider = provider
        self.displayName = displayName
        self.projectPath = projectPath
        self.role = role
        self.limit = limit
    }

    enum CodingKeys: String, CodingKey {
        case command
        case provider
        case displayName = "display_name"
        case projectPath = "project_path"
        case role
        case limit
    }
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
    public var events: [AgentEventPayload]?
    public var snapshot: DaemonRuntimeSnapshotPayload?
    public var error: String?

    public init(
        agent: Agent? = nil,
        agents: [Agent]? = nil,
        events: [AgentEventPayload]? = nil,
        snapshot: DaemonRuntimeSnapshotPayload? = nil,
        error: String? = nil
    ) {
        self.agent = agent
        self.agents = agents
        self.events = events
        self.snapshot = snapshot
        self.error = error
    }
}
