import SwiftUI
import HamAppServices
import HamCore
import HamNotifications

@main
struct HamMenuBarApp: App {
    @StateObject private var viewModel = HamMenuBarApp.makeViewModel()

    var body: some Scene {
        MenuBarExtra {
            MenuBarContentView(viewModel: viewModel)
                .frame(minWidth: 320, minHeight: 220)
                .task {
                    await viewModel.refresh()
                }
        } label: {
            Text(viewModel.statusLine)
        }
        .menuBarExtraStyle(.window)
    }

    private static func makeViewModel() -> MenuBarViewModel {
        let client: HamDaemonClientProtocol
        if let transport = try? UnixSocketDaemonTransport() {
            client = HamDaemonClient(transport: transport)
        } else {
            client = PreviewDaemonClient()
        }
        let viewModel = MenuBarViewModel(
            client: client,
            notificationSink: UserNotificationSink()
        )
        viewModel.start()
        return viewModel
    }
}

private struct MenuBarContentView: View {
    @ObservedObject var viewModel: MenuBarViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("Ham Office")
                    .font(.headline)
                Spacer()
                if viewModel.isRefreshing {
                    ProgressView()
                        .controlSize(.small)
                }
                Button("Refresh") {
                    Task { await viewModel.refresh() }
                }
            }

            if let summary = viewModel.summary {
                HStack {
                    SummaryBadge(title: "Total", value: summary.totalAgents)
                    SummaryBadge(title: "Run", value: summary.runningAgents)
                    SummaryBadge(title: "Wait", value: summary.waitingAgents)
                    SummaryBadge(title: "Done", value: summary.doneAgents)
                }
            }

            if let errorMessage = viewModel.errorMessage {
                Text(errorMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Text("Agents")
                .font(.subheadline.weight(.semibold))

            if viewModel.agents.isEmpty {
                Text("No tracked agents")
                    .foregroundStyle(.secondary)
            } else {
                List(viewModel.agents) { agent in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(agent.displayName)
                            .font(.body.weight(.medium))
                        Text("\(agent.status.rawValue) · \(agent.projectPath)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                }
                .listStyle(.plain)
            }
        }
        .padding(14)
    }
}

private struct SummaryBadge: View {
    let title: String
    let value: Int

    var body: some View {
        VStack(spacing: 4) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(String(value))
                .font(.headline.monospacedDigit())
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .background(Color.gray.opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}

private struct PreviewDaemonClient: HamDaemonClientProtocol {
    func fetchSnapshot() async throws -> DaemonRuntimeSnapshotPayload {
        DaemonRuntimeSnapshotPayload(
            agents: [
                Agent(
                    id: "preview-1",
                    displayName: "preview-reviewer",
                    provider: "claude",
                    host: "localhost",
                    mode: .managed,
                    projectPath: "/tmp/demo",
                    status: .thinking,
                    statusConfidence: 1,
                    lastEventAt: .now
                )
            ],
            generatedAt: .now
        )
    }

    func fetchAgents() async throws -> [Agent] {
        (try await fetchSnapshot()).agents
    }

    func fetchEvents(limit: Int) async throws -> [AgentEventPayload] {
        [
            AgentEventPayload(
                id: "preview-event-1",
                agentID: "preview-1",
                type: "agent.registered",
                summary: "Preview mode active.",
                occurredAt: .now
            )
        ]
    }
}
