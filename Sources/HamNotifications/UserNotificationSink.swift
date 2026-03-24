import Foundation
@preconcurrency import UserNotifications

public protocol UserNotificationCentering: Sendable {
    func requestAuthorization(options: UNAuthorizationOptions) async throws -> Bool
    func add(_ request: UNNotificationRequest) async throws
}

public final class LiveUserNotificationCenter: UserNotificationCentering, @unchecked Sendable {
    private let center: UNUserNotificationCenter

    public init(center: UNUserNotificationCenter = .current()) {
        self.center = center
    }

    public func requestAuthorization(options: UNAuthorizationOptions) async throws -> Bool {
        try await center.requestAuthorization(options: options)
    }

    public func add(_ request: UNNotificationRequest) async throws {
        try await center.add(request)
    }
}

public final class UserNotificationSink: NotificationSink, @unchecked Sendable {
    private let center: UserNotificationCentering
    private let authorizationState = AuthorizationState()

    public init(center: UserNotificationCentering = LiveUserNotificationCenter()) {
        self.center = center
    }

    public func send(_ candidate: NotificationCandidate) {
        Task {
            do {
                guard try await ensureAuthorization() else { return }
                try await center.add(makeRequest(for: candidate))
            } catch {
                return
            }
        }
    }

    private func ensureAuthorization() async throws -> Bool {
        try await authorizationState.ensureAuthorization(center: center)
    }

    private func makeRequest(for candidate: NotificationCandidate) -> UNNotificationRequest {
        let content = UNMutableNotificationContent()
        content.title = candidate.title
        content.body = candidate.body
        content.sound = .default

        return UNNotificationRequest(
            identifier: identifier(for: candidate),
            content: content,
            trigger: nil
        )
    }

    private func identifier(for candidate: NotificationCandidate) -> String {
        switch candidate.event {
        case .done(let agent):
            return "\(agent.id).done"
        case .waitingInput(let agent):
            return "\(agent.id).waiting_input"
        case .error(let agent):
            return "\(agent.id).error"
        }
    }
}

private actor AuthorizationState {
    private var authorizationResolved = false
    private var authorizationGranted = false

    func ensureAuthorization(center: UserNotificationCentering) async throws -> Bool {
        if authorizationResolved {
            return authorizationGranted
        }

        let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound])
        authorizationResolved = true
        authorizationGranted = granted
        return granted
    }
}
