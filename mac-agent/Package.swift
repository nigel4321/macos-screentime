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
        .library(name: "LocalStore", targets: ["LocalStore"])
    ],
    dependencies: [
        // SwiftLint is attached per-target as a build-tool plugin so every
        // `swift build` / `swift test` lints. See §1.3+ targets.
        .package(url: "https://github.com/realm/SwiftLint", from: "0.55.0"),
        .package(url: "https://github.com/groue/GRDB.swift", from: "6.0.0")
    ],
    targets: [
        .target(
            name: "PolicyEngine",
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLint")
            ]
        ),
        .testTarget(
            name: "PolicyEngineTests",
            dependencies: ["PolicyEngine"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLint")
            ]
        ),
        .target(
            name: "LocalStore",
            dependencies: [
                "PolicyEngine",
                .product(name: "GRDB", package: "GRDB.swift")
            ],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLint")
            ]
        ),
        .testTarget(
            name: "LocalStoreTests",
            dependencies: ["LocalStore", "PolicyEngine"],
            plugins: [
                .plugin(name: "SwiftLintBuildToolPlugin", package: "SwiftLint")
            ]
        )
    ]
)
