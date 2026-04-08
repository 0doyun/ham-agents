import Foundation

/// Mirror of `core.CostRecord` from the Go side. Decoded out of the
/// `cost_records` array on a `cost.summary` daemon response.
public struct CostRecordPayload: Codable, Equatable, Sendable {
    public var agentID: String?
    public var sessionID: String?
    public var projectPath: String?
    public var model: String
    public var serviceTier: String?
    public var inputTokens: Int64
    public var cacheCreate5mTokens: Int64
    public var cacheCreate1hTokens: Int64
    public var cacheReadTokens: Int64
    public var outputTokens: Int64
    public var webSearchRequests: Int64
    public var webFetchRequests: Int64
    public var estimatedUSD: Double
    public var recordedAt: Date
    public var source: String
    public var requestID: String?
    public var messageID: String?

    public init(
        agentID: String? = nil,
        sessionID: String? = nil,
        projectPath: String? = nil,
        model: String,
        serviceTier: String? = nil,
        inputTokens: Int64 = 0,
        cacheCreate5mTokens: Int64 = 0,
        cacheCreate1hTokens: Int64 = 0,
        cacheReadTokens: Int64 = 0,
        outputTokens: Int64 = 0,
        webSearchRequests: Int64 = 0,
        webFetchRequests: Int64 = 0,
        estimatedUSD: Double = 0,
        recordedAt: Date,
        source: String,
        requestID: String? = nil,
        messageID: String? = nil
    ) {
        self.agentID = agentID
        self.sessionID = sessionID
        self.projectPath = projectPath
        self.model = model
        self.serviceTier = serviceTier
        self.inputTokens = inputTokens
        self.cacheCreate5mTokens = cacheCreate5mTokens
        self.cacheCreate1hTokens = cacheCreate1hTokens
        self.cacheReadTokens = cacheReadTokens
        self.outputTokens = outputTokens
        self.webSearchRequests = webSearchRequests
        self.webFetchRequests = webFetchRequests
        self.estimatedUSD = estimatedUSD
        self.recordedAt = recordedAt
        self.source = source
        self.requestID = requestID
        self.messageID = messageID
    }

    enum CodingKeys: String, CodingKey {
        case agentID = "agent_id"
        case sessionID = "session_id"
        case projectPath = "project_path"
        case model
        case serviceTier = "service_tier"
        case inputTokens = "input_tokens"
        case cacheCreate5mTokens = "cache_create_5m_tokens"
        case cacheCreate1hTokens = "cache_create_1h_tokens"
        case cacheReadTokens = "cache_read_tokens"
        case outputTokens = "output_tokens"
        case webSearchRequests = "web_search_requests"
        case webFetchRequests = "web_fetch_requests"
        case estimatedUSD = "estimated_usd"
        case recordedAt = "recorded_at"
        case source
        case requestID = "request_id"
        case messageID = "message_id"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        agentID = try container.decodeIfPresent(String.self, forKey: .agentID)
        sessionID = try container.decodeIfPresent(String.self, forKey: .sessionID)
        projectPath = try container.decodeIfPresent(String.self, forKey: .projectPath)
        model = try container.decode(String.self, forKey: .model)
        serviceTier = try container.decodeIfPresent(String.self, forKey: .serviceTier)
        inputTokens = try container.decodeIfPresent(Int64.self, forKey: .inputTokens) ?? 0
        cacheCreate5mTokens = try container.decodeIfPresent(Int64.self, forKey: .cacheCreate5mTokens) ?? 0
        cacheCreate1hTokens = try container.decodeIfPresent(Int64.self, forKey: .cacheCreate1hTokens) ?? 0
        cacheReadTokens = try container.decodeIfPresent(Int64.self, forKey: .cacheReadTokens) ?? 0
        outputTokens = try container.decodeIfPresent(Int64.self, forKey: .outputTokens) ?? 0
        webSearchRequests = try container.decodeIfPresent(Int64.self, forKey: .webSearchRequests) ?? 0
        webFetchRequests = try container.decodeIfPresent(Int64.self, forKey: .webFetchRequests) ?? 0
        estimatedUSD = try container.decodeIfPresent(Double.self, forKey: .estimatedUSD) ?? 0
        recordedAt = try container.decode(Date.self, forKey: .recordedAt)
        source = try container.decodeIfPresent(String.self, forKey: .source) ?? "assistant"
        requestID = try container.decodeIfPresent(String.self, forKey: .requestID)
        messageID = try container.decodeIfPresent(String.self, forKey: .messageID)
    }
}

/// Aggregate cost summary returned by `HamDaemonClient.fetchCostSummary`.
/// totalUSD is the rollup over the requested window; todayUSD is the subset
/// recorded since UTC midnight; byModel buckets the same window by model id.
public struct CostSummaryPayload: Equatable, Sendable {
    public var totalUSD: Double
    public var todayUSD: Double
    public var byModel: [String: Double]
    public var byDay: [String: Double]
    public var byAgent: [String: Double]
    public var records: [CostRecordPayload]

    public init(
        totalUSD: Double,
        todayUSD: Double,
        byModel: [String: Double],
        byDay: [String: Double],
        byAgent: [String: Double],
        records: [CostRecordPayload]
    ) {
        self.totalUSD = totalUSD
        self.todayUSD = todayUSD
        self.byModel = byModel
        self.byDay = byDay
        self.byAgent = byAgent
        self.records = records
    }

    /// Build a CostSummaryPayload from the daemon response, computing the
    /// today rollup locally so the daemon stays stateless w.r.t. wall-clock
    /// midnight.
    public static func from(response: DaemonResponse, now: Date = Date()) -> CostSummaryPayload {
        let calendar = Calendar(identifier: .gregorian)
        var utcCalendar = calendar
        utcCalendar.timeZone = TimeZone(identifier: "UTC") ?? TimeZone.current
        let todayKey = todayKey(for: now, calendar: utcCalendar)
        let byDay = response.byDay ?? [:]
        let todayUSD = byDay[todayKey] ?? 0
        return CostSummaryPayload(
            totalUSD: response.totalUSD ?? 0,
            todayUSD: todayUSD,
            byModel: response.byModel ?? [:],
            byDay: byDay,
            byAgent: response.byAgent ?? [:],
            records: response.costRecords ?? []
        )
    }

    private static func todayKey(for now: Date, calendar: Calendar) -> String {
        let formatter = DateFormatter()
        formatter.calendar = calendar
        formatter.timeZone = calendar.timeZone
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: now)
    }
}
