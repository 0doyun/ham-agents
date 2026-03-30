import SwiftUI
import HamAppServices
import HamCore
import HamNotifications

struct MenuBarContentView: View {
    @ObservedObject var viewModel: MenuBarViewModel
    @State private var selectedAgentID: Agent.ID?
    @State private var quickMessage = ""
    @State private var selectedTeamID = ""
    @State private var selectedWorkspace = ""
    @State private var showSettings = false

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Ham Office")
                    .font(.headline)
                Spacer()
                if viewModel.isRefreshing {
                    ProgressView()
                        .controlSize(.small)
                }
                Button {
                    Task { await viewModel.refresh() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .buttonStyle(.borderless)
                Button {
                    showSettings.toggle()
                } label: {
                    Image(systemName: showSettings ? "gearshape.fill" : "gearshape")
                }
                .buttonStyle(.borderless)
            }
            .padding(.horizontal, 14)
            .padding(.top, 14)
            .padding(.bottom, 8)

            if showSettings {
                settingsContent
            } else {
                officeContent
            }
        }
        .onAppear {
            if selectedAgentID == nil {
                selectedAgentID = viewModel.selectedAgentID ?? viewModel.agents.first?.id
            }
            viewModel.selectedAgentID = selectedAgentID
            viewModel.setRoleDraft(from: selectedAgentID)
        }
        .onChange(of: viewModel.agents.map(\.id)) { ids in
            if selectedAgentID == nil || !ids.contains(selectedAgentID ?? "") {
                selectedAgentID = viewModel.selectedAgentID ?? ids.first
            }
            viewModel.selectedAgentID = selectedAgentID
            viewModel.setRoleDraft(from: selectedAgentID)
        }
        .onChange(of: viewModel.selectedAgentID) { selectedAgentID = $0 }
    }

    private var officeContent: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                if let summary = viewModel.summary {
                    HStack {
                        SummaryBadge(title: "Total", value: summary.totalAgents)
                        SummaryBadge(title: "Run", value: summary.runningAgents)
                        SummaryBadge(title: "Wait", value: summary.waitingAgents)
                    }
                }

                if let errorMessage = viewModel.errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                let filteredAttentionAgents = viewModel.filteredAttentionAgents(teamID: selectedTeamID, workspace: selectedWorkspace)
                let filteredNonAttentionAgents = viewModel.filteredNonAttentionAgents(teamID: selectedTeamID, workspace: selectedWorkspace)
                let filteredOfficeAgents = filteredAttentionAgents + filteredNonAttentionAgents
                let filteredOfficeOccupants = PixelOfficeMapper.occupants(from: filteredOfficeAgents)

                PixelOfficeView(
                    occupants: filteredOfficeOccupants,
                    animationSpeedMultiplier: viewModel.settings.appearance.animationSpeedMultiplier,
                    reduceMotion: viewModel.settings.appearance.reduceMotion,
                    hamsterSkin: viewModel.settings.appearance.hamsterSkin,
                    hat: viewModel.settings.appearance.hat,
                    deskTheme: viewModel.settings.appearance.deskTheme,
                    onSelectAgent: { id in
                        selectedAgentID = id
                        viewModel.selectedAgentID = id
                    }
                )

                if !filteredAttentionAgents.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Needs Attention")
                            .font(.caption.weight(.semibold))
                        ForEach(filteredAttentionAgents) { agent in
                            AttentionAgentRow(
                                name: agent.displayName,
                                subtitle: viewModel.attentionSubtitle(for: agent)
                            )
                        }
                    }
                }

                if filteredAttentionAgents.isEmpty && filteredNonAttentionAgents.isEmpty {
                    Text("No tracked agents")
                        .foregroundStyle(.secondary)
                } else {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(filteredNonAttentionAgents) { agent in
                            Button {
                                selectedAgentID = agent.id
                                viewModel.selectedAgentID = agent.id
                            } label: {
                                VStack(alignment: .leading, spacing: 2) {
                                    HStack(spacing: 6) {
                                        Text(agent.displayName)
                                            .font(.body.weight(.medium))
                                        if let teamRole = agent.teamRole, !teamRole.isEmpty {
                                            TeamRoleBadge(role: teamRole)
                                        }
                                        if let omcMode = agent.omcMode, !omcMode.isEmpty {
                                            OmcModeBadge(mode: omcMode)
                                        }
                                        if let progress = teamTaskProgressText(for: agent) {
                                            Text(progress)
                                                .font(.caption2.monospacedDigit())
                                                .foregroundStyle(.secondary)
                                        }
                                    }
                                    Text("\(viewModel.statusDisplayText(for: agent)) · \(agent.mode.rawValue) · \(viewModel.confidenceLevelText(for: agent)) \(viewModel.confidenceText(for: agent))")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                        .lineLimit(1)
                                    if let summary = agent.lastUserVisibleSummary, !summary.isEmpty {
                                        Text(summary)
                                            .font(.caption2)
                                            .foregroundStyle(.secondary.opacity(0.7))
                                            .lineLimit(2)
                                    }
                                }
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(6)
                                .background(selectedAgentID == agent.id ? Color.accentColor.opacity(0.15) : Color.clear)
                                .clipShape(RoundedRectangle(cornerRadius: 6))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }

                if selectedAgentID != nil {
                    let selectedAgent = viewModel.agent(withID: selectedAgentID)
                    Divider()
                    AgentDetailView(
                        agent: selectedAgent,
                        recentEvents: viewModel.recentEvents(forAgentID: selectedAgentID),
                        recentEventSummaryChips: viewModel.recentEventSummaryChips(forAgentID: selectedAgentID),
                        notificationsMuted: viewModel.isNotificationsMuted(forAgentID: selectedAgentID),
                        quickMessageFeedback: viewModel.quickMessageFeedback,
                        confidenceText: viewModel.confidenceSummaryText(for: selectedAgent),
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
                        openSessionLabel: sessionOpenLabel(for: selectedAgent),
                        canOpenSession: canOpenSession(for: selectedAgent, itermEnabled: viewModel.settings.integrations.itermEnabled),
                        toggleNotifications: {
                            viewModel.toggleNotificationPause(forAgentID: selectedAgentID)
                        },
                        saveRole: {
                            await viewModel.saveRole(forAgentID: selectedAgentID)
                        },
                        stopTracking: {
                            await viewModel.stopTracking(forAgentID: selectedAgentID)
                            selectedAgentID = viewModel.agents.first?.id
                            viewModel.selectedAgentID = selectedAgentID
                        },
                        sendQuickMessage: {
                            viewModel.sendQuickMessage(quickMessage, forAgentID: selectedAgentID)
                            quickMessage = ""
                        }
                    )
                }
            }
            .padding(.horizontal, 14)
            .padding(.bottom, 14)
        }
    }

    private var settingsContent: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                NotificationPermissionRow(
                    status: viewModel.notificationPermissionStatus,
                    requestPermission: {
                        Task { await viewModel.requestNotificationPermission() }
                    }
                )

                GeneralSettingsSection(
                    settings: viewModel.settings.general,
                    updateLaunchAtLogin: { value in
                        Task { await viewModel.updateGeneralSetting(launchAtLogin: value) }
                    },
                    updateCompactMode: { value in
                        Task { await viewModel.updateGeneralSetting(compactMode: value) }
                    },
                    updateShowMenuBarAnimationAlways: { value in
                        Task { await viewModel.updateGeneralSetting(showMenuBarAnimationAlways: value) }
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
                    },
                    updateHeartbeatMinutes: { value in
                        Task { await viewModel.updateNotificationSetting(heartbeatMinutes: value) }
                    }
                )

                AppearanceSettingsSection(
                    settings: viewModel.settings.appearance,
                    updateTheme: { value in
                        Task { await viewModel.updateAppearanceSetting(theme: value) }
                    },
                    updateAnimationSpeed: { value in
                        Task { await viewModel.updateAppearanceSetting(animationSpeedMultiplier: value) }
                    },
                    updateReduceMotion: { value in
                        Task { await viewModel.updateAppearanceSetting(reduceMotion: value) }
                    },
                    updateHamsterSkin: { value in
                        Task { await viewModel.updateAppearanceSetting(hamsterSkin: value) }
                    },
                    updateHat: { value in
                        Task { await viewModel.updateAppearanceSetting(hat: value) }
                    },
                    updateDeskTheme: { value in
                        Task { await viewModel.updateAppearanceSetting(deskTheme: value) }
                    }
                )

                IntegrationSettingsSection(
                    settings: viewModel.settings.integrations,
                    updateItermEnabled: { value in
                        Task { await viewModel.updateIntegrationSetting(itermEnabled: value) }
                    },
                    updateTranscriptDirs: { value in
                        Task {
                            let dirs = value.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
                            await viewModel.updateIntegrationSetting(transcriptDirs: dirs)
                        }
                    },
                    updateProviderAdapter: { key, value in
                        Task {
                            var adapters = viewModel.settings.integrations.providerAdapters
                            adapters[key] = value
                            await viewModel.updateIntegrationSetting(providerAdapters: adapters)
                        }
                    }
                )

                PrivacySettingsSection(
                    settings: viewModel.settings.privacy,
                    updateLocalOnlyMode: { value in
                        Task { await viewModel.updatePrivacySetting(localOnlyMode: value) }
                    },
                    updateEventHistoryRetentionDays: { value in
                        Task { await viewModel.updatePrivacySetting(eventHistoryRetentionDays: value) }
                    },
                    updateTranscriptExcerptStorage: { value in
                        Task { await viewModel.updatePrivacySetting(transcriptExcerptStorage: value) }
                    }
                )

                if viewModel.settings.integrations.itermEnabled && !viewModel.attachableSessions.isEmpty {
                    AttachableSessionsSection(sessions: viewModel.attachableSessions)
                }
            }
            .padding(.horizontal, 14)
            .padding(.bottom, 14)
        }
    }
}

private func sessionOpenLabel(for agent: Agent?) -> String {
    guard let sessionRef = agent?.sessionRef else {
        return "Open in iTerm"
    }
    if sessionRef.hasPrefix("tmux://") {
        return "Open in tmux"
    }
    return "Open in iTerm"
}

private func canOpenSession(for agent: Agent?, itermEnabled: Bool) -> Bool {
    guard let agent else { return false }
    if let sessionRef = agent.sessionRef, !sessionRef.isEmpty {
        return true
    }
    return itermEnabled
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
    let updateHeartbeatMinutes: (Int) -> Void

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
            Picker(
                "Heartbeat",
                selection: Binding(
                    get: { settings.heartbeatMinutes },
                    set: updateHeartbeatMinutes
                )
            ) {
                Text("Off").tag(0)
                Text("10 min").tag(10)
                Text("30 min").tag(30)
                Text("60 min").tag(60)
            }
            .labelsHidden()
        }
        .toggleStyle(.checkbox)
    }

    private func hourLabel(_ hour: Int) -> String {
        String(format: "%02d:00", hour)
    }
}

private struct GeneralSettingsSection: View {
    let settings: DaemonGeneralSettingsPayload
    let updateLaunchAtLogin: (Bool) -> Void
    let updateCompactMode: (Bool) -> Void
    let updateShowMenuBarAnimationAlways: (Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("General")
                .font(.caption.weight(.semibold))

            Toggle("Launch at Login", isOn: Binding(get: { settings.launchAtLogin }, set: updateLaunchAtLogin))
            Toggle("Compact Mode", isOn: Binding(get: { settings.compactMode }, set: updateCompactMode))
            Toggle(
                "Always Animate Menu Bar",
                isOn: Binding(get: { settings.showMenuBarAnimationAlways }, set: updateShowMenuBarAnimationAlways)
            )
        }
        .toggleStyle(.checkbox)
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
    let updateTranscriptDirs: (String) -> Void
    let updateProviderAdapter: (String, Bool) -> Void
    @State private var transcriptDirsDraft = ""

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

            TextField(
                "Transcript directories (comma-separated)",
                text: Binding(
                    get: { transcriptDirsDraft.isEmpty ? settings.transcriptDirs.joined(separator: ", ") : transcriptDirsDraft },
                    set: {
                        transcriptDirsDraft = $0
                        updateTranscriptDirs($0)
                    }
                )
            )
            .textFieldStyle(.roundedBorder)

            ForEach(Array(settings.providerAdapters.keys.sorted()), id: \.self) { key in
                Toggle(
                    key,
                    isOn: Binding(
                        get: { settings.providerAdapters[key] ?? false },
                        set: { updateProviderAdapter(key, $0) }
                    )
                )
            }
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
    let updateAnimationSpeed: (Double) -> Void
    let updateReduceMotion: (Bool) -> Void
    let updateHamsterSkin: (String) -> Void
    let updateHat: (String) -> Void
    let updateDeskTheme: (String) -> Void

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

            HStack {
                Text("Animation")
                Slider(
                    value: Binding(
                        get: { settings.animationSpeedMultiplier },
                        set: updateAnimationSpeed
                    ),
                    in: 0.25 ... 3,
                    step: 0.25
                )
                Text(String(format: "%.2fx", settings.animationSpeedMultiplier))
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Toggle(
                "Reduce Motion",
                isOn: Binding(
                    get: { settings.reduceMotion },
                    set: updateReduceMotion
                )
            )

            Picker("Skin", selection: Binding(get: { settings.hamsterSkin }, set: updateHamsterSkin)) {
                Text("Default").tag("default")
                Text("Night").tag("night")
                Text("Golden").tag("golden")
            }
            .labelsHidden()

            Picker("Hat", selection: Binding(get: { settings.hat }, set: updateHat)) {
                Text("None").tag("none")
                Text("Cap").tag("cap")
                Text("Beanie").tag("beanie")
            }
            .labelsHidden()

            Picker("Desk Theme", selection: Binding(get: { settings.deskTheme }, set: updateDeskTheme)) {
                Text("Classic").tag("classic")
                Text("Night").tag("night")
                Text("Forest").tag("forest")
            }
            .labelsHidden()
        }
    }
}

private struct PrivacySettingsSection: View {
    let settings: DaemonPrivacySettingsPayload
    let updateLocalOnlyMode: (Bool) -> Void
    let updateEventHistoryRetentionDays: (Int) -> Void
    let updateTranscriptExcerptStorage: (Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Privacy")
                .font(.caption.weight(.semibold))

            Toggle("Local Only Mode", isOn: Binding(get: { settings.localOnlyMode }, set: updateLocalOnlyMode))
            Toggle(
                "Store Transcript Excerpts",
                isOn: Binding(get: { settings.transcriptExcerptStorage }, set: updateTranscriptExcerptStorage)
            )

            HStack {
                Text("History Retention")
                Spacer()
                Text("\(settings.eventHistoryRetentionDays)d")
                    .foregroundStyle(.secondary)
                Stepper(
                    "",
                    value: Binding(
                        get: { settings.eventHistoryRetentionDays },
                        set: updateEventHistoryRetentionDays
                    ),
                    in: 1...365
                )
                .labelsHidden()
            }
        }
        .toggleStyle(.checkbox)
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
    let openSessionLabel: String
    let canOpenSession: Bool
    let toggleNotifications: () -> Void
    let saveRole: () async -> Void
    let stopTracking: () async -> Void
    let sendQuickMessage: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            if let agent {
                // Header: name + status badge
                HStack(alignment: .center, spacing: 8) {
                    Text(agent.displayName)
                        .font(.headline)
                    if let teamRole = agent.teamRole, !teamRole.isEmpty {
                        TeamRoleBadge(role: teamRole)
                    }
                    if let omcMode = agent.omcMode, !omcMode.isEmpty {
                        OmcModeBadge(mode: omcMode)
                    }
                    StatusBadge(status: agent.status)
                    Spacer()
                }

                // One-line meta
                Text("\(agent.provider) · \(agent.mode.rawValue) · \(agent.projectPath)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)

                if let errorType = agent.errorType, !errorType.isEmpty {
                    Text("Error Type: \(errorType)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                } else if let reason = agent.statusReason, !reason.isEmpty {
                    Text(reason)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                }

                if let teamRole = agent.teamRole, !teamRole.isEmpty {
                    Text("Team Role: \(teamRoleLabel(for: teamRole))")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }

                if let progress = teamTaskProgress(for: agent) {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text("Team Tasks")
                                .font(.caption.weight(.semibold))
                            Spacer()
                            Text("\(progress.completed)/\(progress.total)")
                                .font(.caption.monospacedDigit())
                                .foregroundStyle(.secondary)
                        }
                        ProgressView(value: progress.fraction)
                            .controlSize(.small)
                    }
                }

                // Quick Message (top — most frequent action)
                HStack(spacing: 6) {
                    TextField("Quick message…", text: $quickMessage)
                        .textFieldStyle(.roundedBorder)
                    Button("Send") {
                        sendQuickMessage()
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(quickMessage.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
                if let quickMessageFeedback {
                    Text(quickMessageFeedback)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                // Action buttons row
                HStack(spacing: 6) {
                    Button(openSessionLabel) {
                        openSession()
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(!canOpenSession)

                    Button("Open Folder") {
                        openProject()
                    }
                    .buttonStyle(.bordered)

                    Menu {
                        // Role editing section
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

                        Divider()

                        Button(notificationsMuted ? "Resume Notifications" : "Pause Notifications") {
                            toggleNotifications()
                        }

                        Divider()

                        Button(role: .destructive) {
                            Task { await stopTracking() }
                        } label: {
                            Text("Stop Tracking")
                        }
                    } label: {
                        Text("⋯")
                            .frame(minWidth: 28)
                    }
                    .menuStyle(.borderlessButton)
                    .fixedSize()
                }

                // Recent Events (compact)
                if !agent.recentTools.isEmpty {
                    Text("Recent Activity")
                        .font(.caption.weight(.semibold))
                        .padding(.top, 2)
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(Array(agent.recentTools.prefix(5).enumerated()), id: \.offset) { _, entry in
                            Text(entry)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                                .lineLimit(2)
                        }
                    }
                }

                Text("Recent")
                    .font(.caption.weight(.semibold))
                    .padding(.top, 2)

                if recentEvents.isEmpty {
                    Text("No recent events.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(recentEvents.prefix(5)) { event in
                            let presentation = AgentEventPresenter.present(event)
                            HStack(spacing: 6) {
                                Circle()
                                    .fill(eventBadgeColor(for: presentation.emphasis))
                                    .frame(width: 7, height: 7)
                                Text(AgentEventPresenter.displaySummary(for: event))
                                    .font(.caption2)
                                    .lineLimit(1)
                                Spacer()
                                Text(event.occurredAt.formatted(.relative(presentation: .named)))
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
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

private struct OmcModeBadge: View {
    let mode: String

    var body: some View {
        Text("[\(mode)]")
            .font(.caption2.monospaced())
            .padding(.horizontal, 5)
            .padding(.vertical, 2)
            .background(Color.accentColor.opacity(0.12))
            .foregroundStyle(Color.accentColor)
            .clipShape(RoundedRectangle(cornerRadius: 6))
    }
}

private struct TeamRoleBadge: View {
    let role: String

    var body: some View {
        Label(teamRoleLabel(for: role), systemImage: role == "lead" ? "crown.fill" : "person.2.fill")
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 5)
            .padding(.vertical, 2)
            .background(role == "lead" ? Color.yellow.opacity(0.18) : Color.gray.opacity(0.18))
            .foregroundStyle(role == "lead" ? Color.yellow.opacity(0.95) : Color.secondary)
            .clipShape(Capsule())
    }
}

private struct StatusBadge: View {
    let status: AgentStatus

    var body: some View {
        Text(statusLabel)
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(statusColor.opacity(0.18))
            .foregroundStyle(statusColor)
            .clipShape(Capsule())
    }

    private var statusLabel: String {
        switch status {
        case .booting:       return "booting"
        case .thinking:      return "thinking"
        case .reading:       return "reading"
        case .runningTool:   return "running tool"
        case .waitingInput:  return "waiting input"
        case .error:         return "error"
        case .disconnected:  return "disconnected"
        case .done:          return "done"
        case .idle:          return "idle"
        case .sleeping:      return "sleeping"
        }
    }

    private var statusColor: Color {
        switch status {
        case .thinking, .reading, .runningTool, .booting:
            return .blue
        case .waitingInput:
            return .orange
        case .error, .disconnected:
            return .red
        case .done:
            return .green
        case .idle, .sleeping:
            return .gray
        }
    }
}

private func eventBadgeColor(for emphasis: AgentEventEmphasis) -> Color {
    switch emphasis {
    case .positive: return .green
    case .warning:  return .orange
    case .info:     return .blue
    case .neutral:  return .gray
    }
}

private func teamRoleLabel(for role: String) -> String {
    switch role {
    case "lead":
        return "Lead"
    case "teammate":
        return "Teammate"
    default:
        return role
    }
}

private func teamTaskProgress(for agent: Agent) -> (completed: Int, total: Int, fraction: Double)? {
    guard agent.teamTaskTotal > 0 else { return nil }
    let total = max(agent.teamTaskTotal, 0)
    let completed = min(max(agent.teamTaskCompleted, 0), total)
    guard total > 0 else { return nil }
    return (completed, total, Double(completed) / Double(total))
}

private func teamTaskProgressText(for agent: Agent) -> String? {
    guard let progress = teamTaskProgress(for: agent) else { return nil }
    return "\(progress.completed)/\(progress.total)"
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
