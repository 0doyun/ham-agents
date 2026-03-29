import Foundation
import HamCore

public enum NotificationEvent: Equatable, Sendable {
    case done(Agent)
    case waitingInput(Agent)
    case error(Agent)
    case silence(Agent)
    case heartbeat(Agent, minutes: Int)
    case teamDigest(String)
}

public enum NotificationInteraction: Equatable, Sendable {
    case focusAgent(String)
    case openTerminal(String)
    case dismiss(String?)
}

public struct HamNotificationService {
    public init() {}

    public func shouldNotify(for event: NotificationEvent) -> Bool {
        switch event {
        case .done(let agent), .waitingInput(let agent), .error(let agent), .silence(let agent), .heartbeat(let agent, _):
            return agent.notificationPolicy != .muted
        case .teamDigest:
            return true
        }
    }
}

public struct NotificationCandidate: Equatable, Sendable {
    public var event: NotificationEvent
    public var title: String
    public var body: String

    public init(event: NotificationEvent, title: String, body: String) {
        self.event = event
        self.title = title
        self.body = body
    }

    public var agentID: String? {
        switch event {
        case .done(let agent), .waitingInput(let agent), .error(let agent), .silence(let agent), .heartbeat(let agent, _):
            return agent.id
        case .teamDigest:
            return nil
        }
    }

    public var supportsAttentionActions: Bool {
        switch event {
        case .waitingInput, .error:
            return true
        default:
            return false
        }
    }
}

public protocol NotificationSink: Sendable {
    func send(_ candidate: NotificationCandidate)
}

public struct NoopNotificationSink: NotificationSink {
    public init() {}
    public func send(_ candidate: NotificationCandidate) {
        _ = candidate
    }
}

public struct StatusChangeNotificationEngine {
    private let service: HamNotificationService
    private let now: @Sendable () -> Date
    private let silenceThreshold: TimeInterval

    public init(
        service: HamNotificationService = HamNotificationService(),
        now: @escaping @Sendable () -> Date = { Date() },
        silenceThreshold: TimeInterval = 10 * 60
    ) {
        self.service = service
        self.now = now
        self.silenceThreshold = silenceThreshold
    }

    public func candidates(
        previous: [Agent],
        current: [Agent],
        previousObservedAt: Date? = nil,
        currentObservedAt: Date? = nil
    ) -> [NotificationCandidate] {
        let previousByID = Dictionary(uniqueKeysWithValues: previous.map { ($0.id, $0) })
        let currentTime = currentObservedAt ?? now()
        let previousTime = previousObservedAt ?? currentTime

        return current.compactMap { agent in
            guard let oldAgent = previousByID[agent.id] else {
                return nil
            }

            let event: NotificationEvent?
            if oldAgent.status != agent.status {
                switch agent.status {
                case .done:
                    event = .done(agent)
                case .waitingInput:
                    event = .waitingInput(agent)
                case .error:
                    event = .error(agent)
                default:
                    event = nil
                }
            } else if shouldEmitSilenceCandidate(previous: oldAgent, current: agent, previousObservedAt: previousTime, currentObservedAt: currentTime) {
                event = .silence(agent)
            } else {
                event = nil
            }

            guard let event, service.shouldNotify(for: event) else {
                return nil
            }

            return NotificationCandidate(
                event: event,
                title: title(for: event),
                body: body(for: event, observedAt: currentTime)
            )
        }
    }

    public func heartbeatCandidates(
        agents: [Agent],
        observedAt: Date,
        intervalMinutes: Int
    ) -> [NotificationCandidate] {
        guard intervalMinutes > 0 else { return [] }

        return agents.compactMap { agent in
            guard isHeartbeatEligible(agent) else { return nil }
            let startedAt = agent.registeredAt ?? agent.lastEventAt
            let elapsedMinutes = Int(observedAt.timeIntervalSince(startedAt) / 60)
            guard elapsedMinutes >= intervalMinutes else { return nil }
            let status = agent.status.humanizedLabel
            let body: String
            if let summary = agent.lastUserVisibleSummary, !summary.isEmpty {
                body = "\(elapsedMinutes)m in \(status). Last: \(summary)"
            } else {
                body = "\(elapsedMinutes)m in \(status) at \(agent.projectPath)"
            }
            let event = NotificationEvent.heartbeat(agent, minutes: elapsedMinutes)
            guard service.shouldNotify(for: event) else { return nil }
            return NotificationCandidate(
                event: event,
                title: "\(agent.displayName) is still running",
                body: body
            )
        }
    }

    private func title(for event: NotificationEvent) -> String {
        switch event {
        case .done(let agent):
            return "\(agent.displayName) finished"
        case .waitingInput(let agent):
            return "\(agent.displayName) needs input"
        case .error(let agent):
            return "\(agent.displayName) hit an error"
        case .silence(let agent):
            return "\(agent.displayName) went quiet"
        case .heartbeat(let agent, _):
            return "\(agent.displayName) is still running"
        case .teamDigest(let name):
            return "\(name) needs attention"
        }
    }

    private func body(for event: NotificationEvent, observedAt: Date) -> String {
        switch event {
        case .done(let agent), .waitingInput(let agent), .error(let agent):
            return agent.lastUserVisibleSummary ?? "\(agent.status.humanizedLabel) at \(agent.projectPath)"
        case .silence(let agent):
            let duration = humanizedSilenceInterval(agent.lastEventAt, now: observedAt)
            if let summary = agent.lastUserVisibleSummary, !summary.isEmpty {
                return "No activity for \(duration). Last seen: \(summary)"
            }
            return "No activity for \(duration) at \(agent.projectPath)"
        case .heartbeat(let agent, let minutes):
            if let summary = agent.lastUserVisibleSummary, !summary.isEmpty {
                return "\(minutes)m in \(agent.status.humanizedLabel). Last: \(summary)"
            }
            return "\(minutes)m in \(agent.status.humanizedLabel) at \(agent.projectPath)"
        case .teamDigest:
            return "Team requires attention."
        }
    }

    private func shouldEmitSilenceCandidate(
        previous: Agent,
        current: Agent,
        previousObservedAt: Date,
        currentObservedAt: Date
    ) -> Bool {
        guard isSilenceTrackable(previous.status), isSilenceTrackable(current.status) else {
            return false
        }
        let previousAge = previousObservedAt.timeIntervalSince(previous.lastEventAt)
        let currentAge = currentObservedAt.timeIntervalSince(current.lastEventAt)
        return previousAge < silenceThreshold && currentAge >= silenceThreshold
    }
    private func isSilenceTrackable(_ status: AgentStatus) -> Bool {
        status.isRunningActivity
    }

    private func humanizedSilenceInterval(_ date: Date, now: Date) -> String {
        let seconds = max(0, Int(now.timeIntervalSince(date)))
        let minutes = seconds / 60
        if minutes >= 60 {
            return "\(minutes / 60)h"
        }
        if minutes > 0 {
            return "\(minutes)m"
        }
        return "\(seconds)s"
    }

    private func isHeartbeatEligible(_ agent: Agent) -> Bool {
        guard let omcMode = agent.omcMode else { return false }
        switch omcMode {
        case "autopilot", "ralph", "team":
            break
        default:
            return false
        }
        return agent.status.isRunningActivity
    }
}
