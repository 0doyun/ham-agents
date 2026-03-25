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
            HStack(spacing: 4) {
                MenuBarHamsterGlyph(
                    state: viewModel.menuBarHamsterState,
                    animationSpeed: viewModel.settings.appearance.animationSpeedMultiplier,
                    reduceMotion: viewModel.settings.appearance.reduceMotion,
                    hamsterSkin: viewModel.settings.appearance.hamsterSkin,
                    hat: viewModel.settings.appearance.hat
                )
                Text(viewModel.statusLine)
            }
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
        let notificationSink = UserNotificationSink()
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
        viewModel.start()
        return viewModel
    }
}
