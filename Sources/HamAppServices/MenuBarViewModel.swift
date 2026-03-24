import Combine
import Foundation
import HamCore

@MainActor
public final class MenuBarViewModel: ObservableObject {
    @Published public private(set) var summary: HamMenuBarSummary?
    @Published public private(set) var agents: [Agent] = []
    @Published public private(set) var isRefreshing = false
    @Published public private(set) var errorMessage: String?

    private let client: HamDaemonClientProtocol
    private let summaryService: MenuBarSummaryService
    private let pollIntervalNanoseconds: UInt64
    private let sleep: @Sendable (UInt64) async throws -> Void
    private var hasStarted = false
    private var refreshTask: Task<Void, Never>?

    public init(
        client: HamDaemonClientProtocol,
        pollIntervalNanoseconds: UInt64 = 15_000_000_000,
        sleep: @escaping @Sendable (UInt64) async throws -> Void = { nanoseconds in
            try await Task.sleep(nanoseconds: nanoseconds)
        }
    ) {
        self.client = client
        self.summaryService = MenuBarSummaryService(client: client)
        self.pollIntervalNanoseconds = pollIntervalNanoseconds
        self.sleep = sleep
    }

    public var statusLine: String {
        guard let summary else {
            return errorMessage == nil ? "ham idle" : "ham offline"
        }
        return "ham \(summary.runningAgents)▶ \(summary.waitingAgents)? \(summary.doneAgents)✓"
    }

    public func start() {
        guard !hasStarted else { return }
        hasStarted = true

        refreshTask = Task { [weak self] in
            guard let self else { return }
            await self.refresh()

            while !Task.isCancelled {
                do {
                    try await self.sleep(self.pollIntervalNanoseconds)
                } catch {
                    break
                }

                if Task.isCancelled {
                    break
                }

                await self.refresh()
            }
        }
    }

    public func stop() {
        refreshTask?.cancel()
        refreshTask = nil
        hasStarted = false
    }

    public func refresh(eventLimit: Int = 5) async {
        isRefreshing = true
        defer { isRefreshing = false }

        do {
            async let loadedSummary = summaryService.refresh(eventLimit: eventLimit)
            async let loadedAgents = client.fetchAgents()

            summary = try await loadedSummary
            agents = try await loadedAgents
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    deinit {
        refreshTask?.cancel()
    }
}
