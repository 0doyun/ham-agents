import SwiftUI
import HamAppServices
import HamCore
import HamNotifications

struct MenuBarContentView: View {
    @ObservedObject var viewModel: MenuBarViewModel
    @State private var selectedAgentID: Agent.ID?
    @State private var quickMessage = ""

    var body: some View {
        HStack(alignment: .top, spacing: 14) {
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
                        SummaryBadge(title: "Attn", value: summary.attentionAgents)
                        SummaryBadge(title: "Run", value: summary.runningAgents)
                        SummaryBadge(title: "Wait", value: summary.waitingAgents)
                        SummaryBadge(title: "Done", value: summary.doneAgents)
                    }
                    let attentionBreakdownChips = viewModel.topSummaryAttentionBreakdownChips
                    if !attentionBreakdownChips.isEmpty {
                        EventSummaryChipsView(chips: attentionBreakdownChips)
                    }
                }

                if let errorMessage = viewModel.errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                if let presentation = viewModel.latestEventPresentation,
                   let summary = viewModel.latestEventSummary {
                    LatestEventBanner(presentation: presentation, summary: summary)
                }

                let recentActivitySeverityChips = viewModel.recentEventSeverityChips(forAgentID: nil)
                let recentActivityChips = viewModel.recentEventSummaryChips(forAgentID: nil)
                if !recentActivitySeverityChips.isEmpty || !recentActivityChips.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Recent Activity")
                            .font(.caption.weight(.semibold))
                        if !recentActivitySeverityChips.isEmpty {
                            EventSummaryChipsView(chips: recentActivitySeverityChips)
                        }
                        if !recentActivityChips.isEmpty {
                            EventSummaryChipsView(chips: recentActivityChips)
                        }
                    }
                }

                NotificationPermissionRow(
                    status: viewModel.notificationPermissionStatus,
                    requestPermission: {
                        Task { await viewModel.requestNotificationPermission() }
                    }
                )

                NotificationSettingsSection(
                    settings: viewModel.settings.notifications,
                    updateDone: { value in
                        Task { await viewModel.updateNotificationSetting(done: value) }
                    },
                    updateError: { value in
                        Task { await viewModel.updateNotificationSetting(error: value) }
                    },
                    updateWaiting: { value in
                        Task { await viewModel.updateNotificationSetting(waitingInput: value) }
                    },
                    updateSilence: { value in
                        Task { await viewModel.updateNotificationSetting(silence: value) }
                    },
                    updateQuietHours: { value in
                        Task { await viewModel.updateNotificationSetting(quietHoursEnabled: value) }
                    },
                    updateQuietStartHour: { value in
                        Task { await viewModel.updateNotificationSetting(quietHoursStartHour: value) }
                    },
                    updateQuietEndHour: { value in
                        Task { await viewModel.updateNotificationSetting(quietHoursEndHour: value) }
                    },
                    updatePreviewText: { value in
                        Task { await viewModel.updateNotificationSetting(previewText: value) }
                    }
                )

                AppearanceSettingsSection(
                    settings: viewModel.settings.appearance,
                    updateTheme: { value in
                        Task { await viewModel.updateAppearanceSetting(theme: value) }
                    }
                )

                IntegrationSettingsSection(
                    settings: viewModel.settings.integrations,
                    updateItermEnabled: { value in
                        Task { await viewModel.updateIntegrationSetting(itermEnabled: value) }
                    }
                )

                if viewModel.settings.integrations.itermEnabled && !viewModel.attachableSessions.isEmpty {
                    AttachableSessionsSection(sessions: viewModel.attachableSessions)
                }

                Text("Agents")
                    .font(.subheadline.weight(.semibold))

                if !viewModel.attentionAgents.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Needs Attention")
                            .font(.caption.weight(.semibold))
                        ForEach(viewModel.attentionAgents) { agent in
                            AttentionAgentRow(
                                name: agent.displayName,
                                subtitle: viewModel.attentionSubtitle(for: agent)
                            )
                        }
                    }
                }

                if viewModel.agents.isEmpty {
                    Text("No tracked agents")
                        .foregroundStyle(.secondary)
                } else {
                    List(selection: $selectedAgentID) {
                        ForEach(viewModel.nonAttentionAgents) { agent in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(agent.displayName)
                                    .font(.body.weight(.medium))
                                Text("\(viewModel.statusDisplayText(for: agent)) · \(agent.mode.rawValue) · \(viewModel.confidenceLevelText(for: agent)) \(viewModel.confidenceText(for: agent))")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .lineLimit(1)
                            }
                            .tag(agent.id)
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .frame(minWidth: 190)

            Divider()

            AgentDetailView(
                agent: viewModel.agent(withID: selectedAgentID),
                recentEvents: viewModel.recentEvents(forAgentID: selectedAgentID),
                recentEventSummaryChips: viewModel.recentEventSummaryChips(forAgentID: selectedAgentID),
                notificationsMuted: viewModel.isNotificationsMuted(forAgentID: selectedAgentID),
                quickMessageFeedback: viewModel.quickMessageFeedback,
                confidenceText: viewModel.confidenceSummaryText(for: viewModel.agent(withID: selectedAgentID)),
                roleDraft: Binding(
                    get: { viewModel.roleDraft },
                    set: { viewModel.roleDraft = $0 }
                ),
                quickMessage: $quickMessage,
                openProject: {
                    viewModel.openProject(forAgentID: selectedAgentID)
                },
                openSession: {
                    viewModel.openSession(forAgentID: selectedAgentID)
                },
                canOpenSession: viewModel.settings.integrations.itermEnabled,
                toggleNotifications: {
                    viewModel.toggleNotificationPause(forAgentID: selectedAgentID)
                },
                saveRole: {
                    await viewModel.saveRole(forAgentID: selectedAgentID)
                },
                stopTracking: {
                    await viewModel.stopTracking(forAgentID: selectedAgentID)
                    selectedAgentID = viewModel.agents.first?.id
                },
                sendQuickMessage: {
                    viewModel.sendQuickMessage(quickMessage, forAgentID: selectedAgentID)
                    quickMessage = ""
                }
            )
            .frame(minWidth: 140, maxWidth: .infinity, alignment: .topLeading)
        }
        .padding(14)
        .onAppear {
            if selectedAgentID == nil {
                selectedAgentID = viewModel.agents.first?.id
            }
            viewModel.setRoleDraft(from: selectedAgentID)
        }
        .onChange(of: viewModel.agents.map(\.id)) { ids in
            if selectedAgentID == nil || !ids.contains(selectedAgentID ?? "") {
                selectedAgentID = ids.first
            }
            viewModel.setRoleDraft(from: selectedAgentID)
        }
    }
}

private struct AttentionAgentRow: View {
    let name: String
    let subtitle: String

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(name)
                    .font(.caption.weight(.semibold))
                Text(subtitle)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
            Spacer()
        }
        .padding(6)
        .background(Color.orange.opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}

private struct NotificationSettingsSection: View {
    let settings: DaemonNotificationSettingsPayload
    let updateDone: (Bool) -> Void
    let updateError: (Bool) -> Void
    let updateWaiting: (Bool) -> Void
    let updateSilence: (Bool) -> Void
    let updateQuietHours: (Bool) -> Void
    let updateQuietStartHour: (Int) -> Void
    let updateQuietEndHour: (Int) -> Void
    let updatePreviewText: (Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Notification Settings")
                .font(.caption.weight(.semibold))

            Toggle("Done", isOn: Binding(get: { settings.done }, set: updateDone))
            Toggle("Error", isOn: Binding(get: { settings.error }, set: updateError))
            Toggle("Waiting Input", isOn: Binding(get: { settings.waitingInput }, set: updateWaiting))
            Toggle("Silence", isOn: Binding(get: { settings.silence }, set: updateSilence))
            Toggle("Quiet Hours", isOn: Binding(get: { settings.quietHoursEnabled }, set: updateQuietHours))
            if settings.quietHoursEnabled {
                HStack {
                    Text("Start")
                    Spacer()
                    Text(hourLabel(settings.quietHoursStartHour))
                        .foregroundStyle(.secondary)
                    Stepper(
                        "",
                        value: Binding(
                            get: { settings.quietHoursStartHour },
                            set: updateQuietStartHour
                        ),
                        in: 0...23
                    )
                    .labelsHidden()
                }
                HStack {
                    Text("End")
                    Spacer()
                    Text(hourLabel(settings.quietHoursEndHour))
                        .foregroundStyle(.secondary)
                    Stepper(
                        "",
                        value: Binding(
                            get: { settings.quietHoursEndHour },
                            set: updateQuietEndHour
                        ),
                        in: 0...23
                    )
                    .labelsHidden()
                }
                Text("Current window \(hourLabel(settings.quietHoursStartHour)) → \(hourLabel(settings.quietHoursEndHour))")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            Toggle("Preview Text", isOn: Binding(get: { settings.previewText }, set: updatePreviewText))
        }
        .toggleStyle(.checkbox)
    }

    private func hourLabel(_ hour: Int) -> String {
        String(format: "%02d:00", hour)
    }
}

private struct NotificationPermissionRow: View {
    let status: NotificationPermissionStatus
    let requestPermission: () -> Void

    var body: some View {
        HStack {
            Text("Notifications")
                .font(.caption.weight(.semibold))
            Spacer()
            Text(statusLabel)
                .font(.caption)
                .foregroundStyle(.secondary)
            if status != .authorized {
                Button("Enable") {
                    requestPermission()
                }
                .buttonStyle(.borderless)
            }
        }
    }

    private var statusLabel: String {
        switch status {
        case .authorized:
            "Enabled"
        case .denied:
            "Denied"
        case .notDetermined:
            "Not requested"
        }
    }
}

private struct AttachableSessionsSection: View {
    let sessions: [DaemonAttachableSessionPayload]

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Attachable iTerm Sessions")
                .font(.caption.weight(.semibold))

            ForEach(sessions.prefix(3)) { session in
                HStack(spacing: 6) {
                    Text(session.title)
                        .lineLimit(1)
                    if session.isActive {
                        Text("Current")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
                .font(.caption)
            }
        }
    }
}

private struct IntegrationSettingsSection: View {
    let settings: DaemonIntegrationSettingsPayload
    let updateItermEnabled: (Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Integrations")
                .font(.caption.weight(.semibold))

            Toggle(
                "iTerm2 Access",
                isOn: Binding(
                    get: { settings.itermEnabled },
                    set: updateItermEnabled
                )
            )
        }
        .toggleStyle(.checkbox)
    }
}

private struct LatestEventBanner: View {
    let presentation: AgentEventPresentation
    let summary: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(presentation.label)
                .font(.caption.weight(.semibold))
            Text(summary)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(2)
        }
        .padding(8)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(eventBadgeBackground(for: presentation.emphasis))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}

private struct AppearanceSettingsSection: View {
    let settings: DaemonAppearanceSettingsPayload
    let updateTheme: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Appearance")
                .font(.caption.weight(.semibold))

            Picker(
                "Theme",
                selection: Binding(
                    get: { settings.theme },
                    set: updateTheme
                )
            ) {
                Text("Auto").tag("auto")
                Text("Day").tag("day")
                Text("Night").tag("night")
            }
            .labelsHidden()
        }
    }
}

private struct AgentDetailView: View {
    let agent: Agent?
    let recentEvents: [AgentEventPayload]
    let recentEventSummaryChips: [AgentEventSummaryChip]
    let notificationsMuted: Bool
    let quickMessageFeedback: String?
    let confidenceText: String
    @Binding var roleDraft: String
    @Binding var quickMessage: String
    let openProject: () -> Void
    let openSession: () -> Void
    let canOpenSession: Bool
    let toggleNotifications: () -> Void
    let saveRole: () async -> Void
    let stopTracking: () async -> Void
    let sendQuickMessage: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Details")
                .font(.subheadline.weight(.semibold))

            if let agent {
                Text(agent.displayName)
                    .font(.headline)
                Text("\(agent.provider) · \(agent.mode.rawValue)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                if let sessionTitle = agent.sessionTitle, !sessionTitle.isEmpty {
                    Text(sessionTitle)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                if agent.mode == .attached {
                    Text(agent.sessionIsActive ? "Current iTerm session" : "Background iTerm session")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    if let sessionTTY = agent.sessionTTY, !sessionTTY.isEmpty {
                        Text("tty \(sessionTTY)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let sessionWorkingDirectory = agent.sessionWorkingDirectory, !sessionWorkingDirectory.isEmpty {
                        Text(sessionWorkingDirectory)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }
                    if let sessionActivity = agent.sessionActivity, !sessionActivity.isEmpty {
                        Text("activity \(sessionActivity)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let sessionProcessID = agent.sessionProcessID {
                        Text("pid \(sessionProcessID)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let sessionCommand = agent.sessionCommand, !sessionCommand.isEmpty {
                        Text(sessionCommand)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }
                }
                Text(agent.projectPath)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
                Text(confidenceText)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                if let statusReason = agent.statusReason, !statusReason.isEmpty {
                    Text("Reason: \(statusReason)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                }

                Text("Role")
                    .font(.caption.weight(.semibold))
                HStack {
                    TextField("Role…", text: $roleDraft)
                        .textFieldStyle(.roundedBorder)
                    Button("Save") {
                        Task { await saveRole() }
                    }
                    .buttonStyle(.bordered)
                }

                if let summary = agent.lastUserVisibleSummary {
                    Text(summary)
                        .font(.caption)
                }

                Button("Open Project Folder") {
                    openProject()
                }
                .buttonStyle(.bordered)

                Button("Open in iTerm") {
                    openSession()
                }
                .buttonStyle(.borderedProminent)
                .disabled(!canOpenSession)

                Button("Stop Tracking") {
                    Task { await stopTracking() }
                }
                .buttonStyle(.bordered)
                .tint(.red)

                Button(notificationsMuted ? "Resume Notifications" : "Pause Notifications") {
                    toggleNotifications()
                }
                .buttonStyle(.bordered)

                Text("Quick Message")
                    .font(.caption.weight(.semibold))
                    .padding(.top, 4)
                TextField("Draft message…", text: $quickMessage, axis: .vertical)
                    .textFieldStyle(.roundedBorder)
                    .lineLimit(2 ... 4)
                Button("Send Message") {
                    sendQuickMessage()
                }
                .buttonStyle(.borderedProminent)
                .disabled(quickMessage.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                if let quickMessageFeedback {
                    Text(quickMessageFeedback)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Text("Recent Events")
                    .font(.caption.weight(.semibold))
                    .padding(.top, 4)

                let recentEventSeverityChips = AgentEventPresenter.summarizeBySeverity(recentEvents)
                if !recentEventSeverityChips.isEmpty {
                    EventSummaryChipsView(chips: recentEventSeverityChips)
                }
                if !recentEventSummaryChips.isEmpty {
                    EventSummaryChipsView(chips: recentEventSummaryChips)
                }

                if recentEvents.isEmpty {
                    Text("No recent events for this agent.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(recentEvents.prefix(3)) { event in
                        let presentation = AgentEventPresenter.present(event)
                        VStack(alignment: .leading, spacing: 2) {
                            HStack(spacing: 6) {
                                Text(presentation.label)
                                    .font(.caption2.weight(.semibold))
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 2)
                                    .background(eventBadgeBackground(for: presentation.emphasis))
                                    .clipShape(Capsule())
                                if presentation.showsTechnicalType {
                                    Text(event.type)
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                } else {
                                    Text(event.occurredAt.formatted(.relative(presentation: .named)))
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                            }
                            Text(AgentEventPresenter.displaySummary(for: event))
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                        .padding(.vertical, 4)
                    }
                }
            } else {
                Text("Select an agent to inspect status, project, and recent events.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Spacer()
        }
    }
}

private struct EventSummaryChipsView: View {
    let chips: [AgentEventSummaryChip]

    var body: some View {
        FlexibleChipRow(items: chips) { chip in
            HStack(spacing: 4) {
                Text(chip.label)
                Text("\(chip.count)")
                    .foregroundStyle(.secondary)
            }
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background(eventBadgeBackground(for: chip.emphasis))
            .clipShape(Capsule())
        }
    }
}

private struct FlexibleChipRow<Item, Content: View>: View {
    let items: [Item]
    let content: (Item) -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            ForEach(Array(items.enumerated()), id: \.offset) { _, item in
                content(item)
            }
        }
    }
}

private func eventBadgeBackground(for emphasis: AgentEventEmphasis) -> Color {
    switch emphasis {
    case .positive:
        return .green.opacity(0.18)
    case .warning:
        return .orange.opacity(0.18)
    case .info:
        return .blue.opacity(0.18)
    case .neutral:
        return .gray.opacity(0.18)
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
