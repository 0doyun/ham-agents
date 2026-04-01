import Foundation
import HamCore

public protocol QuickMessageSending: Sendable {
    func send(message: String, to agent: Agent) -> QuickMessageResult
}

public enum QuickMessageResult: Equatable, Sendable {
    case delivered(String)
    case handoff(String)
    case failed(String)
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
            let target = sessionTargetPlanner.target(for: agent)
            switch target {
            case .itermSession, .tmuxPane:
                return .terminalWrite(target: target, message: message)
            case .externalURL, .workspace:
                break
            }
        }

        return .clipboardHandoff(message: message)
    }
}

public struct NoopQuickMessageSender: QuickMessageSending {
    public init() {}

    public func send(message: String, to agent: Agent) -> QuickMessageResult {
        _ = message
        _ = agent
        return .failed("Quick message sender is unavailable.")
    }
}
