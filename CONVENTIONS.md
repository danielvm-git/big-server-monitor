# BigServerMonitor Conventions

## Architecture
- Swift 6 actors for concurrency, @Observable + @MainActor for UI state
- Components in Sources/Core/ are actors — no @MainActor, no SwiftUI imports
- UI in Sources/UI/ reads @Environment(AppState.self)
- AppState bridges actors → UI via observable properties
- No direct imports between Core/ components

## Swift
- camelCase for properties and functions, PascalCase for types
- Acronyms all-caps: HTTP, URL, API, JSON, PID
- Actors for async state (ProcessMonitor, ActivityStore)
- @Observable + @MainActor for UI-facing state (AppState)
- Tests use Swift Testing: `@Suite`, `@Test`, `#expect`
- Table-driven patterns where possible (use arrays of tuples in tests)

## Git
- Conventional Commits: feat:, fix:, chore:, test:, docs:
- Branch naming: feat/<kebab-case>
- Never commit to main directly

## specs/
- All planning docs go in specs/
- Every implementation task needs a verify: command
