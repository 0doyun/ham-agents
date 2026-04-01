import XCTest
@testable import HamAppServices

final class ItermAppleScriptsTests: XCTestCase {
    func testFocusSessionComparesSessionIDAsStringAndErrorsWhenMissing() {
        let script = ItermAppleScripts.focusSession("abc")

        XCTAssertTrue(script.contains("(id of aSession as string) is \"abc\""))
        XCTAssertTrue(script.contains("error \"session not found\" number 1"))
    }

    func testWriteToSessionDoesNotFallBackToCurrentSession() {
        let script = ItermAppleScripts.writeToSession("abc", message: "hello")

        XCTAssertTrue(script.contains("(id of aSession as string) is \"abc\""))
        XCTAssertFalse(script.contains("tell current window"))
        XCTAssertTrue(script.contains("error \"session not found\" number 1"))
    }
}
