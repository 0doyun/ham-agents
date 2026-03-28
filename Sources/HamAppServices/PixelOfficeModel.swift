import HamCore

public enum OfficeArea: String, CaseIterable, Sendable {
    case desk
    case bookshelf
    case sofa
    case alertLight
}

public enum HamsterSpriteState: String, CaseIterable, Sendable {
    case idle
    case walk
    case run
    case type
    case read
    case think
    case sleep
    case celebrate
    case alert
    case error
}

public struct PixelOfficeOccupant: Equatable, Identifiable, Sendable {
    public let agent: Agent
    public let area: OfficeArea
    public let sprite: HamsterSpriteState
    public let subAgentCount: Int

    public var id: String { agent.id }

    public init(agent: Agent, area: OfficeArea, sprite: HamsterSpriteState, subAgentCount: Int = 0) {
        self.agent = agent
        self.area = area
        self.sprite = sprite
        self.subAgentCount = subAgentCount
    }
}

/// Icon overlay shown above a hamster sprite to communicate status at a glance.
public enum StatusIcon: String, Sendable {
    case question  // ❓ waiting_input
    case warning   // ⚠️ error / disconnected
    case check     // ✅ done
}

public enum MenuBarHamsterState: Equatable, Sendable {
    case idle
    case running
    case waiting
    case error
    case done
}

public enum PixelOfficeMapper {
    public static func occupant(for agent: Agent) -> PixelOfficeOccupant {
        PixelOfficeOccupant(
            agent: agent,
            area: area(for: agent.status),
            sprite: sprite(for: agent.status),
            subAgentCount: agent.subAgentCount
        )
    }

    public static func occupants(from agents: [Agent]) -> [PixelOfficeOccupant] {
        agents.map(occupant(for:))
    }

    public static func area(for status: AgentStatus) -> OfficeArea {
        switch status {
        case .booting, .thinking, .runningTool:
            return .desk
        case .reading:
            return .bookshelf
        case .error, .waitingInput, .disconnected:
            return .alertLight
        case .idle, .sleeping, .done:
            return .sofa
        }
    }

    public static func sprite(for status: AgentStatus) -> HamsterSpriteState {
        switch status {
        case .booting:
            return .walk
        case .thinking:
            return .think
        case .runningTool:
            return .type
        case .reading:
            return .read
        case .waitingInput:
            return .alert
        case .error, .disconnected:
            return .error
        case .done:
            return .celebrate
        case .sleeping:
            return .sleep
        case .idle:
            return .idle
        }
    }

    public static func statusIcon(for status: AgentStatus) -> StatusIcon? {
        switch status {
        case .waitingInput:
            return .question
        case .error, .disconnected:
            return .warning
        case .done:
            return .check
        default:
            return nil
        }
    }

    public static func menuBarState(summary: HamMenuBarSummary?, agents: [Agent]) -> MenuBarHamsterState {
        let sourceAgents = summary == nil ? agents : agents
        if sourceAgents.contains(where: { $0.status == .error || $0.status == .disconnected }) {
            return .error
        }
        if sourceAgents.contains(where: { $0.status == .waitingInput }) {
            return .waiting
        }
        if sourceAgents.contains(where: { $0.status.isRunningActivity }) {
            return .running
        }
        if !sourceAgents.isEmpty && sourceAgents.allSatisfy({ $0.status == .done }) {
            return .done
        }
        return .idle
    }
}
