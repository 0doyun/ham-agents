import Foundation
@preconcurrency import UserNotifications

public protocol UserNotificationCentering: Sendable {
    func requestAuthorization(options: UNAuthorizationOptions) async throws -> Bool
    func add(_ request: UNNotificationRequest) async throws
    func authorizationStatus() async -> NotificationPermissionStatus
    func setNotificationCategories(_ categories: Set<UNNotificationCategory>)
    func setDelegate(_ delegate: UNUserNotificationCenterDelegate?)
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

    public init(center: UNUserNotificationCenter) {
        self.center = center
    }

    /// Returns a live center when running inside an app bundle, nil otherwise.
    public static func makeIfAvailable() -> LiveUserNotificationCenter? {
        guard Bundle.main.bundleIdentifier != nil else { return nil }
        return LiveUserNotificationCenter(center: .current())
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

    public func setNotificationCategories(_ categories: Set<UNNotificationCategory>) {
        center.setNotificationCategories(categories)
    }

    public func setDelegate(_ delegate: UNUserNotificationCenterDelegate?) {
        center.delegate = delegate
    }
}

/// Silent no-op center for environments without an app bundle.
public struct NoopUserNotificationCenter: UserNotificationCentering {
    public init() {}
    public func requestAuthorization(options: UNAuthorizationOptions) async throws -> Bool { false }
    public func add(_ request: UNNotificationRequest) async throws {}
    public func authorizationStatus() async -> NotificationPermissionStatus { .notDetermined }
    public func setNotificationCategories(_ categories: Set<UNNotificationCategory>) { _ = categories }
    public func setDelegate(_ delegate: UNUserNotificationCenterDelegate?) { _ = delegate }
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

public final class UserNotificationSink: NSObject, NotificationSink, NotificationPermissionControlling, UNUserNotificationCenterDelegate, @unchecked Sendable {
    private let center: UserNotificationCentering
    private let authorizationState = AuthorizationState()
    private var interactionHandler: (@Sendable (NotificationInteraction) -> Void)?

    private static let attentionCategoryIdentifier = "ham.agent.attention"
    private static let openTerminalActionIdentifier = "ham.open_terminal"
    private static let dismissActionIdentifier = "ham.dismiss"

    public init(center: UserNotificationCentering? = nil, interactionHandler: (@Sendable (NotificationInteraction) -> Void)? = nil) {
        self.center = center
            ?? LiveUserNotificationCenter.makeIfAvailable()
            ?? NoopUserNotificationCenter()
        self.interactionHandler = interactionHandler
        super.init()
        self.center.setNotificationCategories(Self.notificationCategories)
        self.center.setDelegate(self)
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

    public func setInteractionHandler(_ handler: (@Sendable (NotificationInteraction) -> Void)?) {
        interactionHandler = handler
    }

    private func makeRequest(for candidate: NotificationCandidate) -> UNNotificationRequest {
        let content = UNMutableNotificationContent()
        content.title = candidate.title
        content.body = candidate.body
        content.sound = .default
        if candidate.supportsAttentionActions {
            content.categoryIdentifier = Self.attentionCategoryIdentifier
        }
        if let agentID = candidate.agentID {
            content.userInfo = ["agent_id": agentID]
        }

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
        case .silence(let agent):
            return "\(agent.id).silence"
        case .heartbeat(let agent, _):
            return "\(agent.id).heartbeat"
        case .teamDigest(let teamName):
            return "\(teamName).team_digest"
        }
    }

    private static var notificationCategories: Set<UNNotificationCategory> {
        let openTerminal = UNNotificationAction(
            identifier: openTerminalActionIdentifier,
            title: "Open Terminal",
            options: [.foreground]
        )
        let dismiss = UNNotificationAction(
            identifier: dismissActionIdentifier,
            title: "Dismiss",
            options: []
        )
        return [
            UNNotificationCategory(
                identifier: attentionCategoryIdentifier,
                actions: [openTerminal, dismiss],
                intentIdentifiers: [],
                options: []
            )
        ]
    }
    public func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse) async {
        _ = center
        handleResponse(actionIdentifier: response.actionIdentifier, userInfo: response.notification.request.content.userInfo)
    }

    func handleResponse(actionIdentifier: String, userInfo: [AnyHashable: Any]) {
        guard let handler = interactionHandler else { return }
        let agentID = userInfo["agent_id"] as? String
        switch actionIdentifier {
        case UNNotificationDefaultActionIdentifier:
            if let agentID {
                handler(.focusAgent(agentID))
            }
        case Self.openTerminalActionIdentifier:
            if let agentID {
                handler(.openTerminal(agentID))
            }
        case Self.dismissActionIdentifier, UNNotificationDismissActionIdentifier:
            handler(.dismiss(agentID))
        default:
            break
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
