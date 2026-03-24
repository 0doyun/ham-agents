# assumptions.md

## Purpose
이 문서는 작업 중 생기는 **가정, 모호한 부분, 임시 결정**을 기록한다.

원칙:
- 작은 모호함 때문에 작업을 멈추지 않는다.
- 대신 여기 기록하고 가장 단순한 방향으로 진행한다.
- 나중에 scope 또는 구조 변경 시 이 기록을 참고한다.

---

## Initial Entries
- 2026-03-24: 최종 목표는 `spec.md` 전체 구현으로 본다. `roadmap.md`는 현재 범위를 제한하는 문서로 사용하지 않는다.
- 2026-03-24: 제품 구현 기준 아키텍처는 `Swift UI + Go CLI/runtime` 이원 구조로 본다.
- 2026-03-24: 메뉴바 앱과 macOS 통합 UX는 Swift가 담당하고, `ham` CLI / `hamd` daemon / 상태 수집은 Go가 담당한다.
- 2026-03-24: 현재 Swift 기반 bootstrap 코드는 과도기 산출물로 보고, 최종 구조에 맞게 점진적으로 재편한다.
- 2026-03-24: attached / observed mode는 최종 목표에 포함되지만, 첫 실구현 slice는 managed mode foundation으로 제한한다.
- 2026-03-24: 현재 작업 폴더는 Git 워크트리이며 origin/main push가 가능한 상태다.
- 2026-03-24: 첫 Go persistence는 SQLite 대신 file-based JSON registry로 시작한다. 이유는 dependency 없이 managed slice를 검증 가능한 상태로 만들기 쉽기 때문이다.
- 2026-03-24: 첫 daemon IPC는 Unix domain socket + JSON request/response 1-connection-per-command 방식으로 시작한다. event stream/multiplexing은 이후 slice에서 확장한다.
- 2026-03-24: Codex sandbox에서는 Unix socket bind가 차단될 수 있으므로 daemon-backed smoke 검증은 필요 시 unsandboxed 실행으로 보완한다.
- 2026-03-24: event log는 현재 registry save와 transaction을 공유하지 않으므로 best-effort 로깅으로 취급한다. authoritative state는 agent registry다.
- 2026-03-24: Swift menu bar/app surfaces는 별도 DTO를 재발명하지 않고 Go daemon JSON payload를 우선 공용 계약으로 사용한다.
- 2026-03-24: menu bar baseline 이전 단계에서는 Swift UI가 `HamAppServices` summary layer를 통해 daemon snapshot/event 조합 결과를 소비하는 방향으로 간다.
- 2026-03-24: initial menu bar bootstrap은 daemon transport 구성이 불가능할 때 preview client fallback을 사용하고, 정상 구성일 때는 launch 시점에 즉시 daemon refresh를 시작한다.
- 2026-03-24: 초기 polling 모델은 fixed-interval refresh + manual refresh 조합으로 간다. push/event-stream 기반 UI 동기화는 이후 slice에서 확장한다.
- 2026-03-24: notification foundation 단계에서는 daemon event stream 대신 polled status transition 비교로 done / waiting_input / error 후보를 만든다.
- 2026-03-24: initial macOS notification delivery는 first-send authorization request + immediate local notification request 방식으로 시작한다. richer permission UX는 이후 menu/settings slice에서 다듬는다.
- 2026-03-24: initial iTerm action baseline은 existing session focus 대신 workspace opening을 우선 지원한다. iTerm이 없으면 project-folder opener로 graceful fallback 한다.
- 2026-03-24: permission UX baseline은 full settings screen 대신 popover에서 현재 status 표시 + explicit request button 제공으로 시작한다.
- 2026-03-24: `sessionRef` 가 URL 로 주어지면 Swift opener는 그것을 직접 열어 session focus target 으로 취급한다. URL 이 아니면 workspace opening fallback 을 사용한다.
- 2026-03-25: notification pause/resume 은 process-local override 대신 daemon-backed agent policy update 로 이관한다. dedicated settings schema 는 이후에도 추가될 수 있지만 현재 source of truth 는 persisted agent record 다.
- 2026-03-25: quick message baseline은 iTerm write automation을 우선 시도하고, 실패 시 clipboard + session opening handoff 로 fallback 한다.
- 2026-03-25: quick message feedback baseline은 backend acknowledgement 없이 Swift sender result를 그대로 사용자에게 보여주는 방식으로 시작한다.
- 2026-03-25: role rename baseline은 daemon-backed agent record 를 직접 갱신하는 방식으로 시작한다. richer validation/history 는 이후 collaboration slice 에서 확장한다.
- 2026-03-25: stop-tracking baseline은 session/process termination 대신 registry removal만 수행한다. later slice에서 실제 kill/detach semantics를 붙인다.
- 2026-03-25: attached mode minimal baseline은 explicit `sessionRef` 를 받아 mode=`attached`, status=`idle`, confidence=`0.6` 으로 시작한다. richer inference/metadata sync 는 이후 attached slice에서 확장한다.
- 2026-03-25: mode/confidence baseline은 new heuristics without introducing new inference logic; it only exposes already-available daemon fields in the popover.
- 2026-03-25: observed mode heuristic baseline은 snapshot/list 시점에 source 파일 내용을 읽어 error/done/question/staleness를 판정한다. always-on watching은 later slice에서 추가한다.
- 2026-03-25: observed polling baseline은 background ticker 로 source를 다시 읽는 수준에서 시작한다. OS-level watch 서비스는 later slice에서 붙인다.
- 2026-03-25: `ham open` baseline은 daemon이 계산한 open target 을 CLI가 그대로 소비하는 방식으로 시작한다. richer platform-specific focus/open behavior는 later integration slices에서 확장한다.
- 2026-03-25: first Swift settings integration only exposes notification toggles from the backend settings document; broader settings sections remain later slices.
- 2026-03-25: quiet hours baseline now stores hour-only start/end values (`22 -> 8` default) in the daemon settings document so CLI and Swift UI can round-trip the same schema.
- 2026-03-25: quiet hours evaluation uses local current-hour checks in Swift notification filtering; when start and end hours are equal, the baseline treats that as all-day suppression instead of inventing minute-level semantics.
- 2026-03-25: richer iTerm session identification baseline treats `iterm2://session/<id>` as a structured session target instead of a generic URL, so open/ask can aim at a specific session before falling back to current-session behavior.
- 2026-03-25: attach picker baseline reads a lightweight iTerm session snapshot via AppleScript and only trusts session id/title/current-session marker for now; richer cwd/layout/termination metadata remains a later slice.
