import Foundation
import HamCore

public protocol QuickMessageSending: Sendable {
    func send(message: String, to agent: Agent)
}

public struct NoopQuickMessageSender: QuickMessageSending {
    public init() {}

    public func send(message: String, to agent: Agent) {
        _ = message
        _ = agent
    }
}
