import Foundation

public struct NotificationHistoryEntry: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var key: String
    public var title: String
    public var body: String
    public var createdAt: Date

    public init(id: String = UUID().uuidString, key: String, title: String, body: String, createdAt: Date) {
        self.id = id
        self.key = key
        self.title = title
        self.body = body
        self.createdAt = createdAt
    }
}

public protocol NotificationHistoryStoring: Sendable {
    func load() -> [NotificationHistoryEntry]
    func append(_ entry: NotificationHistoryEntry)
}

public final class InMemoryNotificationHistoryStore: NotificationHistoryStoring, @unchecked Sendable {
    private var entries: [NotificationHistoryEntry]
    private let lock = NSLock()

    public init(entries: [NotificationHistoryEntry] = []) {
        self.entries = entries
    }

    public func load() -> [NotificationHistoryEntry] {
        lock.lock(); defer { lock.unlock() }
        return entries
    }

    public func append(_ entry: NotificationHistoryEntry) {
        lock.lock(); defer { lock.unlock() }
        entries.append(entry)
        entries = Array(entries.suffix(200))
    }
}

public final class FileNotificationHistoryStore: NotificationHistoryStoring, @unchecked Sendable {
    private let path: String
    private let lock = NSLock()
    private let encoder = JSONEncoder()
    private let decoder = JSONDecoder()

    public init(path: String? = nil) {
        self.path = path ?? Self.defaultPath()
        encoder.dateEncodingStrategy = .iso8601
        decoder.dateDecodingStrategy = .iso8601
    }

    public func load() -> [NotificationHistoryEntry] {
        lock.lock(); defer { lock.unlock() }
        guard let data = try? Data(contentsOf: URL(fileURLWithPath: path)) else { return [] }
        return (try? decoder.decode([NotificationHistoryEntry].self, from: data)) ?? []
    }

    public func append(_ entry: NotificationHistoryEntry) {
        lock.lock(); defer { lock.unlock() }
        let url = URL(fileURLWithPath: path)
        try? FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        var entries = (try? Data(contentsOf: url)).flatMap { try? decoder.decode([NotificationHistoryEntry].self, from: $0) } ?? []
        entries.append(entry)
        entries = Array(entries.suffix(200))
        if let data = try? encoder.encode(entries) {
            try? data.write(to: url)
        }
    }

    private static func defaultPath() -> String {
        if let root = ProcessInfo.processInfo.environment["HAM_AGENTS_HOME"], !root.isEmpty {
            return URL(fileURLWithPath: root).appendingPathComponent("notification-history.json").path
        }
        return FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/Application Support/ham-agents/notification-history.json")
            .path
    }
}
