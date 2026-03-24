// swift-tools-version: 5.10
import PackageDescription

let package = Package(
    name: "ham-agents",
    platforms: [
        .macOS(.v13),
    ],
    products: [
        .library(name: "HamCore", targets: ["HamCore"]),
        .library(name: "HamRuntime", targets: ["HamRuntime"]),
        .library(name: "HamPersistence", targets: ["HamPersistence"]),
        .library(name: "HamInference", targets: ["HamInference"]),
        .library(name: "HamNotifications", targets: ["HamNotifications"]),
        .library(name: "HamAdapters", targets: ["HamAdapters"]),
        .executable(name: "ham", targets: ["HamCLI"]),
    ],
    targets: [
        .target(
            name: "HamCore",
            path: "Sources/HamCore"
        ),
        .target(
            name: "HamPersistence",
            dependencies: ["HamCore"],
            path: "Sources/HamPersistence"
        ),
        .target(
            name: "HamRuntime",
            dependencies: ["HamCore", "HamPersistence"],
            path: "Sources/HamRuntime"
        ),
        .target(
            name: "HamInference",
            dependencies: ["HamCore"],
            path: "Sources/HamInference"
        ),
        .target(
            name: "HamNotifications",
            dependencies: ["HamCore"],
            path: "Sources/HamNotifications"
        ),
        .target(
            name: "HamAdapters",
            dependencies: ["HamCore"],
            path: "Sources/HamAdapters"
        ),
        .executableTarget(
            name: "HamCLI",
            dependencies: ["HamCore", "HamPersistence", "HamRuntime"],
            path: "Sources/HamCLI"
        ),
        .testTarget(
            name: "HamCoreTests",
            dependencies: ["HamCore"],
            path: "Tests/HamCoreTests"
        ),
        .testTarget(
            name: "HamRuntimeTests",
            dependencies: ["HamRuntime", "HamPersistence", "HamCore"],
            path: "Tests/HamRuntimeTests"
        ),
    ]
)
