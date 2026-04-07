import XCTest
@testable import HamCore

final class SessionGraphTests: XCTestCase {

    private var decoder: JSONDecoder { DaemonJSONDecoder.make() }

    // MARK: - Helpers

    /// Minimal JSON for an Agent that satisfies all non-optional CodingKeys.
    private func agentJSON(id: String, displayName: String, status: String) -> String {
        """
        {
            "id": "\(id)",
            "display_name": "\(displayName)",
            "provider": "claude",
            "mode": "managed",
            "project_path": "/tmp/project",
            "status": "\(status)"
        }
        """
    }

    // MARK: - Tests

    func testSessionGraph_DecodesFromJSON() throws {
        let json = """
        {
            "roots": [
                {
                    "agent": \(agentJSON(id: "a1", displayName: "alice", status: "running_tool")),
                    "children": [
                        {
                            "agent": \(agentJSON(id: "a2", displayName: "bob", status: "waiting_input")),
                            "children": [],
                            "block_reason": "waiting_input",
                            "depth": 1
                        }
                    ],
                    "block_reason": "",
                    "depth": 0
                }
            ],
            "total_count": 2,
            "blocked_count": 1,
            "generated_at": "2026-04-07T12:00:00Z"
        }
        """.data(using: .utf8)!

        let graph = try decoder.decode(SessionGraph.self, from: json)

        XCTAssertEqual(graph.totalCount, 2)
        XCTAssertEqual(graph.blockedCount, 1)
        XCTAssertEqual(graph.roots.count, 1)

        let root = graph.roots[0]
        XCTAssertEqual(root.agent.id, "a1")
        XCTAssertEqual(root.depth, 0)
        XCTAssertEqual(root.blockReason, "")
        XCTAssertEqual(root.children.count, 1)

        let child = root.children[0]
        XCTAssertEqual(child.agent.id, "a2")
        XCTAssertEqual(child.depth, 1)
        XCTAssertEqual(child.blockReason, "waiting_input")
        XCTAssertEqual(child.children.count, 0)
    }

    func testSessionGraph_EmptyChildren_OmittedField_DecodesOK() throws {
        // Go's omitempty may drop the "children" field entirely for leaf nodes.
        let json = """
        {
            "roots": [
                {
                    "agent": \(agentJSON(id: "a1", displayName: "alice", status: "idle")),
                    "block_reason": "",
                    "depth": 0
                }
            ],
            "total_count": 1,
            "blocked_count": 0,
            "generated_at": "2026-04-07T12:00:00Z"
        }
        """.data(using: .utf8)!

        let graph = try decoder.decode(SessionGraph.self, from: json)
        XCTAssertEqual(graph.roots.count, 1)
        XCTAssertEqual(graph.roots[0].children.count, 0, "omitted children field should decode as empty array")
    }

    func testSessionGraph_MultipleRoots() throws {
        let json = """
        {
            "roots": [
                {
                    "agent": \(agentJSON(id: "r1", displayName: "root-one", status: "done")),
                    "children": [],
                    "block_reason": "done",
                    "depth": 0
                },
                {
                    "agent": \(agentJSON(id: "r2", displayName: "root-two", status: "thinking")),
                    "children": [],
                    "block_reason": "",
                    "depth": 0
                }
            ],
            "total_count": 2,
            "blocked_count": 0,
            "generated_at": "2026-04-07T12:00:00Z"
        }
        """.data(using: .utf8)!

        let graph = try decoder.decode(SessionGraph.self, from: json)
        XCTAssertEqual(graph.roots.count, 2)
        XCTAssertEqual(graph.roots[0].agent.id, "r1")
        XCTAssertEqual(graph.roots[1].agent.id, "r2")
    }

    func testSessionNode_IDComputedFromAgent() throws {
        let node = SessionNode(
            agent: Agent(
                id: "agent-42",
                displayName: "test",
                provider: "claude",
                host: "",
                mode: .managed,
                projectPath: "/tmp",
                status: .idle,
                statusConfidence: 1,
                lastEventAt: Date()
            ),
            depth: 0
        )
        XCTAssertEqual(node.id, "agent-42")
    }

    func testSessionGraph_Equatable() throws {
        let json = """
        {
            "roots": [],
            "total_count": 0,
            "blocked_count": 0,
            "generated_at": "2026-04-07T12:00:00Z"
        }
        """.data(using: .utf8)!

        let g1 = try decoder.decode(SessionGraph.self, from: json)
        let g2 = try decoder.decode(SessionGraph.self, from: json)
        XCTAssertEqual(g1, g2)
    }
}
