# AGENTS.md

## Mission
Build ham-agents incrementally from the product spec and roadmap, but only implement the currently active work scope.

## Mandatory reading order
1. `spec.md`
2. `roadmap.md`
3. `AGENTS.md`
4. `tasks.md`
5. `docs/architecture.md`
6. `docs/assumptions.md`
7. `docs/progress.md`

## Execution contract
- Treat `spec.md` as the long-term product truth.
- Treat `roadmap.md` as the future release plan.
- Do not assume roadmap items are in scope by default.
- Before coding, refine the active work scope into explicit tasks in `tasks.md`.
- Keep implementation incremental and vertical-slice oriented.
- Prefer the smallest working slice that produces visible progress.
- Keep architecture simple and expandable, but do not pre-implement future features unless strictly needed.
- Record important decisions and ambiguities in `docs/assumptions.md`.
- Record progress continuously in `docs/progress.md`.
- Update `tasks.md` whenever a task starts or completes.


## Scope policy
- Start from analysis, then define the active implementation scope.
- The active implementation scope must be written down explicitly in `tasks.md` before major coding begins.
- Future roadmap features must not be implemented early unless there is a clear architectural reason.
- Avoid broad scaffolding with no immediate user value.

## Current priority order (2026-03-25)
The following features are NOT yet implemented. Work on them in this order.
Do NOT go back to polish already-shipped features (observed inference, lifecycle metadata, event presentation, etc.) unless there is a blocking bug.

1. **`ham run` actual session spawn** — `ham run claude` should actually launch a Claude CLI session in iTerm2, not just register a record.
2. **Team model** — `ham team create/add`, team grouping in CLI and menu bar (spec §6).
3. **Workspace filter** — project-path-based grouping and popover filtering (spec §6).
4. **Pixel office canvas** — fixed room layout with zone-based agent placement (spec §9). Start with static placement, no animation yet.
5. **Menu bar hamster sprite** — at minimum one animated hamster reflecting team status (spec §8).
6. **Settings UI completion** — General, Integrations, Privacy, Appearance sections (spec §17).

## Validation rules
Before marking any task done:
- relevant build succeeds
- tests for changed logic pass where practical
- no unresolved TODO/FIXME in changed files
- user-visible behavior matches the task acceptance criteria
- `tasks.md` and `docs/progress.md` are updated

## Decision policy
- If a detail is unspecified, choose the simplest implementation that preserves product direction.
- Record assumptions instead of blocking on small ambiguity.
- Be conservative about scope expansion.

## Delivery policy
- Keep naming and file structure production-lean.
- Prefer boring reliable code over clever abstractions.

## Commit policy
- Do NOT commit every tiny change. Bundle related changes into one meaningful commit per logical unit of work (e.g. one task, one feature slice, one bug fix).
- A single task should ideally produce 1~3 commits, not 10+.
- Use conventional commit messages with a type prefix:
  - `feat:` new feature or capability
  - `fix:` bug fix
  - `refactor:` code restructuring without behavior change
  - `test:` adding or updating tests
  - `docs:` documentation only
  - `chore:` build, config, or maintenance
- Message format: `type: concise summary of what and why`
- Examples:
  - `feat: Add observed-mode sleeping phrase inference`
  - `fix: Prevent duplicate notifications during status flap`
  - `refactor: Simplify daemon IPC payload decoding`
- Do NOT prefix with scope tags like `feat(inference):` — keep it flat.
