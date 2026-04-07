import Foundation
import XCTest
@testable import HamCore

final class DaemonIPCTests: XCTestCase {
    func testDaemonCommand_UnknownRawValue_MapsToUnknown() throws {
        let payload = """
        {"command": "totally-made-up-command"}
        """
        let data = try XCTUnwrap(payload.data(using: .utf8))

        struct Wrapper: Decodable {
            let command: DaemonCommand
        }

        let wrapper = try JSONDecoder().decode(Wrapper.self, from: data)
        XCTAssertEqual(wrapper.command, .unknown)
    }

    func testDaemonCommand_KnownRawValues_DecodeCorrectly() throws {
        let cases: [(String, DaemonCommand)] = [
            ("register.managed", .registerManaged),
            ("managed.stop", .managedStop),
            ("managed.exited", .managedExited),
            ("agents.rename", .agentsRename),
            ("agents.open_target", .agentsOpenTarget),
            ("tmux.sessions", .tmuxSessions),
        ]

        struct Wrapper: Decodable {
            let command: DaemonCommand
        }

        for (raw, expected) in cases {
            let payload = "{\"command\": \"\(raw)\"}"
            let data = try XCTUnwrap(payload.data(using: .utf8))
            let wrapper = try JSONDecoder().decode(Wrapper.self, from: data)
            XCTAssertEqual(wrapper.command, expected, "Expected \(expected) for raw value '\(raw)'")
        }
    }
}
