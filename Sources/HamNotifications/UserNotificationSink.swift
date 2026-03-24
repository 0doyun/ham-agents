import Foundation
@preconcurrency import UserNotifications

public protocol UserNotificationCentering: Sendable {
    func requestAuthorization(options: UNAuthorizationOptions) async throws -> Bool
    func add(_ request: UNNotificationRequest) async throws
    func authorizationStatus() async -> NotificationPermissionStatus
}

public enum NotificationPermissionStatus: String, Equatable, Sendable {
    case notDetermined
    case authorized
    case denied
}

public protocol NotificationPermissionControlling: Sendable {
    func currentPermissionStatus() async -> NotificationPermissionStatus
    func requestPermission() async -> NotificationPermissionStatus
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

    public func authorizationStatus() async -> NotificationPermissionStatus {
        await withCheckedContinuation { continuation in
            center.getNotificationSettings { settings in
                let status: NotificationPermissionStatus
                switch settings.authorizationStatus {
                case .authorized, .ephemeral, .provisional:
                    status = .authorized
                case .denied:
                    status = .denied
                case .notDetermined:
                    status = .notDetermined
                @unknown default:
                    status = .notDetermined
                }
                continuation.resume(returning: status)
            }
        }
    }
}

public struct NoopNotificationPermissionController: NotificationPermissionControlling {
    public init() {}

    public func currentPermissionStatus() async -> NotificationPermissionStatus {
        .notDetermined
    }

    public func requestPermission() async -> NotificationPermissionStatus {
        .notDetermined
    }
}

public final class UserNotificationSink: NotificationSink, NotificationPermissionControlling, @unchecked Sendable {
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

    public func currentPermissionStatus() async -> NotificationPermissionStatus {
        await center.authorizationStatus()
    }

    public func requestPermission() async -> NotificationPermissionStatus {
        do {
            let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound])
            await authorizationState.set(granted: granted)
            return granted ? .authorized : .denied
        } catch {
            return await center.authorizationStatus()
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

    func set(granted: Bool) {
        authorizationResolved = true
        authorizationGranted = granted
    }
}
