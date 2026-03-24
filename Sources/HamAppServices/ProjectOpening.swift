import Foundation
import HamCore

public protocol ProjectOpening: Sendable {
    func openProject(at path: String)
}

public struct NoopProjectOpener: ProjectOpening {
    public init() {}
    public func openProject(at path: String) {
        _ = path
    }
}
