import HamCore

public enum PixelOfficeZone: String, CaseIterable, Sendable {
    case desk
    case library
    case kitchen
    case alertCorner
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
    public let zone: PixelOfficeZone
    public let sprite: HamsterSpriteState

    public var id: String { agent.id }

    public init(agent: Agent, zone: PixelOfficeZone, sprite: HamsterSpriteState) {
        self.agent = agent
        self.zone = zone
        self.sprite = sprite
    }
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
        PixelOfficeOccupant(agent: agent, zone: zone(for: agent.status), sprite: sprite(for: agent.status))
    }

    public static func occupants(from agents: [Agent]) -> [PixelOfficeOccupant] {
        agents.map(occupant(for:))
    }

    public static func zone(for status: AgentStatus) -> PixelOfficeZone {
        switch status {
        case .booting, .thinking, .runningTool:
            return .desk
        case .reading:
            return .library
        case .error, .waitingInput, .disconnected:
            return .alertCorner
        case .idle, .sleeping, .done:
            return .kitchen
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
