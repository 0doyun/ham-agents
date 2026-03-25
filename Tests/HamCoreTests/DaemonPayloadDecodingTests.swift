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
          "status_reason": "Managed launch requested.",
          "last_event_at": "2026-03-24T14:02:18.002914Z",
          "last_user_visible_summary": "Managed session registered.",
          "notification_policy": "default",
          "session_title": "Claude Review",
          "session_is_active": true,
          "session_tty": "ttys001",
          "session_working_directory": "/tmp/demo",
          "session_activity": "claude",
          "session_process_id": 12345,
          "session_command": "/usr/local/bin/claude",
          "avatar_variant": "default"
        }
        """

        let agent = try DaemonJSONDecoder.make().decode(Agent.self, from: Data(payload.utf8))

        XCTAssertEqual(agent.id, "managed-123")
        XCTAssertEqual(agent.displayName, "reviewer")
        XCTAssertEqual(agent.projectPath, "/tmp/demo")
        XCTAssertEqual(agent.status, .booting)
        XCTAssertEqual(agent.statusReason, "Managed launch requested.")
        XCTAssertEqual(agent.notificationPolicy, .default)
        XCTAssertEqual(agent.sessionTitle, "Claude Review")
        XCTAssertTrue(agent.sessionIsActive)
        XCTAssertEqual(agent.sessionTTY, "ttys001")
        XCTAssertEqual(agent.sessionWorkingDirectory, "/tmp/demo")
        XCTAssertEqual(agent.sessionActivity, "claude")
        XCTAssertEqual(agent.sessionProcessID, 12345)
        XCTAssertEqual(agent.sessionCommand, "/usr/local/bin/claude")
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

    func testDaemonRuntimeSnapshotDecodesAttentionSummaryFromGoJSON() throws {
        let payload = """
        {
          "agents": [],
          "generated_at": "2026-03-24T14:02:18.139024Z",
          "attention_count": 2,
          "attention_breakdown": {
            "error": 1,
            "waiting_input": 1,
            "disconnected": 0
          },
          "attention_order": ["agent-2", "agent-1"],
          "attention_subtitles": {
            "agent-2": "error · high confidence · Build failed."
          }
        }
        """

        let snapshot = try DaemonJSONDecoder.make().decode(DaemonRuntimeSnapshotPayload.self, from: Data(payload.utf8))

        XCTAssertEqual(snapshot.attentionCount, 2)
        XCTAssertEqual(snapshot.attentionBreakdown.error, 1)
        XCTAssertEqual(snapshot.attentionBreakdown.waitingInput, 1)
        XCTAssertEqual(snapshot.attentionBreakdown.disconnected, 0)
        XCTAssertEqual(snapshot.attentionOrder, ["agent-2", "agent-1"])
        XCTAssertEqual(snapshot.attentionSubtitles["agent-2"], "error · high confidence · Build failed.")
    }

    func testDaemonRuntimeSnapshotDefaultsMissingAttentionSummaryFields() throws {
        let payload = """
        {
          "agents": [],
          "generated_at": "2026-03-24T14:02:18.139024Z"
        }
        """

        let snapshot = try DaemonJSONDecoder.make().decode(DaemonRuntimeSnapshotPayload.self, from: Data(payload.utf8))

        XCTAssertEqual(snapshot.attentionCount, 0)
        XCTAssertEqual(snapshot.attentionBreakdown.error, 0)
        XCTAssertEqual(snapshot.attentionBreakdown.waitingInput, 0)
        XCTAssertEqual(snapshot.attentionBreakdown.disconnected, 0)
        XCTAssertEqual(snapshot.attentionOrder, [])
        XCTAssertEqual(snapshot.attentionSubtitles, [:])
    }

    func testDaemonSettingsPayloadDecodesSilenceNotificationFlag() throws {
        let payload = """
        {
          "notifications": {
            "done": true,
            "error": true,
            "waiting_input": true,
            "silence": true,
            "quiet_hours_enabled": false,
            "quiet_hours_start_hour": 22,
            "quiet_hours_end_hour": 8,
            "preview_text": false
          },
          "appearance": {
            "theme": "auto"
          },
          "integrations": {
            "iterm_enabled": true
          }
        }
        """

        let settings = try DaemonJSONDecoder.make().decode(DaemonSettingsPayload.self, from: Data(payload.utf8))

        XCTAssertTrue(settings.notifications.silence)
    }

    func testDaemonSettingsPayloadDefaultsMissingSilenceNotificationFlag() throws {
        let payload = """
        {
          "notifications": {
            "done": true,
            "error": true,
            "waiting_input": true,
            "quiet_hours_enabled": false,
            "quiet_hours_start_hour": 22,
            "quiet_hours_end_hour": 8,
            "preview_text": false
          },
          "appearance": {
            "theme": "auto"
          },
          "integrations": {
            "iterm_enabled": true
          }
        }
        """

        let settings = try DaemonJSONDecoder.make().decode(DaemonSettingsPayload.self, from: Data(payload.utf8))

        XCTAssertFalse(settings.notifications.silence)
    }

    func testAgentEventPayloadDecodesFromGoEventsJSON() throws {
        let payload = """
        [
          {
            "id": "event-123",
            "agent_id": "managed-123",
            "type": "agent.registered",
            "summary": "Managed session registered.",
            "occurred_at": "2026-03-24T14:02:18.002914Z",
            "presentation_label": "Managed",
            "presentation_emphasis": "info",
            "presentation_summary": "Managed session registered.",
            "lifecycle_status": "booting",
            "lifecycle_mode": "managed",
            "lifecycle_reason": "Managed launch requested.",
            "lifecycle_confidence": 1
          }
        ]
        """

        let events = try DaemonJSONDecoder.make().decode([AgentEventPayload].self, from: Data(payload.utf8))

        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].agentID, "managed-123")
        XCTAssertEqual(events[0].type, "agent.registered")
        XCTAssertEqual(events[0].presentationLabel, "Managed")
        XCTAssertEqual(events[0].presentationEmphasis, "info")
        XCTAssertEqual(events[0].presentationSummary, "Managed session registered.")
        XCTAssertEqual(events[0].lifecycleStatus, "booting")
        XCTAssertEqual(events[0].lifecycleMode, "managed")
        XCTAssertEqual(events[0].lifecycleReason, "Managed launch requested.")
        XCTAssertEqual(events[0].lifecycleConfidence, 1)
    }
}
