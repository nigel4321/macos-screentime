# Contributing

## Test-driven development

TDD is mandatory for the modules that hold business logic:

- `mac-agent/Core/PolicyEngine` (Swift)
- `mac-agent/Core/LocalStore` (Swift)
- `android-app/core-domain` (Kotlin)
- `backend/internal` handlers and DB layer (Go)

For these modules the workflow is **red → green → refactor**:

1. Write a failing test that describes the next behaviour.
2. Run it and watch it fail for the right reason.
3. Write the minimal code to make it pass.
4. Refactor with the test green.

A pull request that changes one of these modules without adding or changing a test will be bounced on review. Coverage thresholds are **not** enforced by CI — reviewers enforce intent.

Adapter modules (UI, OS-framework wrappers, network clients) are tested at the module boundary with fakes. UI modules are tested with Compose UI tests and SwiftUI snapshot / interaction tests where practical.

## Branches

- `main` is always releasable from each component's perspective.
- Feature branches: `feature/<short-slug>`.
- Fix branches: `fix/<short-slug>`.
- Release tags: `mac-vX.Y.Z`, `android-vX.Y.Z`, `backend-vX.Y.Z`.

## Commits

- Imperative mood: "Add downtime-window test" not "Added…" or "Adds…".
- First line ≤72 characters.
- Body explains *why* when the *what* is not self-evident from the diff.
- Group related changes; avoid mixing refactors with behaviour changes in one commit.

## Pull requests

- One PR per logical change. A PR that touches the Mac agent, the backend, and the Android app is almost always three PRs.
- Title follows commit-line rules.
- Description: what changed, why, how it was tested, anything reviewers should watch out for.
- Link the `ROADMAP.md` checkbox(es) the PR ticks, so the roadmap stays the source of truth.
- CI must be green before review is requested.
- Squash-merge by default.

## Code style

- Swift: enforced by `SwiftLint` (config in `mac-agent/.swiftlint.yml`).
- Kotlin: enforced by `ktlint` and `detekt` (configs in `android-app/`).
- Go: enforced by `go vet` and `golangci-lint` (config in `backend/`).

Run local checks before pushing — CI runs them too, but a two-minute local fix is cheaper than a failed CI run.

## Dependencies

- Prefer standard libraries and the few chosen frameworks listed in `ARCHITECTURE.md`.
- Adding a new direct dependency requires a one-line justification in the PR description.

## Secrets

- Never commit `.env`, keystores, signing certificates, or service-account JSON.
- The root `.gitignore` already excludes the common paths — do not override it.
- Secrets for CI live in GitHub Actions encrypted secrets.

## Resuming the work

The project uses a set of in-repo documents as its durable state. If you pick this up on a new machine (or after a break), read these in order:

1. **`PROJECT.md`** — objective and ways of working.
2. **`ARCHITECTURE.md`** — all technology choices, data model, delivery milestones, and explicitly-rejected alternatives. If something seems underspecified, look here first.
3. **`ROADMAP.md`** — the live checklist. `[x]` items are done, `[~]` are in progress, `[ ]` are todo. The next item to work on is the first unchecked box after the last `[x]`.
4. **`CONTRIBUTING.md`** (this file) — conventions below.

### Working conventions

- **Commit cadence**: one commit per `ROADMAP.md` sub-section (e.g. all of §1.1, then all of §1.2). Not per item, not per milestone. Commit message prefix: `[M<milestone>.<section>]`, e.g. `[M1.3] PolicyEngine scaffolding`.
- **Milestone stops**: after the last item of a milestone is ticked, stop and summarise before starting the next milestone. The roadmap is a contract that the developer (or AI assistant) does not silently overrun.
- **Plan-before-execute**: new planning docs (architecture, roadmap, RFCs) are reviewed before the work they describe begins. Don't treat "create the plan" as implicit permission to build it.

### Environment expectations

- **Pure-Swift modules** (`mac-agent/Package.swift` products like `PolicyEngine`, `LocalStore`, domain types in `UsageCollector`) build and test with `swift test` on Linux *or* macOS. AppKit/SwiftUI code is guarded with `#if canImport(AppKit)` so Linux builds stay green.
- **Xcode app target, codesigning, notarization, `ManagedSettings` / `FamilyControls` work** require macOS. These steps are scaffoldable on Linux only up to the protocol/source level; Xcode project generation and build verification happen on a Mac.
- **Android app, Go backend, CI workflows, documentation** are fully buildable on either OS.

If you see a `ROADMAP.md` item with an inline `(mac-only)` note, that item is waiting for a macOS session.
