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

        let requests = await center.requests
        let authorizationCount = await center.authorizationRequestCount
        let status = await center.status
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

        let authorizationCount = await center.authorizationRequestCount
        let requests = await center.requests
        let status = await center.status
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

        let requests = await center.requests
        XCTAssertEqual(requests.first?.identifier, "frontend.team_digest")
    }

    private func makeAgent() -> Agent {
        Agent(
            id: "agent-1",
            displayName: "builder",
            provider: "claude",
            host: "localhost",
            mode: .managed,
            projectPath: "/tmp/app",
            status: .done,
            statusConfidence: 1,
            lastEventAt: Date(timeIntervalSince1970: 1)
        )
    }
}

private actor RecordingUserNotificationCenter: UserNotificationCentering {
    let granted: Bool
    private(set) var status: NotificationPermissionStatus
    private(set) var authorizationRequestCount = 0
    private(set) var requests: [UNNotificationRequest] = []

    init(granted: Bool, initialStatus: NotificationPermissionStatus = .notDetermined) {
        self.granted = granted
        self.status = initialStatus
    }

    func requestAuthorization(options: UNAuthorizationOptions) async throws -> Bool {
        _ = options
        authorizationRequestCount += 1
        status = granted ? .authorized : .denied
        return granted
    }

    func add(_ request: UNNotificationRequest) async throws {
        requests.append(request)
    }

    func authorizationStatus() async -> NotificationPermissionStatus {
        status
    }
}
