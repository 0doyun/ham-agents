import Foundation
import HamCore

public protocol TerminalAdapter: Sendable {
    func canFocus(agent: Agent) -> Bool
    func focus(agent: Agent) throws
}

public enum AdapterError: Error {
    case unsupported
}

public struct Iterm2Adapter: TerminalAdapter {
    public init() {}

    public func canFocus(agent: Agent) -> Bool {
        agent.mode == .managed || agent.mode == .attached
    }

    public func focus(agent: Agent) throws {
        guard canFocus(agent: agent) else {
            throw AdapterError.unsupported
        }
    }
}
