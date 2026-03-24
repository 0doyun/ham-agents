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
    private var hasStarted = false

    public init(client: HamDaemonClientProtocol) {
        self.client = client
        self.summaryService = MenuBarSummaryService(client: client)
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

        Task { [weak self] in
            await self?.refresh()
        }
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
}
