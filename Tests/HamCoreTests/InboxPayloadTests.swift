import XCTest
@testable import HamCore

final class InboxPayloadTests: XCTestCase {
    private var decoder: JSONDecoder { DaemonJSONDecoder.make() }

    func testInboxItemPayload_DecodesFromJSON() throws {
        let json = """
        {
            "id": "item-1",
            "agent_id": "a1",
            "agent_name": "alice",
            "type": "permission_request",
            "summary": "Tool Bash requested permission",
            "tool_name": "Bash",
            "occurred_at": "2026-04-07T12:00:00Z",
            "read": false,
            "actionable": true
        }
        """.data(using: .utf8)!

        let item = try decoder.decode(InboxItemPayload.self, from: json)
        XCTAssertEqual(item.id, "item-1")
        XCTAssertEqual(item.agentID, "a1")
        XCTAssertEqual(item.agentName, "alice")
        XCTAssertEqual(item.type, "permission_request")
        XCTAssertEqual(item.toolName, "Bash")
        XCTAssertFalse(item.read)
        XCTAssertTrue(item.actionable)
    }

    func testInboxItemPayload_OmitOptionalToolName() throws {
        let json = """
        {"id": "item-2", "agent_id": "a1", "agent_name": "alice", "type": "notification", "summary": "hi", "occurred_at": "2026-04-07T12:00:00Z", "read": false, "actionable": false}
        """.data(using: .utf8)!
        let item = try decoder.decode(InboxItemPayload.self, from: json)
        XCTAssertNil(item.toolName)
    }
}
