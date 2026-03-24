import Foundation
import HamCore

public protocol AgentStore {
    func save(_ agent: Agent)
    func allAgents() -> [Agent]
}

public final class InMemoryAgentStore: AgentStore {
    private var agents: [String: Agent] = [:]

    public init() {}

    public func save(_ agent: Agent) {
        agents[agent.id] = agent
    }

    public func allAgents() -> [Agent] {
        agents.values.sorted { $0.displayName.localizedCaseInsensitiveCompare($1.displayName) == .orderedAscending }
    }
}
