import Foundation
import HamCore
import HamPersistence

public struct RuntimeSnapshot: Equatable, Sendable {
    public var agents: [Agent]

    public init(agents: [Agent]) {
        self.agents = agents
    }

    public var totalCount: Int {
        agents.count
    }

    public var runningCount: Int {
        agents.filter { [.booting, .thinking, .reading, .runningTool].contains($0.status) }.count
    }

    public var waitingCount: Int {
        agents.filter { $0.status == .waitingInput }.count
    }

    public var doneCount: Int {
        agents.filter { $0.status == .done }.count
    }
}

public final class RuntimeRegistry {
    private let store: AgentStore

    public init(store: AgentStore) {
        self.store = store
    }

    public func register(_ agent: Agent) {
        store.save(agent)
    }

    public func snapshot() -> RuntimeSnapshot {
        RuntimeSnapshot(agents: store.allAgents())
    }
}
