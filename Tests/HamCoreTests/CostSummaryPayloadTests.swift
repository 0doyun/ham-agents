import Foundation
import XCTest
@testable import HamCore

final class CostSummaryPayloadTests: XCTestCase {
    func testDecodeCostRecordPayloadFromGoJSON() throws {
        // Mirrors the JSON wire format emitted by the Go FileCostStore.
        let json = """
        {
            "agent_id": "agent-1",
            "session_id": "sess-abc",
            "model": "claude-opus-4-6",
            "service_tier": "standard",
            "input_tokens": 100,
            "cache_create_5m_tokens": 0,
            "cache_create_1h_tokens": 20525,
            "cache_read_tokens": 50,
            "output_tokens": 200,
            "estimated_usd": 1.23,
            "recorded_at": "2026-04-08T12:34:56Z",
            "source": "assistant",
            "request_id": "req_abc",
            "message_id": "msg_001"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        let record = try decoder.decode(CostRecordPayload.self, from: json)
        XCTAssertEqual(record.model, "claude-opus-4-6")
        XCTAssertEqual(record.inputTokens, 100)
        XCTAssertEqual(record.cacheCreate1hTokens, 20525)
        XCTAssertEqual(record.cacheReadTokens, 50)
        XCTAssertEqual(record.outputTokens, 200)
        XCTAssertEqual(record.estimatedUSD, 1.23, accuracy: 1e-9)
        XCTAssertEqual(record.requestID, "req_abc")
        XCTAssertEqual(record.source, "assistant")
    }

    func testDecodeCostRecordPayloadHandlesMissingOptionalFields() throws {
        // The Go side omits zero-value tokens via `omitempty`. The Swift
        // decoder must default them to 0 instead of throwing.
        let json = """
        {
            "model": "claude-haiku-4-5",
            "input_tokens": 5,
            "output_tokens": 10,
            "estimated_usd": 0.0001,
            "recorded_at": "2026-04-08T00:00:00Z",
            "source": "assistant"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        let record = try decoder.decode(CostRecordPayload.self, from: json)
        XCTAssertEqual(record.cacheCreate5mTokens, 0)
        XCTAssertEqual(record.cacheReadTokens, 0)
        XCTAssertNil(record.agentID)
    }

    func testCostSummaryPayloadFromComputesTodayUSD() throws {
        // Pin "now" to a fixed UTC date so the today bucket is deterministic.
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.timeZone = TimeZone(identifier: "UTC")
        let now = formatter.date(from: "2026-04-08")!

        let response = DaemonResponse(
            costRecords: [],
            totalUSD: 5.0,
            byModel: ["claude-opus-4-6": 4.0, "claude-haiku-4-5": 1.0],
            byDay: ["2026-04-07": 2.0, "2026-04-08": 3.0],
            byAgent: ["a1": 4.0, "(orphan)": 1.0]
        )
        let summary = CostSummaryPayload.from(response: response, now: now)
        XCTAssertEqual(summary.totalUSD, 5.0, accuracy: 1e-9)
        XCTAssertEqual(summary.todayUSD, 3.0, accuracy: 1e-9)
        XCTAssertEqual(summary.byModel["claude-opus-4-6"], 4.0)
        XCTAssertEqual(summary.byAgent["(orphan)"], 1.0)
    }

    func testCostSummaryPayloadFromHandlesEmptyResponse() {
        let summary = CostSummaryPayload.from(response: DaemonResponse())
        XCTAssertEqual(summary.totalUSD, 0)
        XCTAssertEqual(summary.todayUSD, 0)
        XCTAssertTrue(summary.byModel.isEmpty)
        XCTAssertTrue(summary.records.isEmpty)
    }
}
