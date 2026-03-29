import AppKit
import SwiftUI
import HamAppServices
import HamCore
import HamNotifications

class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)
    }
}

@MainActor
final class HamOfficeWindowPresenter {
    static let shared = HamOfficeWindowPresenter()

    private var window: NSWindow?

    func show(viewModel: MenuBarViewModel) {
        if let window {
            window.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }

        let contentView = MenuBarContentView(viewModel: viewModel)
            .frame(minWidth: 380, minHeight: 400)
        let hostingController = NSHostingController(rootView: contentView)

        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 420, height: 520),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Ham Office"
        window.isReleasedWhenClosed = false
        window.contentViewController = hostingController
        window.center()
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        self.window = window
    }
}

@main
struct HamMenuBarApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    @StateObject private var viewModel = HamMenuBarApp.makeViewModel()

    var body: some Scene {
        MenuBarExtra {
            MenuBarContentView(viewModel: viewModel)
                .frame(minWidth: 380, minHeight: 400)
                .task {
                    await viewModel.refresh()
                }
        } label: {
            MenuBarHamsterGlyph(
                state: viewModel.menuBarHamsterState,
                animationSpeed: viewModel.settings.appearance.animationSpeedMultiplier,
                reduceMotion: viewModel.settings.appearance.reduceMotion,
                hamsterSkin: viewModel.settings.appearance.hamsterSkin,
                hat: viewModel.settings.appearance.hat
            )
        }
        .menuBarExtraStyle(.window)
    }

    private static func makeViewModel() -> MenuBarViewModel {
        let client: HamDaemonClientProtocol
        if let transport = try? UnixSocketDaemonTransport() {
            let socketPath = (try? DaemonEnvironment.defaultSocketPath()) ?? "unknown"
            NSLog("[ham-menubar] using socket: %@", socketPath)
            client = HamDaemonClient(transport: transport)
        } else {
            NSLog("[ham-menubar] falling back to PreviewDaemonClient")
            client = PreviewDaemonClient()
        }
        let center: UserNotificationCentering = LiveUserNotificationCenter.makeIfAvailable() ?? NoopUserNotificationCenter()
        let notificationSink = UserNotificationSink(center: center)
        let projectOpener = WorkspaceProjectOpener()
        let sessionOpener = ItermSessionOpener(projectOpener: projectOpener)
        let viewModel = MenuBarViewModel(
            client: client,
            notificationSink: notificationSink,
            notificationPermissionController: notificationSink,
            projectOpener: projectOpener,
            sessionOpener: sessionOpener,
            quickMessageSender: ItermQuickMessageSender(
                sessionOpener: sessionOpener,
                projectOpener: projectOpener
            )
        )
        notificationSink.setInteractionHandler { interaction in
            Task { @MainActor in
                NSApp.activate(ignoringOtherApps: true)
                viewModel.handleNotificationInteraction(interaction)
                HamOfficeWindowPresenter.shared.show(viewModel: viewModel)
            }
        }
        viewModel.start()
        return viewModel
    }
}
