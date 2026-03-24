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
        XCTAssertEqual(authorizationCount, 1)
        XCTAssertEqual(requests.count, 1)
        XCTAssertEqual(requests.first?.identifier, "agent-1.done")
        XCTAssertEqual(requests.first?.content.title, "builder finished")
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
        XCTAssertEqual(authorizationCount, 1)
        XCTAssertTrue(requests.isEmpty)
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
    private(set) var authorizationRequestCount = 0
    private(set) var requests: [UNNotificationRequest] = []

    init(granted: Bool) {
        self.granted = granted
    }

    func requestAuthorization(options: UNAuthorizationOptions) async throws -> Bool {
        _ = options
        authorizationRequestCount += 1
        return granted
    }

    func add(_ request: UNNotificationRequest) async throws {
        requests.append(request)
    }
}
