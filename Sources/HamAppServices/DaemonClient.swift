import Foundation
import HamCore
@preconcurrency import Network

public enum HamDaemonClientError: Error, Equatable, LocalizedError {
    case missingPayload(String)
    case server(String)
    case invalidSocketPath(String)
    case encodingFailed
    case transportFailed(String)

    public var errorDescription: String? {
        switch self {
        case .missingPayload(let payload):
            "Missing payload: \(payload)"
        case .server(let message):
            message
        case .invalidSocketPath(let path):
            "Invalid socket path: \(path)"
        case .encodingFailed:
            "Failed to encode daemon request."
        case .transportFailed(let message):
            message
        }
    }
}

public protocol DaemonTransport: Sendable {
    func send(_ request: DaemonRequest) async throws -> DaemonResponse
}

public protocol HamDaemonClientProtocol: Sendable {
    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload
    func fetchAgents() async throws -> [Agent]
    func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload]
    func fetchTeams() async throws -> [DaemonTeamPayload]
    func fetchEvents(limit: Int) async throws -> [AgentEventPayload]
    func followEvents(afterEventID: String, limit: Int, waitMilliseconds: Int) async throws -> [AgentEventPayload]
    func fetchSettings() async throws -> DaemonSettingsPayload
    func updateSettings(_ settings: DaemonSettingsPayload) async throws -> DaemonSettingsPayload
    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent
    func updateRole(agentID: String, role: String) async throws -> Agent
    func removeAgent(agentID: String) async throws
}

public extension HamDaemonClientProtocol {
    func fetchTeams() async throws -> [DaemonTeamPayload] { [] }
}

public struct HamMenuBarSummary: Equatable, Sendable {
    public var generatedAt: Date
    public var totalAgents: Int
    public var attentionAgents: Int
    public var attentionBreakdown: DaemonAttentionBreakdownPayload
    public var attentionOrder: [String]
    public var attentionSubtitles: [String: String]
    public var runningAgents: Int
    public var waitingAgents: Int
    public var doneAgents: Int
    public var recentEvents: [AgentEventPayload]

    public init(
        generatedAt: Date,
        totalAgents: Int,
        attentionAgents: Int,
        attentionBreakdown: DaemonAttentionBreakdownPayload,
        attentionOrder: [String],
        attentionSubtitles: [String: String],
        runningAgents: Int,
        waitingAgents: Int,
        doneAgents: Int,
        recentEvents: [AgentEventPayload]
    ) {
        self.generatedAt = generatedAt
        self.totalAgents = totalAgents
        self.attentionAgents = attentionAgents
        self.attentionBreakdown = attentionBreakdown
        self.attentionOrder = attentionOrder
        self.attentionSubtitles = attentionSubtitles
        self.runningAgents = runningAgents
        self.waitingAgents = waitingAgents
        self.doneAgents = doneAgents
        self.recentEvents = recentEvents
    }
}

public final class HamDaemonClient: HamDaemonClientProtocol, @unchecked Sendable {
    private let transport: DaemonTransport

    public init(transport: DaemonTransport) {
        self.transport = transport
    }

    public func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        let response = try await transport.send(.init(command: .status))
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        guard let snapshot = response.snapshot else {
            throw HamDaemonClientError.missingPayload("snapshot")
        }
        return snapshot
    }

    public func fetchAgents() async throws -> [Agent] {
        let response = try await transport.send(.init(command: .listAgents))
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        return response.agents ?? []
    }

    public func fetchAttachableSessions() async throws -> [DaemonAttachableSessionPayload] {
        let response = try await transport.send(.init(command: .listItermSessions))
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        return response.attachableSessions ?? []
    }

    public func fetchTeams() async throws -> [DaemonTeamPayload] {
        let response = try await transport.send(.init(command: .listTeams))
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        return response.teams ?? []
    }

    public func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        let response = try await transport.send(.init(command: .events, limit: limit))
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        return response.events ?? []
    }

    public func followEvents(afterEventID: String, limit: Int, waitMilliseconds: Int) async throws -> [AgentEventPayload] {
        let response = try await transport.send(
            .init(command: .followEvents, limit: limit, afterEventID: afterEventID, waitMillis: waitMilliseconds)
        )
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        return response.events ?? []
    }

    public func fetchSettings() async throws -> DaemonSettingsPayload {
        let response = try await transport.send(.init(command: .getSettings))
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        guard let settings = response.settings else {
            throw HamDaemonClientError.missingPayload("settings")
        }
        return settings
    }

    public func updateSettings(_ settings: DaemonSettingsPayload) async throws -> DaemonSettingsPayload {
        let response = try await transport.send(.init(command: .updateSettings, settings: settings))
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        guard let updated = response.settings else {
            throw HamDaemonClientError.missingPayload("settings")
        }
        return updated
    }

    public func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent {
        let response = try await transport.send(
            .init(command: .setNotificationPolicy, agentID: agentID, policy: policy.rawValue)
        )
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        guard let agent = response.agent else {
            throw HamDaemonClientError.missingPayload("agent")
        }
        return agent
    }

    public func updateRole(agentID: String, role: String) async throws -> Agent {
        let response = try await transport.send(
            .init(command: .setRole, agentID: agentID, role: role)
        )
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        guard let agent = response.agent else {
            throw HamDaemonClientError.missingPayload("agent")
        }
        return agent
    }

    public func removeAgent(agentID: String) async throws {
        let response = try await transport.send(
            .init(command: .removeAgent, agentID: agentID)
        )
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
    }
}

public struct MenuBarSummaryService: Sendable {
    private let client: HamDaemonClientProtocol

    public init(client: HamDaemonClientProtocol) {
        self.client = client
    }

    public func refresh(eventLimit: Int = 5) async throws -> HamMenuBarSummary {
        let snapshot = try await client.fetchSnapshot()
        let events = try await client.fetchEvents(limit: eventLimit)

        return HamMenuBarSummary(
            generatedAt: snapshot.generatedAt,
            totalAgents: snapshot.totalCount,
            attentionAgents: snapshot.attentionCount,
            attentionBreakdown: snapshot.attentionBreakdown,
            attentionOrder: snapshot.attentionOrder,
            attentionSubtitles: snapshot.attentionSubtitles,
            runningAgents: snapshot.runningCount,
            waitingAgents: snapshot.waitingCount,
            doneAgents: snapshot.doneCount,
            recentEvents: events
        )
    }
}

public enum DaemonEnvironment {
    public static func defaultSocketPath(
        env: [String: String] = ProcessInfo.processInfo.environment,
        homeDirectory: @autoclosure () throws -> URL = FileManager.default.homeDirectoryForCurrentUser
    ) throws -> String {
        if let socketPath = env["HAM_AGENTS_SOCKET"], !socketPath.isEmpty {
            return socketPath
        }
        if let root = env["HAM_AGENTS_HOME"], !root.isEmpty {
            return URL(fileURLWithPath: root).appendingPathComponent("hamd.sock").path
        }
        return try homeDirectory()
            .appendingPathComponent("Library/Application Support/ham-agents/hamd.sock")
            .path
    }
}

public final class UnixSocketDaemonTransport: DaemonTransport, @unchecked Sendable {
    private let socketPath: String
    private let encoder: JSONEncoder
    private let decoder: JSONDecoder
    private let queue: DispatchQueue

    public init(socketPath: String, queueLabel: String = "ham-agents.daemon-transport") {
        self.socketPath = socketPath
        self.encoder = JSONEncoder()
        self.decoder = DaemonJSONDecoder.make()
        self.queue = DispatchQueue(label: queueLabel)
    }

    public convenience init() throws {
        try self.init(socketPath: DaemonEnvironment.defaultSocketPath())
    }

    public func send(_ request: DaemonRequest) async throws -> DaemonResponse {
        guard !socketPath.isEmpty else {
            throw HamDaemonClientError.invalidSocketPath(socketPath)
        }

        let payload: Data
        do {
            payload = try encoder.encode(request)
        } catch {
            throw HamDaemonClientError.encodingFailed
        }

        return try await withCheckedThrowingContinuation { continuation in
            queue.async { [socketPath, decoder] in
                do {
                    let fd = socket(AF_UNIX, SOCK_STREAM, 0)
                    guard fd >= 0 else {
                        throw HamDaemonClientError.transportFailed("socket() failed: \(errno)")
                    }

                    var addr = sockaddr_un()
                    addr.sun_family = sa_family_t(AF_UNIX)
                    let pathBytes = socketPath.utf8CString
                    guard pathBytes.count <= MemoryLayout.size(ofValue: addr.sun_path) else {
                        close(fd)
                        throw HamDaemonClientError.transportFailed("socket path too long")
                    }
                    withUnsafeMutablePointer(to: &addr.sun_path) { ptr in
                        ptr.withMemoryRebound(to: CChar.self, capacity: pathBytes.count) { dest in
                            for i in 0..<pathBytes.count { dest[i] = pathBytes[i] }
                        }
                    }

                    let connectResult = withUnsafePointer(to: &addr) { ptr in
                        ptr.withMemoryRebound(to: sockaddr.self, capacity: 1) { sockaddrPtr in
                            Darwin.connect(fd, sockaddrPtr, socklen_t(MemoryLayout<sockaddr_un>.size))
                        }
                    }
                    guard connectResult == 0 else {
                        close(fd)
                        throw HamDaemonClientError.transportFailed("connect() failed: \(errno)")
                    }

                    // Set socket timeouts to prevent indefinite blocking.
                    var timeout = timeval(tv_sec: 30, tv_usec: 0)
                    setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, socklen_t(MemoryLayout<timeval>.size))
                    setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, socklen_t(MemoryLayout<timeval>.size))

                    // Send
                    let sent = payload.withUnsafeBytes { buf in
                        guard let ptr = buf.baseAddress, buf.count > 0 else { return 0 }
                        return Darwin.write(fd, ptr, buf.count)
                    }
                    guard sent == payload.count else {
                        close(fd)
                        throw HamDaemonClientError.transportFailed("write() incomplete")
                    }

                    // Receive until EOF
                    var responseData = Data()
                    var buf = [UInt8](repeating: 0, count: 65536)
                    while true {
                        let n = Darwin.read(fd, &buf, buf.count)
                        if n <= 0 { break }
                        responseData.append(buf, count: n)
                    }
                    close(fd)

                    let response = try decoder.decode(DaemonResponse.self, from: responseData)
                    continuation.resume(returning: response)
                } catch {
                    continuation.resume(throwing: error)
                }
            }
        }
    }
}

private final class ReceiveAccumulator: @unchecked Sendable {
    private let decoder: JSONDecoder
    private let continuation: CheckedContinuation<DaemonResponse, Error>
    private var buffer = Data()
    private var finished = false
    private let lock = NSLock()

    init(
        decoder: JSONDecoder,
        continuation: CheckedContinuation<DaemonResponse, Error>
    ) {
        self.decoder = decoder
        self.continuation = continuation
    }

    func append(_ content: Data) {
        lock.lock()
        defer { lock.unlock() }
        guard !finished else { return }
        buffer.append(content)
    }

    func succeed() {
        lock.lock()
        defer { lock.unlock() }
        guard !finished else { return }
        finished = true

        do {
            let response = try decoder.decode(DaemonResponse.self, from: buffer)
            continuation.resume(returning: response)
        } catch {
            continuation.resume(throwing: HamDaemonClientError.transportFailed(error.localizedDescription))
        }
    }

    func fail(_ error: Error) {
        lock.lock()
        defer { lock.unlock() }
        guard !finished else { return }
        finished = true
        continuation.resume(throwing: error)
    }
}
