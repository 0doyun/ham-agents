import Foundation
import XCTest
@testable import HamCore

final class AgentEventPayloadTests: XCTestCase {

    func testAgentEventPayload_DecodesNewFields() throws {
        let json = """
        {
          "id": "evt-001",
          "agent_id": "agent-abc",
          "type": "agent.status_updated",
          "summary": "Status updated",
          "occurred_at": "2026-01-15T10:00:00Z",
          "session_id": "sess-xyz",
          "parent_agent_id": "parent-001",
          "task_name": "code-review",
          "task_desc": "Review the pull request for correctness",
          "artifact_type": "file",
          "artifact_ref": "src/main.go",
          "artifact_data": "package main\\n",
          "tool_name": "Bash",
          "tool_input": "go build ./...",
          "tool_type": "bash",
          "tool_duration_ms": 1234
        }
        """

        let event = try DaemonJSONDecoder.make().decode(AgentEventPayload.self, from: Data(json.utf8))

        XCTAssertEqual(event.id, "evt-001")
        XCTAssertEqual(event.agentID, "agent-abc")
        XCTAssertEqual(event.sessionID, "sess-xyz")
        XCTAssertEqual(event.parentAgentID, "parent-001")
        XCTAssertEqual(event.taskName, "code-review")
        XCTAssertEqual(event.taskDesc, "Review the pull request for correctness")
        XCTAssertEqual(event.artifactType, "file")
        XCTAssertEqual(event.artifactRef, "src/main.go")
        XCTAssertEqual(event.artifactData, "package main\n")
        XCTAssertEqual(event.toolName, "Bash")
        XCTAssertEqual(event.toolInput, "go build ./...")
        XCTAssertEqual(event.toolType, "bash")
        XCTAssertEqual(event.toolDurationMs, 1234)
    }

    func testAgentEventPayload_LegacyJSON_NewFieldsNil() throws {
        let json = """
        {
          "id": "evt-legacy",
          "agent_id": "agent-old",
          "type": "agent.registered",
          "summary": "Old agent registered",
          "occurred_at": "2026-01-01T00:00:00Z",
          "presentation_label": "Registered",
          "lifecycle_status": "booting",
          "lifecycle_confidence": 1.0
        }
        """

        let event = try DaemonJSONDecoder.make().decode(AgentEventPayload.self, from: Data(json.utf8))

        // Legacy fields decode correctly
        XCTAssertEqual(event.id, "evt-legacy")
        XCTAssertEqual(event.agentID, "agent-old")
        XCTAssertEqual(event.presentationLabel, "Registered")
        XCTAssertEqual(event.lifecycleStatus, "booting")
        XCTAssertEqual(event.lifecycleConfidence, 1.0)

        // New fields are nil
        XCTAssertNil(event.sessionID)
        XCTAssertNil(event.parentAgentID)
        XCTAssertNil(event.taskName)
        XCTAssertNil(event.taskDesc)
        XCTAssertNil(event.artifactType)
        XCTAssertNil(event.artifactRef)
        XCTAssertNil(event.artifactData)
        XCTAssertNil(event.toolName)
        XCTAssertNil(event.toolInput)
        XCTAssertNil(event.toolType)
        XCTAssertNil(event.toolDurationMs)
    }
}
