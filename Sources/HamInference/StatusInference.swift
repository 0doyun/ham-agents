import Foundation
import HamCore

public struct InferenceSignal: Equatable, Sendable {
    public var hasStructuredEvent: Bool
    public var silenceDuration: TimeInterval
    public var lastOutputPreview: String

    public init(
        hasStructuredEvent: Bool,
        silenceDuration: TimeInterval,
        lastOutputPreview: String
    ) {
        self.hasStructuredEvent = hasStructuredEvent
        self.silenceDuration = silenceDuration
        self.lastOutputPreview = lastOutputPreview
    }
}

public struct InferenceResult: Equatable, Sendable {
    public var status: AgentStatus
    public var confidence: Double
    public var reason: String

    public init(status: AgentStatus, confidence: Double, reason: String) {
        self.status = status
        self.confidence = confidence
        self.reason = reason
    }
}

public struct StatusInferenceEngine {
    public init() {}

    public func infer(from signal: InferenceSignal) -> InferenceResult {
        if signal.lastOutputPreview.localizedCaseInsensitiveContains("error") {
            return InferenceResult(status: .error, confidence: 0.85, reason: "Error-like output detected.")
        }

        if signal.lastOutputPreview.contains("?") && signal.silenceDuration > 15 {
            return InferenceResult(status: .waitingInput, confidence: 0.72, reason: "Question-like output followed by silence.")
        }

        if signal.hasStructuredEvent {
            return InferenceResult(status: .runningTool, confidence: 0.9, reason: "Structured runtime event available.")
        }

        return InferenceResult(status: .thinking, confidence: 0.45, reason: "Fallback heuristic inferred active work.")
    }
}
