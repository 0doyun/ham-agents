import Foundation
import HamCore
import HamPersistence
import HamRuntime

enum CLIError: Error {
    case unsupportedCommand(String)
}

let store = InMemoryAgentStore()
let runtime = RuntimeRegistry(store: store)

func makeManagedAgent(name: String, provider: String) -> Agent {
    Agent(
        id: UUID().uuidString,
        displayName: name,
        provider: provider,
        host: Host.current().localizedName ?? "localhost",
        mode: .managed,
        projectPath: FileManager.default.currentDirectoryPath,
        status: .booting,
        statusConfidence: 1.0,
        lastEventAt: Date(),
        lastUserVisibleSummary: "Managed session registered."
    )
}

func printHelp() {
    print(
        """
        ham-agents bootstrap CLI

        Usage:
          ham help
          ham run [provider] [name]
          ham list
          ham status

        Note:
          This is a bootstrap surface for the first implementation slices.
        """
    )
}

func handle(command: [String]) throws {
    let subcommand = command.first ?? "help"

    switch subcommand {
    case "help":
        printHelp()
    case "run":
        let provider = command.dropFirst().first ?? "unknown"
        let name = command.dropFirst(2).first ?? "managed-agent"
        let agent = makeManagedAgent(name: name, provider: provider)
        runtime.register(agent)
        print("registered \(agent.displayName) [\(agent.id)] via \(agent.provider)")
    case "list":
        let agents = runtime.snapshot().agents
        if agents.isEmpty {
            print("no tracked agents")
            return
        }

        for agent in agents {
            print("\(agent.displayName)\t\(agent.status.rawValue)\t\(agent.mode.rawValue)")
        }
    case "status":
        let snapshot = runtime.snapshot()
        print("total=\(snapshot.totalCount) running=\(snapshot.runningCount) waiting=\(snapshot.waitingCount) done=\(snapshot.doneCount)")
    default:
        throw CLIError.unsupportedCommand(subcommand)
    }
}

do {
    try handle(command: Array(CommandLine.arguments.dropFirst()))
} catch CLIError.unsupportedCommand(let subcommand) {
    fputs("unsupported command: \(subcommand)\n", stderr)
    printHelp()
    exit(1)
} catch {
    fputs("unexpected error: \(error)\n", stderr)
    exit(1)
}
