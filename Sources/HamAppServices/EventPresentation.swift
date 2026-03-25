import Foundation
import HamCore

public enum AgentEventEmphasis: String, Equatable, Sendable {
    case neutral
    case positive
    case warning
    case info
}

public struct AgentEventPresentation: Equatable, Sendable {
    public var label: String
    public var emphasis: AgentEventEmphasis
    public var showsTechnicalType: Bool

    public init(label: String, emphasis: AgentEventEmphasis, showsTechnicalType: Bool = false) {
        self.label = label
        self.emphasis = emphasis
        self.showsTechnicalType = showsTechnicalType
    }
}

public struct AgentEventSummaryChip: Equatable, Sendable, Identifiable {
    public var id: String { "\(label)-\(emphasis.rawValue)" }
    public var label: String
    public var emphasis: AgentEventEmphasis
    public var count: Int

    public init(label: String, emphasis: AgentEventEmphasis, count: Int) {
        self.label = label
        self.emphasis = emphasis
        self.count = count
    }
}

public enum AgentEventPresenter {
    public static func present(_ event: AgentEventPayload) -> AgentEventPresentation {
        if let hinted = hintedPresentation(for: event) {
            return softenedIfLowConfidence(hinted, event: event)
        }
        if let metadata = metadataPresentation(for: event) {
            return softenedIfLowConfidence(metadata, event: event)
        }
        switch event.type {
        case "agent.registered":
            return softenedIfLowConfidence(presentRegisteredEvent(event), event: event)
        case "agent.role_updated":
            return softenedIfLowConfidence(AgentEventPresentation(label: "Role", emphasis: .info), event: event)
        case "agent.notification_policy_updated":
            return softenedIfLowConfidence(AgentEventPresentation(label: "Notifications", emphasis: .info), event: event)
        case "agent.status_updated":
            return softenedIfLowConfidence(presentStatusUpdatedEvent(event), event: event)
        case "agent.disconnected":
            return softenedIfLowConfidence(AgentEventPresentation(label: "Disconnected", emphasis: .warning), event: event)
        case "agent.reconnected":
            return softenedIfLowConfidence(AgentEventPresentation(label: "Reconnected", emphasis: .positive), event: event)
        case "agent.removed":
            return softenedIfLowConfidence(AgentEventPresentation(label: "Stopped", emphasis: .neutral), event: event)
        default:
            return softenedIfLowConfidence(AgentEventPresentation(label: event.type, emphasis: .neutral, showsTechnicalType: true), event: event)
        }
    }

    public static func summarize(_ events: [AgentEventPayload]) -> [AgentEventSummaryChip] {
        var buckets: [String: AgentEventSummaryChip] = [:]

        for event in events {
            let presentation = present(event)
            let key = "\(presentation.label)-\(presentation.emphasis.rawValue)"
            if var existing = buckets[key] {
                existing.count += 1
                buckets[key] = existing
            } else {
                buckets[key] = AgentEventSummaryChip(
                    label: presentation.label,
                    emphasis: presentation.emphasis,
                    count: 1
                )
            }
        }

        return buckets.values.sorted {
            let lhsPriority = sortPriority($0.emphasis)
            let rhsPriority = sortPriority($1.emphasis)
            if lhsPriority == rhsPriority {
                if $0.count == $1.count {
                    return $0.label < $1.label
                }
                return $0.count > $1.count
            }
            return lhsPriority < rhsPriority
        }
    }

    public static func summarizeBySeverity(_ events: [AgentEventPayload]) -> [AgentEventSummaryChip] {
        var counts: [AgentEventEmphasis: Int] = [:]

        for event in events {
            let emphasis = present(event).emphasis
            counts[emphasis, default: 0] += 1
        }

        return counts.keys.sorted { sortPriority($0) < sortPriority($1) }.compactMap { emphasis in
            guard let count = counts[emphasis], count > 0 else { return nil }
            return AgentEventSummaryChip(
                label: severityLabel(for: emphasis),
                emphasis: emphasis,
                count: count
            )
        }
    }

    public static func ordered(_ events: [AgentEventPayload]) -> [AgentEventPayload] {
        events.sorted { lhs, rhs in
            let lhsPriority = sortPriority(present(lhs).emphasis)
            let rhsPriority = sortPriority(present(rhs).emphasis)
            if lhsPriority == rhsPriority {
                if lhs.occurredAt == rhs.occurredAt {
                    return lhs.id > rhs.id
                }
                return lhs.occurredAt > rhs.occurredAt
            }
            return lhsPriority < rhsPriority
        }
    }

    public static func displaySummary(for event: AgentEventPayload) -> String {
        if let summary = event.presentationSummary?.trimmingCharacters(in: .whitespacesAndNewlines), !summary.isEmpty {
            return summary
        }
        if let reason = event.lifecycleReason?.trimmingCharacters(in: .whitespacesAndNewlines), !reason.isEmpty {
            if let confidence = event.lifecycleConfidence, confidence < 0.5 {
                return "\(reason) (low confidence)"
            }
            return reason
        }
        return event.summary
    }

    private static func sortPriority(_ emphasis: AgentEventEmphasis) -> Int {
        switch emphasis {
        case .warning:
            return 0
        case .positive:
            return 1
        case .info:
            return 2
        case .neutral:
            return 3
        }
    }

    private static func severityLabel(for emphasis: AgentEventEmphasis) -> String {
        switch emphasis {
        case .warning:
            return "Needs Attention"
        case .positive:
            return "Positive"
        case .info:
            return "Info"
        case .neutral:
            return "Other"
        }
    }

    private static func presentRegisteredEvent(_ event: AgentEventPayload) -> AgentEventPresentation {
        let summary = event.summary.lowercased()
        if summary.contains("attached session registered") {
            return AgentEventPresentation(label: "Attached", emphasis: .info)
        }
        if summary.contains("observed source registered") {
            return AgentEventPresentation(label: "Observed", emphasis: .info)
        }
        return AgentEventPresentation(label: "Managed", emphasis: .info)
    }

    private static func presentStatusUpdatedEvent(_ event: AgentEventPayload) -> AgentEventPresentation {
        let summary = event.summary.lowercased()
        switch true {
        case summary.contains("status changed to error"):
            return AgentEventPresentation(label: "Error", emphasis: .warning)
        case summary.contains("status changed to waiting_input"):
            return AgentEventPresentation(label: "Needs Input", emphasis: .warning)
        case summary.contains("status changed to done"):
            return AgentEventPresentation(label: "Done", emphasis: .positive)
        case summary.contains("status changed to disconnected"):
            return AgentEventPresentation(label: "Disconnected", emphasis: .warning)
        case summary.contains("status changed to idle"):
            return AgentEventPresentation(label: "Idle", emphasis: .info)
        default:
            return AgentEventPresentation(label: "Status", emphasis: .info)
        }
    }

    private static func hintedPresentation(for event: AgentEventPayload) -> AgentEventPresentation? {
        guard
            let label = event.presentationLabel?.trimmingCharacters(in: .whitespacesAndNewlines),
            !label.isEmpty,
            let rawEmphasis = event.presentationEmphasis?.trimmingCharacters(in: .whitespacesAndNewlines),
            let emphasis = AgentEventEmphasis(rawValue: rawEmphasis)
        else {
            return nil
        }

        return AgentEventPresentation(label: label, emphasis: emphasis)
    }

    private static func softenedIfLowConfidence(
        _ presentation: AgentEventPresentation,
        event: AgentEventPayload
    ) -> AgentEventPresentation {
        guard let confidence = event.lifecycleConfidence, confidence < 0.5 else {
            return presentation
        }
        guard presentation.label.hasPrefix("Likely ") == false else {
            return presentation
        }
        return AgentEventPresentation(
            label: "Likely \(presentation.label)",
            emphasis: presentation.emphasis,
            showsTechnicalType: presentation.showsTechnicalType
        )
    }

    private static func metadataPresentation(for event: AgentEventPayload) -> AgentEventPresentation? {
        switch event.type {
        case "agent.registered":
            switch event.lifecycleMode {
            case "attached":
                return AgentEventPresentation(label: "Attached", emphasis: .info)
            case "observed":
                return AgentEventPresentation(label: "Observed", emphasis: .info)
            case "managed":
                return AgentEventPresentation(label: "Managed", emphasis: .info)
            default:
                return nil
            }
        case "agent.status_updated", "agent.disconnected", "agent.reconnected":
            switch event.lifecycleStatus {
            case "error":
                return AgentEventPresentation(label: "Error", emphasis: .warning)
            case "waiting_input":
                return AgentEventPresentation(label: "Needs Input", emphasis: .warning)
            case "done":
                return AgentEventPresentation(label: "Done", emphasis: .positive)
            case "disconnected":
                return AgentEventPresentation(label: "Disconnected", emphasis: .warning)
            case "idle":
                return AgentEventPresentation(label: "Idle", emphasis: .info)
            default:
                return nil
            }
        default:
            return nil
        }
    }
}
