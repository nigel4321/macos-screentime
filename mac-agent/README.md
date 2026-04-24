# mac-agent

The macOS Screen Time agent. This directory is a Swift Package
(`Package.swift`) containing the core modules. The Xcode app target is
added on a macOS environment.

## Structure (target state)

```
mac-agent/
├── Package.swift             # Swift Package Manager manifest
├── Sources/                  # Library module sources (populated per ROADMAP)
│   ├── PolicyEngine/         # §1.3 — pure Swift, TDD-first
│   ├── LocalStore/           # §1.7 — GRDB persistence
│   ├── UsageCollector/       # §1.8 — NSWorkspace adapter (AppKit-guarded)
│   ├── SyncClient/           # §2.10 — REST + WebSocket client
│   ├── Enforcer/             # §3.4 — ManagedSettings adapter (mac-only build)
│   └── PolicySubscriber/     # §3.5 — WebSocket subscription
├── Tests/                    # XCTest targets, one per library module
├── .swiftlint.yml            # §1.2
└── App/                      # (mac-only) Xcode app target, added later
    └── MacAgent.xcodeproj
```

## Building and testing

Pure-Swift modules are portable; AppKit- and SwiftUI-dependent code
paths are guarded with `#if canImport(AppKit)` so Linux builds stay
green.

```sh
swift build
swift test
```

On macOS, additionally:

```sh
open MacAgent.xcworkspace   # after the Xcode app target is added
xcodebuild test             # same as `swift test` for library targets
```

## Pending Mac-only work

These `ROADMAP.md` items require macOS:

- §1.1 — Xcode workspace + app target creation, local signing
- §1.2 — SwiftLint Xcode build phase (config file itself is portable)
- §1.9 — App wiring (`MenuBarExtra`, `Info.plist`, `LSUIElement`)
- §1.10 — Today view SwiftUI
- §1.11 — `xcodebuild test` in CI (runs on `macos-14` GitHub runner)
- §3.3 — Family Controls authorization flow
- §3.4 — `ManagedSettings` adapter build verification
- §3.9 — Codesign + notarization pipeline

Items pending a macOS environment are marked `(mac-only)` in
`ROADMAP.md`.

## Xcode project tracking

When the Xcode project is added on macOS, `MacAgent.xcodeproj` is
tracked in git. User-specific Xcode state (`xcuserdata/`,
`*.xcuserstate`, user-specific schemes) is excluded by the root
`.gitignore`.
