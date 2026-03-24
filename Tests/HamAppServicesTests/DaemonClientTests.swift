import Foundation
import XCTest
@testable import HamAppServices
@testable import HamCore

final class DaemonClientTests: XCTestCase {
    func testFollowEventsUsesFollowCommandAndCursor() async throws {
        let transport = RecordingTransport(
            response: DaemonResponse(
                events: [
                    AgentEventPayload(
                        id: "event-2",
                        agentID: "agent-1",
                        type: "agent.registered",
                        summary: "Registered",
                        occurredAt: Date(timeIntervalSince1970: 2)
                    )
                ]
            )
        )
        let client = HamDaemonClient(transport: transport)

        let events = try await client.followEvents(afterEventID: "event-1", limit: 10, waitMilliseconds: 1500)

        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(transport.requests.first?.command, .followEvents)
        XCTAssertEqual(transport.requests.first?.afterEventID, "event-1")
        XCTAssertEqual(transport.requests.first?.waitMillis, 1500)
    }
}

private final class RecordingTransport: DaemonTransport, @unchecked Sendable {
    let response: DaemonResponse
    private(set) var requests: [DaemonRequest] = []

    init(response: DaemonResponse) {
        self.response = response
    }

    func send(_ request: DaemonRequest) async throws -> DaemonResponse {
        requests.append(request)
        return response
    }
}
