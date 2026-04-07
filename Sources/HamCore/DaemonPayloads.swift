import Foundation

public struct DaemonStatusPayload: Codable, Equatable, Sendable {
    public var total: Int
    public var running: Int
    public var waiting: Int
    public var done: Int
    public var generatedAt: Date

    public init(total: Int, running: Int, waiting: Int, done: Int, generatedAt: Date) {
        self.total = total
        self.running = running
        self.waiting = waiting
        self.done = done
        self.generatedAt = generatedAt
    }
}

public struct AgentEventPayload: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var agentID: String
    public var type: String
    public var summary: String
    public var occurredAt: Date
    public var presentationLabel: String?
    public var presentationEmphasis: String?
    public var presentationSummary: String?
    public var lifecycleStatus: String?
    public var lifecycleMode: String?
    public var lifecycleReason: String?
    public var lifecycleConfidence: Double?
    public var sessionID: String?
    public var parentAgentID: String?
    public var taskName: String?
    public var taskDesc: String?
    public var artifactType: String?
    public var artifactRef: String?
    public var artifactData: String?
    public var toolName: String?
    public var toolInput: String?
    public var toolType: String?
    public var toolDurationMs: Int?

    public init(
        id: String,
        agentID: String,
        type: String,
        summary: String,
        occurredAt: Date,
        presentationLabel: String? = nil,
        presentationEmphasis: String? = nil,
        presentationSummary: String? = nil,
        lifecycleStatus: String? = nil,
        lifecycleMode: String? = nil,
        lifecycleReason: String? = nil,
        lifecycleConfidence: Double? = nil,
        sessionID: String? = nil,
        parentAgentID: String? = nil,
        taskName: String? = nil,
        taskDesc: String? = nil,
        artifactType: String? = nil,
        artifactRef: String? = nil,
        artifactData: String? = nil,
        toolName: String? = nil,
        toolInput: String? = nil,
        toolType: String? = nil,
        toolDurationMs: Int? = nil
    ) {
        self.id = id
        self.agentID = agentID
        self.type = type
        self.summary = summary
        self.occurredAt = occurredAt
        self.presentationLabel = presentationLabel
        self.presentationEmphasis = presentationEmphasis
        self.presentationSummary = presentationSummary
        self.lifecycleStatus = lifecycleStatus
        self.lifecycleMode = lifecycleMode
        self.lifecycleReason = lifecycleReason
        self.lifecycleConfidence = lifecycleConfidence
        self.sessionID = sessionID
        self.parentAgentID = parentAgentID
        self.taskName = taskName
        self.taskDesc = taskDesc
        self.artifactType = artifactType
        self.artifactRef = artifactRef
        self.artifactData = artifactData
        self.toolName = toolName
        self.toolInput = toolInput
        self.toolType = toolType
        self.toolDurationMs = toolDurationMs
    }

    enum CodingKeys: String, CodingKey {
        case id
        case agentID = "agent_id"
        case type
        case summary
        case occurredAt = "occurred_at"
        case presentationLabel = "presentation_label"
        case presentationEmphasis = "presentation_emphasis"
        case presentationSummary = "presentation_summary"
        case lifecycleStatus = "lifecycle_status"
        case lifecycleMode = "lifecycle_mode"
        case lifecycleReason = "lifecycle_reason"
        case lifecycleConfidence = "lifecycle_confidence"
        case sessionID = "session_id"
        case parentAgentID = "parent_agent_id"
        case taskName = "task_name"
        case taskDesc = "task_desc"
        case artifactType = "artifact_type"
        case artifactRef = "artifact_ref"
        case artifactData = "artifact_data"
        case toolName = "tool_name"
        case toolInput = "tool_input"
        case toolType = "tool_type"
        case toolDurationMs = "tool_duration_ms"
    }
}

public struct InboxItemPayload: Codable, Equatable, Sendable, Identifiable {
    public let id: String
    public let agentID: String
    public let agentName: String
    public let type: String
    public let summary: String
    public let toolName: String?
    public let occurredAt: Date
    public let read: Bool
    public let actionable: Bool

    public init(
        id: String,
        agentID: String,
        agentName: String,
        type: String,
        summary: String,
        toolName: String? = nil,
        occurredAt: Date,
        read: Bool,
        actionable: Bool
    ) {
        self.id = id
        self.agentID = agentID
        self.agentName = agentName
        self.type = type
        self.summary = summary
        self.toolName = toolName
        self.occurredAt = occurredAt
        self.read = read
        self.actionable = actionable
    }

    enum CodingKeys: String, CodingKey {
        case id
        case agentID = "agent_id"
        case agentName = "agent_name"
        case type
        case summary
        case toolName = "tool_name"
        case occurredAt = "occurred_at"
        case read
        case actionable
    }
}

public enum DaemonJSONDecoder {
    public static func make() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let string = try container.decode(String.self)
            if let date = iso8601WithFractional.date(from: string) { return date }
            if let date = iso8601Plain.date(from: string) { return date }
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Cannot parse date: \(string)")
        }
        return decoder
    }

    private static let iso8601WithFractional: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()

    private static let iso8601Plain: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()
}
