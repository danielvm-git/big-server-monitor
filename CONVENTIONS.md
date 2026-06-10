# PortKeeper Conventions

## Architecture: ECC
- Kernel owns component lifecycle (Init → Start → Stop)
- Components communicate via EventBus (no direct imports)
- Each component lives in components/<name>/<name>.go

## Go
- camelCase unexported, PascalCase exported
- Acronyms all-caps: HTTP, URL, API, JSON, PID
- Error wrapping: fmt.Errorf("context: %w", err)
- Table-driven tests with t.Run

## Git
- Conventional Commits: feat:, fix:, chore:, test:, docs:
- Branch naming: feat/<kebab-case>
- Never commit to main directly

## specs/
- All planning docs go in specs/
- Every implementation task needs a verify: command
