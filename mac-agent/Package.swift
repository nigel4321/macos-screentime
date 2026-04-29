// swift-tools-version:5.9
import PackageDescription

// Targets and products are populated as modules are scaffolded in later
// ROADMAP.md sub-sections:
//   §1.3 — PolicyEngine
//   §1.7 — LocalStore
//   §1.8 — UsageCollector
//   §2.10 — SyncClient
//   §3.4 — Enforcer
//   §3.5 — PolicySubscriber
//
// The Xcode app target (mac-only, pending macOS environment) will depend on
// this package as a local Swift Package reference.
let package = Package(
    name: "MacAgentCore",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .library(name: "PolicyEngine", targets: ["PolicyEngine"]),
        .library(name: "LocalStore", targets: ["LocalStore"]),
        .library(name: "UsageCollector", targets: ["UsageCollector"]),
        .library(name: "SyncClient", targets: ["SyncClient"]),
        .library(name: "AppMetadata", targets: ["AppMetadata"])
    ],
    dependencies: [
        // SwiftLint is attached per-target as a build-tool plugin so every
        // `swift build` / `swift test` lints. We use SwiftLintPlugins (binary
        // distribution) instead of realm/SwiftLint directly because Swift 6
        // forbids prebuild commands from using executables built from source.
        .package(url: "https://github.com/SimplyDanny/SwiftLintPlugins", from: "0.57.0"),
        .package(url: "https://github.com/groue/GRDB.swift", from: "6.0.0")
    ],
    targets: [
        .target(
            name: "PolicyEngine",
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .testTarget(
            name: "PolicyEngineTests",
            dependencies: ["PolicyEngine"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .target(
            name: "LocalStore",
            dependencies: [
                "PolicyEngine",
                .product(name: "GRDB", package: "GRDB.swift")
            ],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .testTarget(
            name: "LocalStoreTests",
            dependencies: ["LocalStore", "PolicyEngine"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .target(
            name: "UsageCollector",
            dependencies: ["PolicyEngine"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .testTarget(
            name: "UsageCollectorTests",
            dependencies: ["UsageCollector", "PolicyEngine"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .target(
            name: "SyncClient",
            dependencies: ["PolicyEngine", "LocalStore"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .testTarget(
            name: "SyncClientTests",
            dependencies: ["SyncClient", "PolicyEngine", "LocalStore"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .target(
            name: "AppMetadata",
            dependencies: ["PolicyEngine"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        ),
        .testTarget(
            name: "AppMetadataTests",
            dependencies: ["AppMetadata", "PolicyEngine"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLintPlugins")
            ]
        )
    ]
)
