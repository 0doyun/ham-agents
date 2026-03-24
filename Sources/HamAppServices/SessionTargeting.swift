import Foundation
import HamCore

public enum SessionTarget: Equatable, Sendable {
    case externalURL(URL)
    case workspace(path: String)
}

public struct SessionTargetPlanner: Sendable {
    public init() {}

    public func target(for agent: Agent) -> SessionTarget {
        if let sessionRef = agent.sessionRef,
           let url = URL(string: sessionRef),
           url.scheme != nil {
            return .externalURL(url)
        }

        return .workspace(path: agent.projectPath)
    }
}
