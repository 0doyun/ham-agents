import Foundation
import HamCore

public enum NotificationEvent: Equatable, Sendable {
    case done(Agent)
    case waitingInput(Agent)
    case error(Agent)
}

public struct HamNotificationService {
    public init() {}

    public func shouldNotify(for event: NotificationEvent) -> Bool {
        switch event {
        case .done(let agent), .waitingInput(let agent), .error(let agent):
            return agent.notificationPolicy != .muted
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

    public init(service: HamNotificationService = HamNotificationService()) {
        self.service = service
    }

    public func candidates(previous: [Agent], current: [Agent]) -> [NotificationCandidate] {
        let previousByID = Dictionary(uniqueKeysWithValues: previous.map { ($0.id, $0) })

        return current.compactMap { agent in
            guard let oldAgent = previousByID[agent.id], oldAgent.status != agent.status else {
                return nil
            }

            let event: NotificationEvent?
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

            guard let event, service.shouldNotify(for: event) else {
                return nil
            }

            return NotificationCandidate(
                event: event,
                title: title(for: event),
                body: body(for: event)
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
        }
    }

    private func body(for event: NotificationEvent) -> String {
        switch event {
        case .done(let agent), .waitingInput(let agent), .error(let agent):
            return agent.lastUserVisibleSummary ?? "\(humanizedStatusLabel(agent.status)) at \(agent.projectPath)"
        }
    }

    private func humanizedStatusLabel(_ status: AgentStatus) -> String {
        switch status {
        case .waitingInput:
            return "needs input"
        case .runningTool:
            return "running tool"
        default:
            return status.rawValue.replacingOccurrences(of: "_", with: " ")
        }
    }
}
