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

## Progression policy
When the current Active Scope checklist in `tasks.md` is fully checked off:
1. **Move to the next epic.** Do not stay in the completed scope to find more things to polish, refine, or extend. The next unstarted epic in `tasks.md` becomes the new Active Scope.
2. **Polish is deferred.** Wording, presentation, label, and cosmetic improvements to already-shipped slices are not new tasks. Collect them in a dedicated "Polish backlog" section and execute them only after all feature epics are complete, or when explicitly requested.
3. **If you are unsure whether something is a new feature or polish** — if it does not add a user-visible capability that did not exist before, it is polish.
4. **Update `tasks.md` Active Scope** to the new epic before writing any code for it.

## Code quality policy
- **Search before you write.** Before adding a new helper, utility, or mapping function, search the existing codebase for identical or near-identical logic. Reuse what exists instead of duplicating.
- **Shared logic goes in shared packages.** If both `cmd/ham` and `internal/runtime` need the same function, put it in one place and import it. Do not copy-paste across files.
- **One file, one responsibility.** If a source file contains multiple unrelated responsibilities (e.g. CLI subcommands + rendering helpers + sorting logic, or SwiftUI views + AppleScript automation + preview mocks), split along responsibility boundaries before adding more code. Do not split files by arbitrary line count.
- **One commit, one purpose.** Do not split a single logical change into multiple tiny commits that each add one keyword or one label. Group related work into a single coherent commit.

## Scope policy
- Start from analysis, then define the active implementation scope.
- The active implementation scope must be written down explicitly in `tasks.md` before major coding begins.
- Future roadmap features must not be implemented early unless there is a clear architectural reason.
- Avoid broad scaffolding with no immediate user value.
- A scope is **done** when its checklist is complete. Do not invent new checklist items for a completed scope. Move on.

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
- Do NOT create bookkeeping-only commits like "ledger update" or "catch up docs". Include docs updates in the work commit.
