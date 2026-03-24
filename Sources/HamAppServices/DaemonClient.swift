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
    func fetchEvents(limit: Int) async throws -> [AgentEventPayload]
    func updateNotificationPolicy(agentID: String, policy: NotificationPolicy) async throws -> Agent
    func updateRole(agentID: String, role: String) async throws -> Agent
}

public struct HamMenuBarSummary: Equatable, Sendable {
    public var generatedAt: Date
    public var totalAgents: Int
    public var runningAgents: Int
    public var waitingAgents: Int
    public var doneAgents: Int
    public var recentEvents: [AgentEventPayload]

    public init(
        generatedAt: Date,
        totalAgents: Int,
        runningAgents: Int,
        waitingAgents: Int,
        doneAgents: Int,
        recentEvents: [AgentEventPayload]
    ) {
        self.generatedAt = generatedAt
        self.totalAgents = totalAgents
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

    public func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        let response = try await transport.send(.init(command: .events, limit: limit))
        if let error = response.error {
            throw HamDaemonClientError.server(error)
        }
        return response.events ?? []
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

        let connection = NWConnection(to: .unix(path: socketPath), using: NWParameters(tls: nil))

        return try await withCheckedThrowingContinuation { continuation in
            let accumulator = ReceiveAccumulator(
                decoder: decoder,
                continuation: continuation
            )

            connection.stateUpdateHandler = { (state: NWConnection.State) in
                switch state {
                case .ready:
                    connection.send(content: payload, completion: NWConnection.SendCompletion.contentProcessed { error in
                        if let error {
                            accumulator.fail(HamDaemonClientError.transportFailed(error.localizedDescription))
                            connection.cancel()
                            return
                        }

                        Self.receiveNextChunk(on: connection, accumulator: accumulator)
                    })
                case .failed(let error):
                    accumulator.fail(HamDaemonClientError.transportFailed(error.localizedDescription))
                    connection.cancel()
                default:
                    break
                }
            }

            connection.start(queue: queue)
        }
    }

    private static func receiveNextChunk(on connection: NWConnection, accumulator: ReceiveAccumulator) {
        connection.receive(minimumIncompleteLength: 1, maximumLength: 64 * 1024) { content, _, isComplete, error in
            if let error {
                accumulator.fail(HamDaemonClientError.transportFailed(error.localizedDescription))
                connection.cancel()
                return
            }

            if let content, !content.isEmpty {
                accumulator.append(content)
            }

            if isComplete {
                accumulator.succeed()
                connection.cancel()
                return
            }

            receiveNextChunk(on: connection, accumulator: accumulator)
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
