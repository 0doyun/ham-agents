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

    public init(
        id: String,
        agentID: String,
        type: String,
        summary: String,
        occurredAt: Date,
        presentationLabel: String? = nil,
        presentationEmphasis: String? = nil,
        presentationSummary: String? = nil
    ) {
        self.id = id
        self.agentID = agentID
        self.type = type
        self.summary = summary
        self.occurredAt = occurredAt
        self.presentationLabel = presentationLabel
        self.presentationEmphasis = presentationEmphasis
        self.presentationSummary = presentationSummary
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
    }
}

public enum DaemonJSONDecoder {
    public static func make() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return decoder
    }
}
