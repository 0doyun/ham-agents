import Foundation

public extension AgentStatus {
    var humanizedLabel: String {
        switch self {
        case .waitingInput:
            return "needs input"
        case .runningTool:
            return "running tool"
        default:
            return rawValue.replacingOccurrences(of: "_", with: " ")
        }
    }

    var isRunningActivity: Bool {
        switch self {
        case .booting, .thinking, .reading, .runningTool:
            return true
        default:
            return false
        }
    }

    /// Broader active-work check that includes all non-idle, non-terminal statuses.
    /// Use this for status bar tinting where any active work should show as busy.
    var isActiveWork: Bool {
        switch self {
        case .idle, .done, .error, .disconnected, .sleeping, .waitingInput:
            return false
        default:
            return true
        }
    }
}
