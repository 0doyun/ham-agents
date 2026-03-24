import Foundation
import HamCore

public protocol SessionOpening: Sendable {
    func openSession(for agent: Agent)
}

public struct NoopSessionOpener: SessionOpening {
    public init() {}

    public func openSession(for agent: Agent) {
        _ = agent
    }
}
