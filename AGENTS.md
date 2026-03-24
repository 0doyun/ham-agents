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
- Commit in small, reviewable increments.
- Keep naming and file structure production-lean.
- Prefer boring reliable code over clever abstractions.
