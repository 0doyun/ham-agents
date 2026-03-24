import Foundation
import HamCore

public enum AgentEventEmphasis: String, Equatable, Sendable {
    case neutral
    case positive
    case warning
    case info
}

public struct AgentEventPresentation: Equatable, Sendable {
    public var label: String
    public var emphasis: AgentEventEmphasis

    public init(label: String, emphasis: AgentEventEmphasis) {
        self.label = label
        self.emphasis = emphasis
    }
}

public struct AgentEventSummaryChip: Equatable, Sendable, Identifiable {
    public var id: String { "\(label)-\(emphasis.rawValue)" }
    public var label: String
    public var emphasis: AgentEventEmphasis
    public var count: Int

    public init(label: String, emphasis: AgentEventEmphasis, count: Int) {
        self.label = label
        self.emphasis = emphasis
        self.count = count
    }
}

public enum AgentEventPresenter {
    public static func present(_ event: AgentEventPayload) -> AgentEventPresentation {
        switch event.type {
        case "agent.registered":
            return AgentEventPresentation(label: "Registered", emphasis: .info)
        case "agent.role_updated":
            return AgentEventPresentation(label: "Role", emphasis: .info)
        case "agent.notification_policy_updated":
            return AgentEventPresentation(label: "Notifications", emphasis: .info)
        case "agent.disconnected":
            return AgentEventPresentation(label: "Disconnected", emphasis: .warning)
        case "agent.reconnected":
            return AgentEventPresentation(label: "Reconnected", emphasis: .positive)
        case "agent.removed":
            return AgentEventPresentation(label: "Stopped", emphasis: .neutral)
        default:
            return AgentEventPresentation(label: event.type, emphasis: .neutral)
        }
    }

    public static func summarize(_ events: [AgentEventPayload]) -> [AgentEventSummaryChip] {
        var buckets: [String: AgentEventSummaryChip] = [:]

        for event in events {
            let presentation = present(event)
            let key = "\(presentation.label)-\(presentation.emphasis.rawValue)"
            if var existing = buckets[key] {
                existing.count += 1
                buckets[key] = existing
            } else {
                buckets[key] = AgentEventSummaryChip(
                    label: presentation.label,
                    emphasis: presentation.emphasis,
                    count: 1
                )
            }
        }

        return buckets.values.sorted {
            let lhsPriority = sortPriority($0.emphasis)
            let rhsPriority = sortPriority($1.emphasis)
            if lhsPriority == rhsPriority {
                if $0.count == $1.count {
                    return $0.label < $1.label
                }
                return $0.count > $1.count
            }
            return lhsPriority < rhsPriority
        }
    }

    private static func sortPriority(_ emphasis: AgentEventEmphasis) -> Int {
        switch emphasis {
        case .warning:
            return 0
        case .positive:
            return 1
        case .info:
            return 2
        case .neutral:
            return 3
        }
    }
}
