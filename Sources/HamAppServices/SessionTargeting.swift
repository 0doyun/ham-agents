import Foundation
import HamCore

public enum SessionTarget: Equatable, Sendable {
    case itermSession(id: String, url: URL)
    case externalURL(URL)
    case workspace(path: String)
}

public struct SessionTargetPlanner: Sendable {
    public init() {}

    public func target(for agent: Agent) -> SessionTarget {
        if let sessionRef = agent.sessionRef,
           let url = URL(string: sessionRef),
           url.scheme != nil {
            if url.scheme == "iterm2",
               url.host == "session" {
                let sessionID = url.path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
                if !sessionID.isEmpty {
                    return .itermSession(id: sessionID, url: url)
                }
            }
            return .externalURL(url)
        }

        return .workspace(path: agent.projectPath)
    }
}
