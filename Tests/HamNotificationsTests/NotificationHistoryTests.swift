import Foundation
import XCTest
@testable import HamNotifications

final class NotificationHistoryTests: XCTestCase {
    func testInMemoryStoreAppendsAndLoadsEntries() {
        let store = InMemoryNotificationHistoryStore()
        let entry = NotificationHistoryEntry(
            key: "agent:1:error",
            title: "builder hit an error",
            body: "Build failed.",
            createdAt: Date(timeIntervalSince1970: 1)
        )

        store.append(entry)

        XCTAssertEqual(store.load(), [entry])
    }
}
