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
    products: [],
    dependencies: [],
    targets: []
)
