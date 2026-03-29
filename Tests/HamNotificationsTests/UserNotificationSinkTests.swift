import Foundation
import UserNotifications
import XCTest
@testable import HamCore
@testable import HamNotifications

final class UserNotificationSinkTests: XCTestCase {
    func testSendRequestsAuthorizationAndEnqueuesRequest() async throws {
        let center = RecordingUserNotificationCenter(granted: true)
        let sink = UserNotificationSink(center: center)

        sink.send(
            NotificationCandidate(
                event: .done(makeAgent()),
                title: "builder finished",
                body: "Build completed."
            )
        )

        try await Task.sleep(nanoseconds: 100_000_000)

        let requests = center.requests
        let authorizationCount = center.authorizationRequestCount
        let status = center.status
        XCTAssertEqual(authorizationCount, 1)
        XCTAssertEqual(requests.count, 1)
        XCTAssertEqual(requests.first?.identifier, "agent-1.done")
        XCTAssertEqual(requests.first?.content.title, "builder finished")
        XCTAssertEqual(status, .authorized)
    }

    func testSendSkipsAddWhenAuthorizationDenied() async throws {
        let center = RecordingUserNotificationCenter(granted: false)
        let sink = UserNotificationSink(center: center)

        sink.send(
            NotificationCandidate(
                event: .error(makeAgent()),
                title: "builder hit an error",
                body: "Build failed."
            )
        )

        try await Task.sleep(nanoseconds: 100_000_000)

        let authorizationCount = center.authorizationRequestCount
        let requests = center.requests
        let status = center.status
        XCTAssertEqual(authorizationCount, 1)
        XCTAssertTrue(requests.isEmpty)
        XCTAssertEqual(status, .denied)
    }

    func testCurrentPermissionStatusMirrorsCenterState() async {
        let center = RecordingUserNotificationCenter(granted: true, initialStatus: .denied)
        let sink = UserNotificationSink(center: center)

        let status = await sink.currentPermissionStatus()

        XCTAssertEqual(status, .denied)
    }

    func testTeamDigestUsesStableIdentifier() async throws {
        let center = RecordingUserNotificationCenter(granted: true)
        let sink = UserNotificationSink(center: center)

        sink.send(
            NotificationCandidate(
                event: .teamDigest("frontend"),
                title: "frontend needs attention",
                body: "1 needs input"
            )
        )

        try await Task.sleep(nanoseconds: 100_000_000)

        let requests = center.requests
        XCTAssertEqual(requests.first?.identifier, "frontend.team_digest")
    }

    func testAttentionNotificationUsesActionCategoryAndAgentPayload() async throws {
        let center = RecordingUserNotificationCenter(granted: true)
        let sink = UserNotificationSink(center: center)

        sink.send(
            NotificationCandidate(
                event: .waitingInput(makeAgent(status: .waitingInput)),
                title: "builder needs input",
                body: "Approve patch?"
            )
        )

        try await Task.sleep(nanoseconds: 100_000_000)

        let requests = center.requests
        XCTAssertEqual(requests.first?.content.categoryIdentifier, "ham.agent.attention")
        XCTAssertEqual(requests.first?.content.userInfo["agent_id"] as? String, "agent-1")
    }

    func testHandleResponseRoutesOpenTerminalInteraction() async {
        let center = RecordingUserNotificationCenter(granted: true)
        let expectation = expectation(description: "interaction")
        let box = InteractionBox()
        let sink = UserNotificationSink(center: center) { interaction in
            box.set(interaction)
            expectation.fulfill()
        }

        sink.handleResponse(
            actionIdentifier: "ham.open_terminal",
            userInfo: ["agent_id": "agent-1"]
        )

        await fulfillment(of: [expectation], timeout: 1.0)
        XCTAssertEqual(box.currentValue, .openTerminal("agent-1"))
    }

    private func makeAgent(status: AgentStatus = .done) -> Agent {
        Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: status,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
    }
}

private final class RecordingUserNotificationCenter: UserNotificationCentering, @unchecked Sendable {
    let granted: Bool
    private let queue = DispatchQueue(label: "RecordingUserNotificationCenter")
    private(set) var status: NotificationPermissionStatus
    private(set) var authorizationRequestCount = 0
    private(set) var requests: [UNNotificationRequest] = []

    init(granted: Bool, initialStatus: NotificationPermissionStatus = .notDetermined) {
        self.granted = granted
        self.status = initialStatus
    }

    func requestAuthorization(options: UNAuthorizationOptions) async throws -> Bool {
        _ = options
        return queue.sync {
            authorizationRequestCount += 1
            status = granted ? .authorized : .denied
            return granted
        }
    }

    func add(_ request: UNNotificationRequest) async throws {
        queue.sync {
            requests.append(request)
        }
    }

    func authorizationStatus() async -> NotificationPermissionStatus {
        queue.sync { status }
    }

    func setNotificationCategories(_ categories: Set<UNNotificationCategory>) {
        _ = categories
    }

    func setDelegate(_ delegate: UNUserNotificationCenterDelegate?) {
        _ = delegate
    }
}

private final class InteractionBox: @unchecked Sendable {
    private let lock = NSLock()
    private var value: NotificationInteraction?

    func set(_ interaction: NotificationInteraction) {
        lock.lock()
        defer { lock.unlock() }
        value = interaction
    }

    var currentValue: NotificationInteraction? {
        lock.lock()
        defer { lock.unlock() }
        return value
    }
}
