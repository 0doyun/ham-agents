import Foundation
import XCTest
@testable import HamCore

final class DaemonPayloadDecodingTests: XCTestCase {
    func testAgentDecodesFromGoDaemonJSON() throws {
        let payload = """
        {
          "id": "managed-123",
          "display_name": "reviewer",
          "provider": "claude",
          "host": "MacBook-Pro.local",
          "mode": "managed",
          "project_path": "/tmp/demo",
          "role": "reviewer",
          "status": "booting",
          "status_confidence": 1,
          "last_event_at": "2026-03-24T14:02:18.002914Z",
          "last_user_visible_summary": "Managed session registered.",
          "notification_policy": "default",
          "session_title": "Claude Review",
          "session_is_active": true,
          "avatar_variant": "default"
        }
        """

        let agent = try DaemonJSONDecoder.make().decode(Agent.self, from: Data(payload.utf8))

        XCTAssertEqual(agent.id, "managed-123")
        XCTAssertEqual(agent.displayName, "reviewer")
        XCTAssertEqual(agent.projectPath, "/tmp/demo")
        XCTAssertEqual(agent.status, .booting)
        XCTAssertEqual(agent.notificationPolicy, .default)
        XCTAssertEqual(agent.sessionTitle, "Claude Review")
        XCTAssertTrue(agent.sessionIsActive)
    }

    func testDaemonStatusPayloadDecodesFromGoStatusJSON() throws {
        let payload = """
        {
          "done": 0,
          "generatedAt": "2026-03-24T14:02:18.139024Z",
          "running": 1,
          "total": 1,
          "waiting": 0
        }
        """

        let status = try DaemonJSONDecoder.make().decode(DaemonStatusPayload.self, from: Data(payload.utf8))

        XCTAssertEqual(status.total, 1)
        XCTAssertEqual(status.running, 1)
        XCTAssertEqual(status.waiting, 0)
    }

    func testAgentEventPayloadDecodesFromGoEventsJSON() throws {
        let payload = """
        [
          {
            "id": "event-123",
            "agent_id": "managed-123",
            "type": "agent.registered",
            "summary": "Managed session registered.",
            "occurred_at": "2026-03-24T14:02:18.002914Z"
          }
        ]
        """

        let events = try DaemonJSONDecoder.make().decode([AgentEventPayload].self, from: Data(payload.utf8))

        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].agentID, "managed-123")
        XCTAssertEqual(events[0].type, "agent.registered")
    }
}
