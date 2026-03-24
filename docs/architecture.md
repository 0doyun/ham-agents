# architecture.md

## Purpose
이 문서는 **현재 구현 기준의 실제 아키텍처 초안**을 기록한다.

주의:
- 이 문서는 최종 설계 문서가 아니라 현재 작업 범위 기준의 살아있는 문서다.
- 미래 확장 방향은 적을 수 있지만, 현재 구현과 분리해서 적는다.

---

## Long-Term Product Architecture Direction

현재 ham-agents의 장기 구조 방향은 다음과 같다.

1. `ham` CLI
2. `hamd` local runtime / daemon
3. macOS menu bar app
4. adapter layer
5. local persistence

---

## Active Implementation Architecture

현재 구현은 **Swift UI + Go CLI/runtime** 이원 구조를 기준으로 잡는다.

### Language split

#### Swift responsibilities

- macOS menu bar app
- popover / pixel office UI
- local notifications
- app lifecycle and permissions UX
- runtime stream consumer and command sender
- daemon payload decoding, socket client, and menu-bar-facing summary composition

#### Go responsibilities

- `ham` CLI
- `hamd` daemon
- agent/session registry
- process and session lifecycle management
- event ingestion
- persistence layer
- inference engine
- iTerm2 / transcript / generic process adapters

### Repository direction

1. `apps/macos/HamMenuBarApp`
   - SwiftUI/AppKit 메뉴바 앱
1a. `Sources/HamAppServices`
   - daemon client, payload bridge, menu bar summary logic
2. `go/cmd/ham`
   - 사용자 CLI 진입점
3. `go/cmd/hamd`
   - 백그라운드 daemon
4. `go/internal/core`
   - 도메인 모델
5. `go/internal/runtime`
   - agent lifecycle, session coordination
6. `go/internal/store`
   - SQLite 또는 file-based persistence
7. `go/internal/inference`
   - 상태 판정과 confidence 계산
8. `go/internal/adapters`
   - iTerm2, transcript, generic process adapters
9. `go/internal/ipc`
   - Unix domain socket, JSON RPC/event stream

### Transitional note

- 현재 저장소에는 Swift 기반 bootstrap code가 이미 존재한다.
- 이 코드는 초기 탐색/도메인 부트스트랩 산출물로 보고, 이후 Go runtime/CLI 구조로 점진적으로 재편한다.
- 문서와 실행 계획은 최종적으로 hybrid architecture를 기준으로 유지한다.

### Current primary data flow

1. 사용자가 `ham run`, `ham attach`, `ham status` 같은 CLI 명령을 실행
2. `ham` CLI가 `hamd`에 IPC 요청을 전송
3. `hamd`가 agent/session registry와 adapter를 통해 상태를 갱신
4. inference engine이 `(status, confidence, reason)` 을 계산
5. local store가 registry, event log, settings를 저장
6. menu bar app이 daemon event stream을 구독
7. UI, feed, notification layer가 최신 snapshot을 반영

### IPC contract direction

- 기본 전송 경로: Unix domain socket
- payload: JSON request/response + JSON event stream
- 명령 계층:
  - query: status, list, logs
  - command: run, attach, ask, stop, open
  - stream: lifecycle, alert, notification-worthy events

### Current technical constraints

- 첫 단계에서는 새로운 외부 의존성을 추가하지 않는다.
- 저장소는 항상 빌드 가능한 상태를 유지한다.
- 메뉴바 앱은 macOS 전용이지만 runtime/daemon 계층은 UI와 분리한다.
- CLI와 daemon은 UI와 독립적으로 동작해야 한다.
- iTerm2 연동은 초기에 adapter boundary만 고정하고 실제 제어는 점진적으로 구현한다.
- 스펙 전체 구현이 목표지만, 구현 순서는 managed-mode 중심의 vertical slice로 진행한다.
- 현재 Swift bootstrap과 최종 hybrid 구조 사이에 과도기 정리가 필요하다.

## Deferred Architecture

- transcript storage와 privacy masking의 구체 설계
- attached / observed mode용 adapter 계층 확장
- pixel office 렌더링 엔진과 sprite asset pipeline
- multi-workspace/team synchronization 방식
- settings storage schema
- iTerm2 권한 및 접근성 fallback 전략

## Build Surfaces

- Current bootstrap surface: SwiftPM (`swift build`, `swift test`)
- Current backend bootstrap surface: Go CLI/daemon module (`go test ./...`, `go run ./go/cmd/ham ...`, `go run ./go/cmd/hamd serve --once`)
- Target UI surface: Xcode/macOS app target for menu bar app

## Current Implemented Backend Slice

- `go/internal/core` owns the canonical managed-agent model for the new backend path.
- `go/internal/store` persists the managed registry to a local JSON file under `HAM_AGENTS_HOME` or `~/Library/Application Support/ham-agents/`.
- `go/internal/runtime` exposes register/list/snapshot behavior for managed agents and appends lifecycle events to the local event log.
- `go/internal/ipc` now owns the JSON request/response contract, Unix socket server, and daemon client used by the CLI.
- `go/cmd/ham` now talks to `hamd` over the local Unix socket for `run`, `list`, and `status`.
- `go/cmd/hamd` serves the runtime over the socket in normal mode and still supports `serve --once` / `snapshot` bootstrap inspection commands.
- `Sources/HamAppServices` now gives Swift a matching daemon request/response contract, Unix socket transport, and summary service for future menu bar surfaces.
- `apps/macos/HamMenuBarApp/Sources` now contains a compile-tested menu bar bootstrap that renders summary badges and a tracked-agent list using the shared Swift service layer.
- `MenuBarViewModel` now owns initial refresh + polling/retry behavior so menu bar state can track daemon changes without waiting for popover interaction.
- `HamNotifications` now owns transition-based notification candidate generation and a sink boundary so later macOS notification delivery can attach without changing polling logic.
- `HamNotifications` now also includes a `UserNotifications`-backed sink so the current polling/menu bar flow can emit real macOS notification requests.
- The current popover baseline includes agent selection, detail metadata, and per-agent recent-event context without requiring a second backend query path.
- Baseline popover actions currently start with project-folder opening via an injected opener boundary so later iTerm/session actions can follow the same pattern.
- Notification permission state is now exposed back into the Swift view-model layer so the popover can surface permission UX without duplicating delivery logic.
- Session opening now uses a small planning layer: if daemon data provides a URL-shaped `sessionRef`, Swift treats it as the preferred focus target; otherwise it falls back to workspace opening behavior.
- Per-agent notification pause/resume now goes through daemon IPC and file-backed agent persistence so mute state survives refreshes and remains the backend source of truth.
- Quick-message UI now routes through a dedicated sender boundary, with the current app-edge implementation preferring iTerm write automation and falling back to clipboard + session opening.
- Quick-message result feedback currently stays in the Swift view-model layer so the popover can surface delivery vs fallback outcomes without requiring backend acknowledgements yet.
- Selected-agent role editing now uses the same daemon mutation pattern as notification policy changes so role state remains backend-backed rather than Swift-local.
- Selected-agent role editing now follows the same daemon mutation pattern as notification policy changes so role state remains backend-backed rather than Swift-local.
