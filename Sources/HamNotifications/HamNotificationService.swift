import Foundation
import HamCore

public enum NotificationEvent: Equatable, Sendable {
    case done(Agent)
    case waitingInput(Agent)
    case error(Agent)
}

public struct HamNotificationService {
    public init() {}

    public func shouldNotify(for event: NotificationEvent) -> Bool {
        switch event {
        case .done(let agent), .waitingInput(let agent), .error(let agent):
            return agent.notificationPolicy != .muted
        }
    }
}
