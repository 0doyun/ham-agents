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
}
