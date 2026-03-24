import Foundation
import HamCore

public protocol QuickMessageSending: Sendable {
    func send(message: String, to agent: Agent)
}

public enum QuickMessagePlan: Equatable, Sendable {
    case terminalWrite(target: SessionTarget, message: String)
    case clipboardHandoff(message: String)
}

public struct QuickMessagePlanner: Sendable {
    private let sessionTargetPlanner: SessionTargetPlanner

    public init(sessionTargetPlanner: SessionTargetPlanner = SessionTargetPlanner()) {
        self.sessionTargetPlanner = sessionTargetPlanner
    }

    public func plan(message: String, for agent: Agent, supportsTerminalAutomation: Bool) -> QuickMessagePlan {
        if supportsTerminalAutomation {
            return .terminalWrite(target: sessionTargetPlanner.target(for: agent), message: message)
        }

        return .clipboardHandoff(message: message)
    }
}

public struct NoopQuickMessageSender: QuickMessageSending {
    public init() {}

    public func send(message: String, to agent: Agent) {
        _ = message
        _ = agent
    }
}
