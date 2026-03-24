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
}
