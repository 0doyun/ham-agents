import Foundation
import HamCore

public enum SessionTarget: Equatable, Sendable {
    case itermSession(id: String, url: URL)
    case tmuxPane(target: String, sessionName: String, windowIndex: Int, paneIndex: Int)
    case externalURL(URL)
    case workspace(path: String)
}

public struct SessionTargetPlanner: Sendable {
    public init() {}

    public func target(for agent: Agent) -> SessionTarget {
        if let sessionRef = agent.sessionRef {
            if let tmuxTarget = tmuxTarget(for: sessionRef) {
                return tmuxTarget
            }
            if let url = URL(string: sessionRef), url.scheme != nil {
                if url.scheme == "iterm2",
                   url.host == "session" {
                    let sessionID = url.path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
                    if !sessionID.isEmpty {
                        return .itermSession(id: sessionID, url: url)
                    }
                }
                return .externalURL(url)
            }
        }

        return .workspace(path: agent.projectPath)
    }

    private func tmuxTarget(for sessionRef: String) -> SessionTarget? {
        let prefix = "tmux://"
        guard sessionRef.hasPrefix(prefix) else { return nil }
        let target = String(sessionRef.dropFirst(prefix.count))
        guard let dot = target.lastIndex(of: "."),
              let colon = target[..<dot].lastIndex(of: ":"),
              let windowIndex = Int(target[target.index(after: colon)..<dot]),
              let paneIndex = Int(target[target.index(after: dot)...]) else {
            return nil
        }

        let sessionName = String(target[..<colon])
        guard !sessionName.isEmpty else { return nil }
        return .tmuxPane(
            target: target,
            sessionName: sessionName,
            windowIndex: windowIndex,
            paneIndex: paneIndex
        )
    }
}
