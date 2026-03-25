# progress.md

## Purpose
이 문서는 구현 진행 상황과 주요 결정을 순서대로 기록한다.

원칙:
- 큰 작업 시작 전 계획을 적는다.
- slice가 끝날 때마다 결과를 남긴다.
- build/test/검증 결과도 함께 적는다.

---

## Log

### 2026-03-24
- 문서 초기 세팅 완료
- spec / roadmap / AGENTS / tasks / docs 뼈대 작성
- 다음 단계: 분석 후 현재 활성 구현 범위 정의

### 2026-03-24 (structure bootstrap)
- 전체 목표를 `spec.md` 전체 구현으로 재정의하고, `tasks.md`를 장기 backlog + 현재 slice 중심 구조로 재작성
- `docs/architecture.md`에 Swift 기반 모듈 아키텍처와 현재 데이터 흐름 초안 반영
- `docs/assumptions.md`에 언어/플랫폼/실행 순서 관련 초기 가정 기록
- Ralph 실행을 위한 `.omx/plans/prd-ham-agents.md`, `.omx/plans/test-spec-ham-agents.md` 생성
- SwiftPM 패키지와 핵심 모듈/테스트 골격 추가
- 남은 전제조건: 이 디렉터리를 실제 Git 워크트리로 연결해 commit/push 가능한 상태로 만들기

### 2026-03-24 (spec and architecture realignment)
- 초기 상세 스펙 초안을 기준으로 `spec.md`를 다시 확장해 제품 truth를 복원
- 빠져 있던 설정 화면, UX 플로우, 성능 목표, 단계별 범위, 구현 순서, 아키텍처 디테일을 `spec.md`에 재반영
- `docs/architecture.md`를 `Swift UI + Go CLI/runtime` hybrid 구조 기준으로 재작성
- `tasks.md`와 `docs/assumptions.md`를 hybrid 구조와 현재 Git 상태에 맞게 갱신
- 목적: Ralph가 압축본 스펙과 실제 기술 방향 사이에서 재해석하지 않도록 기준 문서를 정렬

### 2026-03-24 (hybrid Go bootstrap + managed registry slice)
- 저장소 레이아웃을 `apps/macos/HamMenuBarApp` + `go/...` 구조로 실제 정렬했다.
- 루트 `go.mod`와 `go/cmd/ham`, `go/cmd/hamd`, `go/internal/{core,runtime,store,ipc,adapters}` 를 추가했다.
- Go backend 첫 slice로 managed agent domain model, file-backed registry store, runtime snapshot, `ham run/list/status`, `hamd serve --once` bootstrap을 구현했다.
- Swift bootstrap은 그대로 유지하고, 새 Go slice를 병행 검증 가능한 상태로 만들었다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - smoke:
    - `go run ./go/cmd/ham list` → `no tracked agents`
    - `go run ./go/cmd/ham run claude reviewer --project /tmp/demo --role reviewer` → managed agent 등록
    - `go run ./go/cmd/ham list` / `status` → persisted agent + counts 확인
    - `go run ./go/cmd/hamd serve --once` → bootstrap socket/state 경로 출력
- 다음 우선순위: `ham` ↔ `hamd` 실제 IPC 연결, event/runtime coordinator, menu bar baseline이 읽을 snapshot contract 고정

### 2026-03-24 (daemon IPC + event flow foundation)
- `go/internal/ipc`에 JSON request/response contract, Unix socket client/server, daemon dispatch를 추가했다.
- `go/cmd/ham`은 더 이상 store를 직접 읽지 않고 `hamd` daemon client를 통해 `run`, `list`, `status` 를 호출한다.
- `go/internal/store`에 JSONL event log를 추가하고, managed agent 등록 시 `agent.registered` 이벤트를 기록하도록 runtime을 확장했다.
- `go/cmd/hamd serve --once=false` 는 실제 socket server를 띄우고, `serve --once` / `snapshot` 은 bootstrap inspection 용도로 유지했다.
- architect review에서 지적된 unsafe socket cleanup 을 수정해 stale unix socket만 제거하도록 바꿨다.
- event append 실패가 `ham run` 을 false-negative 로 만들지 않도록 registry 저장을 authoritative path로 유지하고 event logging은 best-effort 로 정리했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - unsandboxed smoke:
    - `go run ./go/cmd/hamd serve --once=false` 백그라운드 실행 ✅
    - `go run ./go/cmd/ham list` → `no tracked agents` ✅
    - `go run ./go/cmd/ham run claude reviewer --project /tmp/demo --role reviewer` → daemon 경유 등록 ✅
    - `go run ./go/cmd/ham status --json` → runtime snapshot 반환 ✅
- 남은 갭: long-lived event stream, richer lifecycle transitions, menu bar app runtime consumption

### 2026-03-24 (event query / feed-ready backend slice)
- `ham events` CLI를 추가해 daemon-backed recent event feed를 바로 조회할 수 있게 했다.
- IPC contract에 `events.list` + `limit` 필드를 추가하고, daemon이 runtime event log를 recent-first bounded query로 노출하도록 연결했다.
- runtime에 `Events(limit)` 조회를 추가해 future activity feed / menu bar detail panel이 같은 backend surface를 재사용할 수 있게 했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - unsandboxed smoke:
    - `go run ./go/cmd/hamd serve --once=false` ✅
    - `go run ./go/cmd/ham list` → `no tracked agents` ✅
    - `go run ./go/cmd/ham run claude reviewer --project /tmp/demo --role reviewer` ✅
    - `go run ./go/cmd/ham events --json --limit 5` → `agent.registered` event 반환 ✅
    - `go run ./go/cmd/ham status --json` ✅
- 다음 우선순위 후보: event stream/follow mode, menu bar baseline target, runtime lifecycle transition enrichment

### 2026-03-24 (Swift daemon payload decoding prep)
- `HamCore.Agent` 에 Go daemon JSON과 맞는 `CodingKeys` 를 추가해 Swift UI 쪽이 backend payload를 직접 decode 할 수 있게 맞췄다.
- `DaemonStatusPayload`, `AgentEventPayload`, `DaemonJSONDecoder` 를 추가해 menu bar baseline이 재사용할 최소 bridge surface를 만들었다.
- Go smoke output 형식과 맞춘 fixture 기반 decoding tests를 추가해 Swift가 agent/status/event payload를 읽을 수 있음을 고정했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: actual Swift daemon client transport, menu bar target/app bootstrap, event follow/stream surface

### 2026-03-24 (Swift daemon client + menu bar summary baseline prep)
- `HamCore` 에 daemon IPC request/response + runtime snapshot payload 모델을 추가해 Swift가 CLI가 아니라 daemon contract 자체를 이해할 수 있게 했다.
- `HamAppServices` 타깃을 추가하고, Unix socket 기반 `UnixSocketDaemonTransport`, `HamDaemonClient`, `MenuBarSummaryService` 를 구현했다.
- summary service가 daemon snapshot + recent events 를 menu bar-friendly count/feed data로 합성하도록 만들고, fixture/stub 기반 tests로 polling behavior를 보호했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: 실제 menu bar app target bootstrap, Swift-side live daemon polling integration, popover agent list baseline

### 2026-03-24 (menu bar executable bootstrap)
- `ham-menubar` SwiftPM executable target을 추가해 menu bar baseline이 실제 build graph에 들어오도록 했다.
- `MenuBarViewModel` 을 추가해 daemon snapshot/events/agent list refresh를 하나의 Swift UI-facing state object로 정리했다.
- `apps/macos/HamMenuBarApp/Sources/HamMenuBarApp.swift` 에 `MenuBarExtra` 기반 baseline UI를 추가해 status line, summary badges, tracked agent list, refresh button을 렌더링하도록 만들었다.
- live daemon transport를 우선 사용하되 연결 구성이 없을 때는 preview client fallback 으로 shell UI를 계속 띄울 수 있게 했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: live polling timer / reconnect behavior, popup agent detail actions, notification triggers, actual macOS app packaging

### 2026-03-24 (menu bar polling / recovery refinement)
- `MenuBarViewModel` 에 start/stop + polling task를 추가해 launch 이후에도 daemon state를 주기적으로 다시 읽도록 만들었다.
- initial refresh failure 뒤에도 이후 poll cycle에서 recovery 할 수 있게 만들고, 관련 behavior를 Swift tests로 고정했다.
- menu bar label이 popover를 열기 전에도 daemon-backed 상태로 갱신되도록 launch 시점 refresh를 유지한 채 polling 모델로 확장했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: popover agent detail actions, notification trigger hookup, live event follow / stream consumption

### 2026-03-24 (notification trigger foundation)
- `HamNotifications` 에 status transition 비교 기반 `StatusChangeNotificationEngine` 과 `NotificationSink` boundary 를 추가했다.
- `MenuBarViewModel` refresh path를 확장해 이전 agent 상태와 새 상태를 비교하고 done / waiting_input / error 전이에서 notification candidate를 sink로 보낼 수 있게 했다.
- 반복 poll에서 같은 상태를 다시 알리지 않도록 transition-based dedupe 를 기본으로 잡고, muted policy 도 계속 존중하도록 유지했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: actual macOS notification delivery sink, popover agent detail actions, live event stream/follow integration

### 2026-03-24 (macOS notification delivery sink)
- `HamNotifications` 에 `UserNotificationCentering`, `LiveUserNotificationCenter`, `UserNotificationSink` 를 추가해 notification candidate를 실제 macOS notification request로 보낼 수 있게 했다.
- sink는 첫 전송 시 권한 요청을 수행하고, 승인된 경우 `done` / `waiting_input` / `error` 후보를 `UNNotificationRequest` 로 변환해 전달한다.
- menu bar app이 이제 noop sink 대신 실제 `UserNotificationSink` 를 주입받아 polling 기반 transition detection과 notification delivery를 바로 연결한다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: richer notification permission UX, popover agent detail actions, live event stream/follow integration

### 2026-03-24 (popover detail panel + recent event context)
- `MenuBarViewModel` 에 selected-agent helper 와 recent-event filtering helper 를 추가해 popover detail pane 이 현재 daemon snapshot/event data를 그대로 재사용하게 했다.
- menu bar popover를 2-column baseline으로 확장해 좌측 agent list, 우측 detail panel, recent events block 을 함께 보여주도록 만들었다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: popover agent detail actions, richer notification permission UX, live event stream/follow integration

### 2026-03-24 (popover baseline action wiring)
- `ProjectOpening` boundary 와 default noop implementation을 추가해 menu bar/UI 쪽 action wiring을 testable 하게 만들었다.
- popover detail pane에 `Open Project Folder` 버튼을 추가하고, app target에서는 `NSWorkspace` 기반 opener 로 연결했다.
- Swift tests로 selected agent에서 project path 가 opener 로 전달되는지 검증했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: iTerm/session-focused actions, richer notification permission UX, live event stream/follow integration

### 2026-03-24 (iTerm/workspace action baseline)
- `SessionOpening` boundary 를 추가해 selected agent action 이 AppKit/iTerm specific implementation 과 분리되도록 만들었다.
- popover detail pane에 `Open in iTerm` 버튼을 추가하고, app target에서는 iTerm 설치 시 workspace 경로를 iTerm으로 열고, 없으면 project-folder opener 로 fallback 하도록 연결했다.
- Swift tests로 selected agent 가 injected session opener 로 전달되는지 고정했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: real existing-session focus semantics, iTerm message/send actions, richer notification permission UX

### 2026-03-24 (notification permission UX baseline)
- `UserNotificationSink` 에 current/request permission status surface 를 추가해 notification delivery 와 permission UX 가 같은 boundary 를 공유하게 만들었다.
- `MenuBarViewModel` 이 refresh 시 notification permission status 를 함께 읽고, popover에서 `Notifications` 상태와 `Enable` 액션을 보여줄 수 있게 했다.
- Swift tests로 permission status refresh 와 explicit request action 이 published state 를 갱신하는지 고정했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: real existing-session focus semantics, iTerm message/send actions, live event stream/follow integration

### 2026-03-24 (sessionRef-aware opener refinement)
- `SessionTargetPlanner` 를 추가해 agent의 `sessionRef` 가 URL 형태일 때는 그 URL 을 우선 열고, 그렇지 않으면 workspace 기반 iTerm/finder fallback 을 사용하도록 정리했다.
- Swift tests로 session target selection 규칙을 고정하고, existing `openSession` action wiring 은 그대로 재사용하도록 유지했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: richer session identification from daemon, iTerm message/send actions, live event stream/follow integration

### 2026-03-24 (popover notification pause/resume baseline)
- `MenuBarViewModel` 에 local notification policy override 를 추가해 selected agent 별로 notification mute 상태를 즉시 바꿀 수 있게 했다.
- popover detail pane에 `Pause Notifications` / `Resume Notifications` 버튼을 추가하고, muted 상태일 때 이후 done transition notification candidate 가 suppress 되는 것을 테스트로 고정했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: richer session identification from daemon, iTerm message/send actions, backend-persisted notification settings

### 2026-03-25 (quick message baseline)
- `QuickMessageSending` boundary 를 추가하고, `MenuBarViewModel` 에서 selected agent 기준으로 quick message handoff action 을 route 하도록 만들었다.
- popover detail pane에 multi-line draft field 와 `Copy & Open Session` 버튼을 추가했다.
- app target baseline sender 는 message 를 clipboard 에 복사한 뒤 session opening path 를 재사용하도록 연결해, actual terminal write automation 전에도 정직한 handoff UX 를 제공한다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: real terminal write automation for iTerm, backend-persisted notification settings, live event stream/follow integration

### 2026-03-25 (iTerm quick message write baseline)
- `QuickMessagePlanner` 를 추가해 quick message path 가 terminal write 를 시도할지 clipboard handoff 로 갈지 분리했다.
- app target sender 는 iTerm 이 준비되어 있으면 AppleScript `write text` automation 을 먼저 시도하고, 실패하면 clipboard + session/workspace opening fallback 으로 내려가도록 만들었다.
- Swift tests로 quick message planner selection 규칙을 고정했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: richer iTerm session identification, actual send acknowledgement/error surfacing, backend-persisted notification settings

### 2026-03-25 (quick message feedback baseline)
- `QuickMessageSending` 이 `QuickMessageResult` 를 반환하도록 바꿔 sender가 delivered / handoff / failed outcome 을 명시하게 했다.
- `MenuBarViewModel` 이 quick message 결과를 published feedback text 로 저장하고, popover detail pane에서 사용자에게 즉시 보여주도록 연결했다.
- sender가 구성되지 않았거나 agent가 선택되지 않은 경우에도 조용히 무시하지 않고 feedback 을 남기도록 baseline error surface 를 추가했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: richer iTerm session identification, actual send acknowledgement from backend/runtime, backend-persisted notification settings

### 2026-03-25 (daemon-persisted notification policy baseline)
- Go runtime/IPC 에 agent notification policy update path를 추가해 mute state를 daemon-backed source of truth 로 옮겼다.
- Swift `MenuBarViewModel` 은 더 이상 process-local override 에 의존하지 않고 daemon client 를 통해 notification policy 를 갱신한다.
- Go/Swift tests 를 추가해 notification policy update 가 persistence 와 UI refresh 양쪽에서 유지되는지 보호했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: richer iTerm session identification, actual send acknowledgement from backend/runtime, backend-persisted broader settings state

### 2026-03-25 (agent role rename baseline)
- Go runtime/IPC/client 에 role update path 를 추가해 selected agent role 을 daemon-backed source of truth 쪽에서 갱신할 수 있게 했다.
- Swift detail pane에 role draft field 와 `Save` action 을 추가하고, view model 이 daemon mutation 결과로 local agent list 를 갱신하도록 연결했다.
- Go/Swift tests 로 role update persistence 와 selected-agent role save behavior 를 보호했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: broader backend-persisted settings state, richer iTerm session identification/send acknowledgement, live event stream/follow integration

### 2026-03-25 (stop tracking baseline)
- Go runtime/IPC/client 에 remove agent path 를 추가해 selected agent 를 daemon-backed registry 에서 제거할 수 있게 했다.
- popover detail pane에 `Stop Tracking` action 을 추가하고, Swift view model 이 성공 시 local agent list 와 selection state 를 즉시 정리하도록 연결했다.
- Go/Swift tests 로 remove flow 를 보호했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: actual session/process termination semantics, broader backend-persisted settings state, live event stream/follow integration

### 2026-03-25 (attached mode minimal foundation)
- Go runtime/IPC/CLI 에 `ham attach` path 를 추가해 explicit sessionRef 기반 attached agent 를 등록할 수 있게 했다.
- attached agent 는 mode=`attached`, status=`idle`, confidence=`0.6` 으로 시작하게 해 managed 와 구분된 낮은 확신도를 baseline 에 반영했다.
- unsandboxed smoke 로 `hamd serve --once=false` 뒤 `ham attach ...`, `ham list`, `ham status --json` 을 확인했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - unsandboxed smoke:
    - `go run ./go/cmd/ham attach iterm2://session/abc ops --project /tmp/demo --role reviewer` ✅
    - `go run ./go/cmd/ham list` → attached mode 노출 ✅
    - `go run ./go/cmd/ham status --json` ✅
- 다음 우선순위 후보: richer attached metadata/session identification, observed mode baseline, live event stream/follow integration

### 2026-03-25 (mode/confidence UI baseline)
- agent list 와 detail pane 에 `mode` 와 `statusConfidence` 기반 confidence text 를 노출해 managed/attached/observed 구분과 tracking certainty가 baseline UI 에서 바로 보이게 했다.
- Swift view model 에 confidence formatting helper 를 추가하고, tests 로 percentage formatting 을 고정했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: actual session/process termination semantics, broader backend-persisted settings state, live event stream/follow integration

### 2026-03-25 (observed source refresh + heuristic baseline)
- Go runtime 에 observed source refresh helper 를 추가해 list/snapshot 시점에 transcript/log 파일 내용을 읽고 error / done / waiting_input / sleeping 류의 lightweight heuristic 을 적용하도록 만들었다.
- `ham observe` path 를 CLI/daemon 에 추가해 explicit source ref 기반 observed agent 를 등록할 수 있게 했다.
- unsandboxed smoke 로 `hamd serve --once=false` 뒤 `ham attach ...`, `ham observe ...`, `ham list`, `ham status --json` 을 확인했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - unsandboxed smoke:
    - `go run ./go/cmd/ham observe /tmp/demo.log watcher --project /tmp/demo --role watcher` ✅
    - `go run ./go/cmd/ham list` / `status --json` ✅
- 다음 우선순위 후보: always-on observed watching, richer attached/iTerm session identification, broader backend-persisted settings state

### 2026-03-25 (observed source polling baseline)
- `RuntimeRegistry.RefreshObserved` 공개 entrypoint 를 추가하고, `hamd serve` 가 2초 간격으로 observed source 를 refresh 하도록 polling loop 를 붙였다.
- Go tests 로 refresh entrypoint 가 persisted observed status 를 갱신하는지 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: OS-level observed watching, richer attached/iTerm session identification, broader backend-persisted settings state

### 2026-03-25 (CLI open target baseline)
- Go runtime/IPC/client 에 open-target resolution path 를 추가해 agent의 current open target 을 daemon source of truth 쪽에서 계산하도록 만들었다.
- `ham open <agent>` CLI baseline 을 추가해 print/json path 를 지원하고, attached sessionRef URL 이 있으면 URL target 을 우선 사용하도록 했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: broader backend-persisted settings state, richer iTerm session identification/send acknowledgement, OS-level observed watching

### 2026-03-25 (backend settings persistence baseline)
- `go/internal/store/settings.go` 와 `runtime.SettingsService` 를 추가해 backend settings document 를 JSON 파일로 읽고 쓰는 baseline 을 만들었다.
- daemon IPC 와 `ham settings --json` / `ham settings notifications ...` CLI path 를 추가해 settings 조회/갱신의 첫 automation surface 를 열었다.
- Go tests 로 settings store/runtime/IPC persistence 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: broader settings schema, richer attached/iTerm session identification/send acknowledgement, OS-level observed watching

### 2026-03-25 (settings UI integration baseline)
- Swift daemon payload/client 에 settings fetch/update path 를 추가하고, `MenuBarViewModel` 이 notification settings document 를 함께 읽고 수정할 수 있게 했다.
- menu bar popover에 notification settings toggle section 을 추가해 Done/Error/Waiting Input/Preview Text 토글을 backend-persisted settings 문서와 round-trip 하도록 연결했다.
- Swift tests 로 settings fetch/update round-trip 이 published state 에 반영되는지 보호했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: broader settings schema, richer attached/iTerm session identification/send acknowledgement, OS-level observed watching

### 2026-03-25 (notification settings enforcement baseline)
- `MenuBarViewModel` notification filtering 이 backend settings document 를 읽어 done/error/waiting_input 토글을 실제 delivery behavior 에 반영하도록 연결했다.
- preview text 가 꺼져 있을 때 notification body 를 최소 문구로 마스킹하고, quiet hours enabled 가 켜져 있으면 current baseline 에서는 모든 notification candidate 를 suppress 하도록 만들었다.
- Swift tests 로 done suppression, preview-text masking, quiet-hours suppression behavior 를 고정했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: richer quiet-hours schema, broader settings sections, richer attached/iTerm session identification/send acknowledgement

### 2026-03-25 (notification settings enforcement baseline)
- `MenuBarViewModel` notification filtering 이 backend settings document 를 읽어 done/error/waiting_input 토글을 실제 delivery behavior 에 반영하도록 연결했다.
- preview text 가 꺼져 있을 때 notification body 를 최소 문구로 마스킹하고, quiet hours enabled 가 켜져 있으면 current baseline 에서는 모든 notification candidate 를 suppress 하도록 만들었다.
- Swift tests 로 done suppression, preview-text masking, quiet-hours suppression behavior 를 고정했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: richer quiet-hours schema, broader settings sections, richer attached/iTerm session identification/send acknowledgement

### 2026-03-25 (agent role rename baseline)
- Go runtime/IPC/client 에 role update path 를 추가해 selected agent role 을 daemon-backed source of truth 쪽에서 갱신할 수 있게 했다.
- popover detail pane에 role draft field 와 `Save` action 을 추가하고, Swift view model 이 daemon mutation 결과로 local agent list 를 갱신하도록 연결했다.
- Go/Swift tests 로 role update persistence 와 selected-agent role save behavior 를 보호했다.
- 검증:
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
- 다음 우선순위 후보: broader backend-persisted settings state, richer iTerm session identification/send acknowledgement, live event stream/follow integration

### 2026-03-25 (quiet hours schedule baseline)
- Go settings schema 에 `quiet_hours_start_hour` / `quiet_hours_end_hour` 를 추가하고, legacy settings 파일도 default window (`22 -> 8`) 로 backfill 되도록 했다.
- `ham settings notifications` CLI 가 quiet-hours on/off 와 start/end hour 를 수정할 수 있게 확장했고, Go tests 로 hour parsing / store / runtime / IPC round-trip 을 보호했다.
- Swift daemon payload, menu bar settings section, `MenuBarViewModel` 을 업데이트해 quiet-hours start/end 을 UI 에서 조정하고 현재 시각 기준 window suppression 을 적용하도록 연결했다.
- Swift tests 로 overnight quiet-hours suppression 과 outside-window delivery behavior 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer attached/iTerm session identification/send acknowledgement, broader settings sections, live event stream/follow integration

### 2026-03-25 (richer attached/iTerm session identification baseline)
- daemon open-target resolution 이 `iterm2://session/<id>` 를 generic URL 대신 structured `iterm_session` target 으로 해석하고, `session_id` 를 함께 전달하도록 확장했다.
- `ham open` / `ham ask` 는 daemon이 계산한 structured target 을 그대로 재사용하고, Go quick-message sender 는 iTerm current session 대신 matching session id 를 우선 찾아 write 하도록 정교화했다.
- Swift `SessionTargetPlanner`, menu bar session opener, quick-message sender 도 같은 session-id parsing 규칙을 사용해 specific iTerm session focus/write 를 먼저 시도하고, 실패 시 기존 URL/workspace fallback 을 유지한다.
- Go/Swift tests 로 iTerm session target parsing, open-target payload, quick-message targeting behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: attach picker / iTerm session listing baseline, broader settings sections, live event stream/follow integration

### 2026-03-25 (attach picker / iTerm session listing baseline)
- Go `Iterm2Adapter` 에 AppleScript 기반 session listing baseline 을 추가하고, attachable session snapshot (`id`, `title`, `session_ref`, `is_active`) 을 daemon/CLI 가 재사용할 수 있게 만들었다.
- daemon IPC 에 `iterm.sessions` surface 를 추가하고, CLI `ham attach --list-iterm-sessions` / `ham attach --pick-iterm-session` 로 list + interactive picker attach baseline 을 열었다.
- Swift daemon client / menu bar view model 도 attachable session snapshot 을 읽어 popover 에 상위 iTerm session 목록을 표시하도록 연결했다.
- Go/Swift tests 로 session listing parsing, picker selection, daemon round-trip, Swift refresh exposure 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: attached session termination detection baseline, broader settings sections, live event stream/follow integration

### 2026-03-25 (attached session termination detection baseline)
- runtime registry 에 attached-session refresh path 를 추가해 polled iTerm session snapshot 에서 사라진 attached agent 를 `disconnected` 로 표시하고, 같은 `session_ref` 가 다시 보이면 `idle` 로 복구하도록 만들었다.
- `hamd serve` background poll loop 가 observed refresh와 함께 iTerm session listing 을 읽어 attached disconnect detection 을 주기적으로 갱신하도록 연결했다.
- Swift tests 로 disconnected attached agent surface 를 보호하고, Go tests 로 disconnect/reconnect refresh behavior 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: broader settings sections baseline, live event stream/follow integration, richer attached metadata sync

### 2026-03-25 (broader settings sections baseline)
- daemon settings schema 에 `appearance.theme` (`auto|day|night`) section 을 추가해 settings 문서가 notifications-only 구조에서 한 단계 확장되도록 만들었다.
- CLI `ham settings appearance --theme=...` 와 Swift menu bar `Appearance` section 을 연결해 backend-backed non-notification setting 을 양쪽 surface 에서 수정 가능하게 만들었다.
- Go tests 로 appearance theme persistence/validation 을 보호하고, Swift tests 로 published appearance setting round-trip 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: live event stream / follow baseline, richer attached metadata sync, stronger settings sections

### 2026-03-25 (live event stream / follow baseline)
- daemon/runtime 에 `events.follow` long-poll surface 를 추가해 마지막 event id 이후의 새 이벤트만 기다려 반환할 수 있게 만들었다.
- CLI `ham events` 에 `--follow`, `--after-id`, `--wait-ms` 를 추가해 기존 snapshot 조회 외에 follow mode 로 새 이벤트를 계속 읽을 수 있게 했다.
- Swift daemon client 도 same follow command 를 호출할 수 있도록 `followEvents(afterEventID:limit:waitMilliseconds:)` surface 를 추가하고, Swift tests 로 request encoding/round-trip 을 보호했다.
- Go tests 로 follow-after-cursor behavior 와 CLI event formatting helper 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer attached metadata sync, stronger settings sections, higher-fidelity event-driven UI updates

### 2026-03-25 (richer attached metadata sync baseline)
- attached agent model 에 `session_title` / `session_is_active` metadata 를 추가하고, daemon attached refresh path 가 iTerm session listing 에서 이 값을 동기화하도록 확장했다.
- attached disconnect/reconnect refresh 와 같은 path 에서 title/current-session marker 를 함께 갱신해 backend snapshot 이 richer attached metadata 를 담도록 만들었다.
- Swift `Agent` decoding 과 detail pane 도 이 metadata 를 표시하도록 맞췄고, attached agent detail 에 current/background iTerm session 힌트를 노출했다.
- Go/Swift tests 로 attached metadata sync, decoding, and UI surface behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: stronger settings sections, higher-fidelity event-driven UI updates, richer attached cwd/activity metadata

### 2026-03-25 (stronger settings sections baseline)
- daemon settings schema 에 `integrations.iterm_enabled` section 을 추가해 appearance 다음 단계의 non-notification settings 영역을 확장했다.
- CLI `ham settings integrations --iterm-enabled=...` 와 Swift menu bar `Integrations` section 을 연결해 iTerm integration toggle 을 daemon-backed settings 로 수정할 수 있게 했다.
- current UI behavior 도 이 설정을 일부 존중하도록 연결해 iTerm integration 이 꺼져 있으면 attachable session preview 와 session-open action 을 막도록 만들었다.
- Go/Swift tests 로 settings persistence, daemon round-trip, and Swift integration-gated UI behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: event-driven UI refresh baseline, richer attached cwd/activity metadata, stronger settings sections

### 2026-03-25 (event-driven UI refresh baseline)
- `MenuBarViewModel` 에 daemon `followEvents` 를 사용하는 background lane 을 추가해 polling 외에도 새 이벤트가 도착하면 즉시 refresh 를 트리거할 수 있게 했다.
- 기존 polling loop 는 safety net 으로 유지하고, follow lane 은 latest event id 이후의 새 이벤트가 생겼을 때 summary/events reload 를 촉발하는 방식으로 추가했다.
- Swift tests 로 follow lane 이 새 이벤트 도착 시 summary refresh 를 다시 수행하는지 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer attached cwd/activity metadata baseline, stronger settings sections, higher-fidelity event-driven UI updates

### 2026-03-25 (richer attached cwd/activity metadata baseline)
- iTerm session listing baseline 을 확장해 session tty 를 읽고, `ps` + `lsof` 기반 heuristic 으로 foreground command/activity 와 working directory 를 attachable session metadata 로 보강했다.
- attached agent refresh path 가 title/current-session marker 뿐 아니라 tty, working directory, activity metadata 도 함께 sync 하도록 확장했다.
- Swift `Agent` decoding 과 detail UI 도 tty / working directory / activity 를 표시하도록 맞췄고, Go/Swift tests 로 sync + decode + UI surface behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: higher-fidelity event-driven UI updates, stronger settings sections, richer attached shell-state fidelity

### 2026-03-25 (higher-fidelity event-driven UI update baseline)
- `MenuBarViewModel.followLatestEvents` 가 follow-event 도착 후 full refresh 대신 lighter snapshot/agent/event merge path 를 사용하도록 바꿨다.
- follow lane 에서는 기존 summary events 와 새 followed events 를 merge 하고, settings / attachable sessions / permission state 는 polling lane 에 맡겨 event-driven refresh cost 를 줄였다.
- Swift tests 로 follow lane 이 새 이벤트를 반영하면서도 settings/event fetch 호출을 불필요하게 반복하지 않는지 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer attached shell-state fidelity, stronger event semantics, lower-latency UI updates

### 2026-03-25 (richer attached shell-state fidelity baseline)
- attached shell-state metadata 를 pid + full command string까지 확장해 iTerm tty 기반 heuristic 이 richer process context 를 담도록 만들었다.
- iTerm adapter 는 `ps` 결과에서 pid/command 를 읽고, attached refresh path 는 이를 agent snapshot 으로 동기화하도록 확장했다.
- Swift decoding/detail UI 도 pid/command 를 함께 표시하도록 맞췄고, Go/Swift tests 로 sync + decode + UI surface behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: stronger event semantics, lower-latency UI updates, richer attached shell-state fidelity

### 2026-03-25 (stronger event semantics baseline)
- daemon event taxonomy 를 확장해 role update, notification policy update, tracking removal, attached disconnect/reconnect 같은 richer lifecycle/admin events 를 기록하도록 만들었다.
- runtime mutation paths 가 event log 에 해당 summaries 를 append 하도록 연결해 activity feed 가 `agent.registered` 하나에만 머물지 않게 했다.
- Go tests 로 mutation/disconnect paths 가 기대한 richer event types 를 남기는지 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: lower-latency UI updates, richer event-driven UI semantics, richer attached shell-state fidelity

### 2026-03-25 (lower-latency UI update baseline)
- `MenuBarViewModel.followLatestEvents` 가 follow-event 이후 full refresh 대신 agent fetch + local summary rebuild path 를 사용하도록 바꿨다.
- follow lane 는 merged recent events 와 fetched agents 만으로 counts 를 다시 계산하고, snapshot/settings/attachable-session reload 는 polling lane 에 남겨 hot-path wakeup cost 를 더 줄였다.
- Swift tests 로 follow lane 이 새 이벤트를 반영하면서도 snapshot/settings/event fetch 수를 과도하게 늘리지 않는지 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer event-driven UI semantics, lower-latency visual updates, richer attached shell-state fidelity

### 2026-03-25 (richer event-driven UI semantics baseline)
- `HamAppServices` 에 event presentation helper 를 추가해 richer daemon event types를 UI label/emphasis semantics로 매핑하도록 만들었다.
- menu bar detail 의 recent event section 이 raw event type 문자열만 보여주지 않고, richer semantic badge (`Disconnected`, `Reconnected`, `Notifications`, 등) 를 함께 렌더링하도록 바꿨다.
- Swift tests 로 event presentation mapping 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: lower-latency visual updates, stronger feed semantics, richer attached shell-state fidelity

### 2026-03-25 (lower-latency visual updates baseline)
- `MenuBarViewModel` 이 latest presented event semantics 를 바로 드러낼 수 있도록 latest-event presentation/symbol surface 를 추가했다.
- menu bar status line 이 warning/info/positive recent event에 맞춰 간단한 indicator prefix 를 붙이도록 만들고, popover 상단에도 compact latest-event banner 를 추가했다.
- Swift tests 로 warning event 가 status line visual cue 로 바로 반영되는지 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: stronger feed semantics, lower-latency visual polish, richer attached shell-state fidelity

### 2026-03-25 (lower-latency visual polish baseline)
- latest event/detail feed rendering 을 다듬어 known event types 는 raw technical code 대신 semantic label + relative-time/cleaner secondary text를 우선 보여주도록 정리했다.
- event presentation mapping 에 `showsTechnicalType` 를 추가해 unknown/custom events 만 raw type 을 계속 노출하게 만들고, known events 는 더 조용한 visual hierarchy 로 정리했다.
- Swift tests 로 unknown event fallback 과 known event semantic treatment 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: stronger feed semantics, lower-latency visual polish, richer attached shell-state fidelity

### 2026-03-25 (stronger feed semantics baseline)
- `AgentEventPresenter` 에 recent event summary aggregation 을 추가해 event feed 가 richer event semantics를 label/emphasis 기준으로 구조적으로 묶을 수 있게 했다.
- agent detail 의 recent event section 에 grouped summary chips 를 추가해 raw chronological list 위에 “Disconnected 2”, “Reconnected 1” 같은 semantic grouping 을 바로 보여주도록 만들었다.
- popover 상단에도 전체 recent activity summary chips 를 추가해 현재 feed shape 를 더 빨리 훑어볼 수 있게 했다.
- Swift tests 로 event summary grouping 과 view-model recent-event summary surface 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: observed lifecycle event baseline, lower-latency visual polish, richer attached shell-state fidelity

### 2026-03-25 (observed lifecycle event baseline)
- observed refresh path 가 status 변화가 실제로 일어났을 때 `agent.status_updated` event 를 남기도록 확장했다.
- `AgentEventPresenter` 도 새 event type 을 semantic `Status` label 로 매핑해 feed/UI semantics 와 맞추었다.
- Go tests 로 observed refresh 뒤 status-update event emission 을 보호하고, Swift tests 로 새 event presentation mapping 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: stronger feed semantics, lower-latency visual polish, richer attached shell-state fidelity

### 2026-03-25 (attached shell-state heuristic refinement baseline)
- iTerm tty heuristic 이 shell process와 foreground tool이 함께 보일 때 non-shell foreground command 를 우선 선택하도록 정교화했다.
- shell-only tty 인 경우에는 raw shell command 를 유지하되 activity label 은 generic `shell` 로 보정해 attached metadata noise 를 줄였다.
- Go adapter tests 로 shell noise vs foreground tool prioritization과 shell fallback behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: lower-latency visual polish, stronger feed semantics, richer attached shell-state fidelity

### 2026-03-25 (status reason baseline)
- agent schema 에 `status_reason` 을 추가해 status/confidence 옆에 짧은 설명 문자열을 함께 보관할 수 있게 했다.
- observed inference 와 attached transition paths 가 reason 을 채우도록 확장하고, registration baseline 도 간단한 initial reason 을 남기도록 정리했다.
- Swift `Agent` decoding 과 detail UI 도 reason 을 표시하도록 맞췄고, Go/Swift tests 로 decode + observed/attached reason behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: confidence/reason refinement baseline, stronger feed semantics, lower-latency visual polish

### 2026-03-25 (confidence/reason refinement baseline)
- Swift presentation layer 에 `High/Medium/Low confidence` wording, low-confidence `likely ...` 상태 문구, concise confidence summary 문구를 추가했다.
- agent list/detail 이 raw percentage만 보여주지 않고 confidence level + softened low-confidence wording + short reason 을 함께 보여주도록 정리했다.
- Swift tests 로 medium/low confidence wording 과 reason summary rendering behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: attention queue baseline, stronger feed semantics, lower-latency visual polish

### 2026-03-25 (confidence/reason refinement baseline)
- `MenuBarViewModel` 에 confidence level / confidence summary / low-confidence `likely` wording helpers 를 추가해 inferred states를 더 절제된 표현으로 보여주도록 만들었다.
- agent list/detail UI 가 raw percentage만 보여주지 않고 `High/Medium/Low confidence` 표현과 concise reason 문구를 함께 보여주도록 정리했다.
- Swift tests 로 medium/low confidence wording 과 reason display behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: stronger feed semantics, lower-latency visual polish, richer attached shell-state fidelity

### 2026-03-25 (attention queue baseline)
- `MenuBarViewModel` 에 attention priority helper 를 추가해 `error`, `waiting_input`, `disconnected` 상태의 agent 를 별도 attention queue 로 분리하고 우선순위/최근성 기준으로 정렬하도록 만들었다.
- menu bar popover 상단에 `Needs Attention` 섹션을 추가해 attention-required agent 를 일반 agent list 보다 먼저 보이게 만들었다.
- Swift tests 로 attention agent ordering/filtering behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: severity-aware feed ordering, runtime lifecycle transition baseline, richer feed semantics

### 2026-03-25 (attention queue follow-up)
- attention queue row subtitle 이 status reason 을 포함하도록 정리해, urgent agent 가 왜 attention 대상인지 list-level 에서 바로 보이게 만들었다.
- Swift tests 로 urgent agent subtitle wording 이 expected attention context 를 포함하는지 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: stronger feed semantics, lower-latency visual polish, richer attached shell-state fidelity

### 2026-03-25 (CLI confidence/reason visibility baseline)
- `ham list` human-readable output 이 agent mode/status/confidence/reason 을 함께 보여주도록 정리해, JSON 없이도 low-confidence inferred state 와 그 근거를 바로 읽을 수 있게 만들었다.
- low-confidence status 는 Swift detail wording 과 맞춰 `likely ...` 로 soften 하고, confidence 는 `high|medium|low NN%` 형태로 노출하도록 공통 helper 를 추가했다.
- `ham status` human-readable output 도 `attention=<N>` 요약을 포함하게 만들어 waiting/error/disconnected agent 수를 한 줄에서 바로 확인할 수 있게 했다.
- Go tests 로 human-readable `ham list` / `ham status` output 과 JSON output contract 가 각각 drift 하지 않게 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: CLI attention detail baseline, richer lifecycle coverage, broader operator-facing CLI scanability

### 2026-03-25 (CLI attention detail baseline)
- `ham list` human-readable output 이 error / waiting_input / disconnected agent 를 먼저 보여주도록 정리해, 긴 list 에서도 urgent agent 를 위에서 바로 찾을 수 있게 만들었다.
- `ham status` human-readable output 이 attention count 아래에 urgent agent detail line 을 붙여, status / confidence / reason 을 별도 JSON 조회 없이 함께 읽을 수 있게 만들었다.
- severity (`error` > `waiting_input` > `disconnected`) 와 최근 event 시각 기준으로 urgent ordering 을 맞추고, JSON output 은 기존 machine-readable shape 를 그대로 유지했다.
- Go tests 로 attention-first list ordering, urgent detail line ordering, human-vs-JSON status/list behavior 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, broader operator-facing CLI scanability, daemon-backed attention model

### 2026-03-25 (CLI attention breakdown baseline)
- `ham status` human-readable output 이 summary 아래에 `attention_breakdown` line 을 추가해, error / waiting_input / disconnected 분포를 한 번에 스캔할 수 있게 만들었다.
- breakdown 은 human-readable path 에만 붙이고 JSON status output 은 기존 machine-readable shape 를 유지하도록 분리했다.
- Go tests 로 breakdown line presence, JSON non-leakage, urgent detail ordering과의 공존을 함께 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, broader operator-facing CLI scanability, daemon-backed attention model

### 2026-03-25 (CLI stop baseline)
- CLI spec 에 맞춰 `ham stop <agent>` 를 추가하고, 현재 baseline 의미를 existing remove/tracking-removal path 에 매핑했다.
- stop 결과는 human path 에서 `stopped tracking <agent>` 로 보여주고, JSON path 에서는 `{ "removed": "<agent>" }` 만 반환하게 분리해 machine contract 를 단순하게 유지했다.
- Go tests 로 stop input parsing 과 human/JSON result rendering 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: broader operator-facing CLI scanability, richer lifecycle coverage, daemon-backed attention model

### 2026-03-25 (CLI logs baseline)
- CLI spec 에 맞춰 `ham logs <agent>` baseline 을 추가하고, 현재 daemon event log 를 agent id 기준으로 client-side filtering 해서 per-agent recent log view 를 제공하게 만들었다.
- logs 는 human path 에서 기존 event row formatting 을 재사용하고, JSON path 에서는 existing newline-delimited event JSON contract 를 그대로 유지한다.
- fetch 는 recent-window best-effort baseline 으로 시작해, requested limit 보다 넓은 recent event window 를 먼저 읽고 그 안에서 해당 agent event 만 tail-limit 하도록 정리했다.
- Go tests 로 logs input parsing, per-agent filtering/tail limiting, fetch-window floor 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: broader operator-facing CLI scanability, richer lifecycle coverage, daemon-backed attention model

### 2026-03-25 (CLI list summary baseline)
- human-readable `ham list` 상단에 summary line 을 추가해 total / attention / managed / attached / observed count 를 먼저 스캔할 수 있게 만들었다.
- summary line 은 human path 에만 붙고, JSON list output 은 기존 raw agent array contract 를 그대로 유지한다.
- Go tests 로 summary wording, attention-first ordering과의 공존, JSON non-leakage 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed attention model, severity-aware feed scanning

### 2026-03-25 (CLI doctor baseline)
- CLI spec 에 맞춰 `ham doctor` baseline 을 추가하고, current local installation 상태를 socket/state/event/settings path 기준으로 바로 확인할 수 있게 만들었다.
- doctor 는 human path 에서 root source + resolved paths + socket reachability/파일 존재 상태를 읽기 쉬운 줄 단위로 보여주고, JSON path 에서는 같은 정보를 structured payload 로 반환한다.
- baseline 은 local path/socket inspection 에 집중하고, HAM_AGENTS_HOME override 와 default app-support 경로를 함께 진단하도록 정리했다.
- Go tests 로 env-root report gathering, human/JSON render contract 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed attention model, severity-aware feed scanning

### 2026-03-25 (severity-aware feed scanning baseline)
- Recent Activity 영역에 severity summary chip row 를 추가해 warning / positive / info / neutral 분포를 label-group chips 보다 먼저 스캔할 수 있게 만들었다.
- agent detail 의 Recent Events 영역도 같은 severity summary row 를 먼저 보여주도록 맞춰, 개별 agent feed 에서도 현재 분위기를 더 빨리 읽을 수 있게 만들었다.
- 기존 label-group chips 와 recent-event ordering 은 유지하고, severity summary 는 presentation-level 집계로만 추가했다.
- Swift tests 로 severity chip grouping/order 와 view model severity chip surface 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed attention model, printEvents empty-json writer consistency

### 2026-03-25 (event JSON writer consistency baseline)
- `printEvents(..., asJSON: true)` empty case 가 stdout 으로 새지 않고 caller-provided writer 를 그대로 사용하도록 정리했다.
- empty JSON event output regression test 를 추가해, logs/events helper 가 buffer/pipe writer 를 쓰는 경로에서도 `[]` 가 올바른 target 에 기록되도록 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed attention model, CLI/UI polish follow-up

### 2026-03-25 (daemon-backed attention summary baseline)
- daemon snapshot payload 에 `attention_count` 와 `attention_breakdown` 을 추가해, attention summary 가 Swift-side 재계산만이 아니라 Go snapshot contract 에도 직접 실리게 만들었다.
- Swift snapshot decoding 과 menu bar summary layer 를 이 새 payload 에 맞춰 정리하고, top summary badge row 에 daemon-backed `Attn` count 를 노출했다.
- follow-events fallback summary builder 는 기존 agent list 로 같은 attention count 를 재구성하도록 유지해, snapshot-based refresh 와 partial-update refresh 가 같은 summary shape 를 공유하게 만들었다.
- Go/Swift tests 로 runtime snapshot attention summary, daemon payload decoding, summary service / view model attention count surface 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed attention breakdown UI, CLI/UI polish follow-up

### 2026-03-25 (daemon-backed attention breakdown UI baseline)
- menu bar top summary 영역이 daemon-backed attention breakdown 을 summary chips 로 직접 보여주도록 정리해, error / needs input / disconnected 분포를 badge row 바로 아래에서 읽을 수 있게 만들었다.
- 이 breakdown UI 는 `HamMenuBarSummary` 와 `MenuBarViewModel.topSummaryAttentionBreakdownChips` seam 을 통해 노출되며, follow-events partial refresh path 도 fetched agents 로 같은 breakdown 을 재구성하도록 보호했다.
- 기존 row ordering / attention queue / status line semantics 는 유지하고, top summary 쪽에 additive scanability layer 만 추가했다.
- Swift tests 로 summary service breakdown propagation, refresh/follow path breakdown surface, top summary chip seam 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, CLI/UI polish follow-up, daemon-backed attention ordering

### 2026-03-25 (daemon-backed attention ordering baseline)
- daemon snapshot contract 에 `attention_order` 를 추가해, urgency ordering 이 Swift-only heuristic 이 아니라 daemon summary contract 에도 직접 실리게 만들었다.
- Swift summary/view model 은 daemon-provided attention order 를 우선 사용하고, missing/partial order 일 때만 기존 priority/recency fallback 을 쓰도록 정리했다.
- follow-events partial refresh path 는 fetched agents 로 같은 ordering shape 를 재구성해, initial snapshot refresh 와 event-follow refresh 가 일관된 attention ordering 을 유지하게 만들었다.
- Go/Swift tests 로 runtime snapshot attention ordering, payload decoding default/order preservation, summary propagation, view model daemon-order preference 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, CLI/UI polish follow-up, daemon-backed attention subtitle model

### 2026-03-25 (daemon-backed attention subtitle baseline)
- daemon snapshot contract 에 `attention_subtitles` 를 추가해, urgent row subtitle wording 이 Swift-only composition 이 아니라 daemon attention contract 에도 직접 실리게 만들었다.
- Swift summary/view model 은 daemon-provided subtitle 을 우선 사용하고, older/partial payload 또는 follow-events local rebuild 경로에서는 기존 fallback wording 을 같은 shape 로 재구성하도록 정리했다.
- Go/Swift tests 로 runtime snapshot subtitle population, payload decoding default/subtitle preservation, summary propagation, refresh/follow path subtitle usage 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, CLI/UI polish follow-up, daemon-backed lifecycle summary

### 2026-03-25 (CLI status attention subtitle contract baseline)
- `ham status --json` 이 daemon-backed `attention_subtitles` map 을 함께 내보내도록 정리해, automation path 도 urgent row subtitle contract 를 직접 소비할 수 있게 만들었다.
- human `ham status` output 은 기존 terse summary/attention line 형태를 그대로 유지하고, richer subtitle data 는 additive JSON field 로만 노출되게 분리했다.
- Go tests 로 status JSON output 이 `attention_subtitles` 를 포함하면서 human summary wording 을 섞지 않는지 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: CLI ui baseline, richer lifecycle coverage, daemon-backed lifecycle summary

### 2026-03-25 (CLI ui baseline)
- spec-listed `ham ui` baseline 을 추가해 menu bar executable launch path 를 CLI 에서 직접 호출할 수 있게 만들었다.
- resolution 은 `HAM_UI_EXECUTABLE` override → current executable sibling `ham-menubar` → local SwiftPM build artifact → PATH lookup 순서로 정리했고, `--print` / `--json` 으로 planned target 을 확인할 수 있게 했다.
- actual launch path 는 resolved `ham-menubar` executable 을 detached start 하는 baseline 으로 시작한다.
- Go tests 로 environment override, build-artifact fallback, unexpected argument rejection 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle summary, CLI/UI polish follow-up

### 2026-03-25 (lifecycle-aware event presentation baseline)
- `agent.status_updated` presentation 이 generic `Status` badge 만 보여주지 않고 `Done`, `Needs Input`, `Error`, `Idle` 같은 lifecycle-aware label/emphasis 를 더 직접 반영하도록 정리했다.
- `agent.registered` 도 summary copy 를 바탕으로 `Managed`, `Attached`, `Observed` registration context 를 더 직접 보여주도록 만들어 feed summary chips 와 detail rows 가 registration mode 를 더 빨리 읽게 했다.
- Swift tests 로 lifecycle-aware presentation mapping 과 warning-status-update ordering behavior 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle summary, CLI/UI polish follow-up

### 2026-03-25 (daemon-backed event presentation hint baseline)
- daemon event payload 에 `presentation_label` / `presentation_emphasis` optional 필드를 추가해, known lifecycle/admin events 의 presentation hint 를 Go 쪽에서 함께 내보내도록 만들었다.
- runtime event append path 는 known event type + summary 에서 baseline hint 를 채우고, Swift `AgentEventPresenter` 는 이 daemon-provided hint 가 있으면 summary-string inference 보다 우선 사용하도록 정리했다.
- 이 변경은 additive contract 로 유지되어 older payload 는 기존 Swift inference path 를 그대로 타고, unknown events 는 여전히 technical fallback 을 유지한다.
- Go/Swift tests 로 stored event hint population, event payload decoding, hint-overrides-summary behavior 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle summary, CLI/UI polish follow-up

### 2026-03-25 (daemon-backed lifecycle summary baseline)
- daemon event payload 에 `presentation_summary` optional 필드를 추가해, lifecycle/admin event row 가 raw summary 문자열 대신 daemon-authored concise display summary 를 함께 받을 수 있게 만들었다.
- runtime event append path 는 known lifecycle summaries (`Status changed to …`, disconnect/reconnect 등) 에서 display-friendly summary 를 분리해 hint 로 채우고, Swift recent event rows 는 이 daemon-provided summary hint 가 있으면 우선 사용하도록 정리했다.
- older/partial payload 는 기존 raw `summary` 표시로 fallback 하므로 contract 는 additive 로 유지된다.
- Go/Swift tests 로 stored event presentation summary, event payload decoding, summary-hint override/fallback behavior 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle metadata, CLI/UI polish follow-up

### 2026-03-25 (daemon-backed lifecycle metadata baseline)
- daemon event payload 에 optional `lifecycle_status` / `lifecycle_mode` 필드를 추가해, known registration / status-transition events 가 summary-string parsing 없이도 lifecycle context 를 함께 전달하게 만들었다.
- runtime event append path 는 managed/attached/observed registration 과 status/disconnect/reconnect events 에 이 metadata 를 채우고, Swift presenter 는 daemon hint 가 없더라도 이 lifecycle metadata 를 summary-string inference 보다 먼저 사용하도록 정리했다.
- older/partial payload 는 기존 summary-string inference fallback 을 유지하므로 contract 는 additive 로 유지된다.
- Go/Swift tests 로 stored event lifecycle metadata, payload decoding, metadata-over-summary behavior 를 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle detail, CLI/UI polish follow-up

### 2026-03-25 (daemon-backed lifecycle reason baseline)
- daemon event payload 에 optional `lifecycle_reason` 필드를 추가해, known status/disconnect/reconnect/register events 가 lifecycle context 뿐 아니라 concise reason string 도 함께 전달하게 만들었다.
- runtime event append path 는 agent `StatusReason` 를 lifecycle-bearing events 에 함께 싣고, Swift payload decoding 도 이 structured reason 을 보존하도록 정리했다.
- older/partial payload 는 reason field 가 비어 있어도 기존 summary/presentation fallback 이 유지되므로 contract 는 additive 로 유지된다.
- Go/Swift tests 로 stored event lifecycle reason 과 payload decoding preservation 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: daemon-backed lifecycle detail, richer lifecycle coverage, CLI/UI polish follow-up

### 2026-03-25 (latest-event lifecycle detail baseline)
- top latest-event banner 도 raw recent event summary 대신 shared `AgentEventPresenter.displaySummary(...)` 경로를 사용하도록 정리해, detail list 와 같은 lifecycle reason/summary fallback 을 따라가게 만들었다.
- lifecycle reason 만 있는 event 에서는 low-confidence wording `(low confidence)` 까지 banner 에 그대로 반영되도록 맞췄다.
- Swift tests 로 latest event banner summary 가 daemon-backed lifecycle detail fallback 을 사용하는지 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: daemon-backed lifecycle detail follow-up, richer lifecycle coverage, CLI/UI polish follow-up

### 2026-03-25 (daemon-backed lifecycle detail baseline)
- Swift event detail summary 가 daemon-provided `presentation_summary` 가 없을 때도 `lifecycle_reason` 을 우선 사용하도록 정리해, status-transition row 가 raw `Status changed to ...` 문장보다 더 직접적인 이유를 보여주게 만들었다.
- lifecycle confidence 가 낮은 경우에는 detail summary 에 `(low confidence)` 를 붙여, daemon이 준 lifecycle reason 이라도 과한 확신처럼 읽히지 않게 만들었다.
- unknown/older payload 는 기존 raw `summary` fallback 을 그대로 유지한다.
- Swift tests 로 lifecycle reason 기반 summary fallback 과 low-confidence wording 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: daemon-backed lifecycle detail follow-up, richer lifecycle coverage, CLI/UI polish follow-up

### 2026-03-25 (latest-event lifecycle detail baseline)
- top latest-event banner 도 raw recent event summary 대신 `AgentEventPresenter.displaySummary(...)` 를 사용하도록 정리해, banner 가 generic `Status changed to ...` 문장보다 더 직접적인 lifecycle reason/summary 를 보여주게 만들었다.
- lifecycle reason only event 에서는 low-confidence wording까지 banner 에 그대로 반영되도록 맞췄다.
- Swift tests 로 latest event banner summary 가 daemon-backed lifecycle detail fallback 을 사용하는지 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: daemon-backed lifecycle detail follow-up, richer lifecycle coverage, CLI/UI polish follow-up

### 2026-03-25 (daemon-backed lifecycle detail follow-up)
- event detail summary fallback chain 을 한 단계 더 정리해, `presentation_summary` 가 없을 때 `lifecycle_reason` 를 우선 쓰고, low-confidence lifecycle case 에만 `(low confidence)` 를 붙이도록 만들었다.
- 이 fallback 은 detail rows 뿐 아니라 shared `displaySummary` 를 쓰는 다른 surfaces 와도 같은 wording 을 공유한다.
- Swift tests 로 lifecycle reason-first summary fallback 과 low-confidence wording 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle detail metadata, CLI/UI polish follow-up

### 2026-03-25 (low-confidence lifecycle event presentation baseline)
- lifecycle-aware event presentation 이 low-confidence lifecycle metadata/hint 를 받을 때 `Likely ...` label 을 붙이도록 정리해, warning/info/positive event 도 과한 확정처럼 읽히지 않게 만들었다.
- 이 완화는 daemon-provided presentation hint 와 lifecycle metadata path 모두에 적용되고, detail summary fallback 의 `(low confidence)` wording 과 함께 동작한다.
- Swift tests 로 low-confidence lifecycle presentation label softening 을 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle detail follow-up, CLI/UI polish follow-up

### 2026-03-25 (CLI human event detail baseline)
- human `ham events` / `ham logs` 출력도 raw `Status changed to ...` summary 대신 `presentation_summary` → `lifecycle_reason` → raw `summary` fallback chain 을 따르도록 정리했다.
- low-confidence lifecycle event 는 human event row 에서도 `(low confidence)` 를 붙여, CLI event feed 와 Swift event detail 이 같은 caution tone 을 공유하게 만들었다.
- Go tests 로 human event row 가 `presentation_summary` 와 low-confidence lifecycle reason fallback 을 우선 사용하는지 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle detail follow-up, CLI/UI polish follow-up

### 2026-03-25 (CLI event lifecycle reason contract baseline)
- `ham events --json` / `ham logs --json` 경로가 daemon-backed `lifecycle_reason` 필드를 그대로 유지하도록 tests 를 보강해, CLI automation path 도 structured lifecycle reason contract 를 읽을 수 있게 고정했다.
- filtering/tail limiting helper 가 lifecycle reason 을 떨어뜨리지 않는지, newline-delimited event JSON line output 이 lifecycle reason 을 계속 포함하는지 Go tests 로 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: daemon-backed lifecycle detail, richer lifecycle coverage, CLI/UI polish follow-up

### 2026-03-25 (CLI event lifecycle confidence contract baseline)
- `ham events --json` / `ham logs --json` 경로가 daemon-backed `lifecycle_confidence` 필드를 그대로 유지하도록 tests 를 보강해, CLI automation path 도 structured lifecycle confidence contract 를 읽을 수 있게 고정했다.
- filtering/tail limiting helper 가 lifecycle confidence 를 떨어뜨리지 않는지, newline-delimited event JSON line output 이 lifecycle confidence 를 계속 포함하는지 Go tests 로 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: daemon-backed lifecycle detail, richer lifecycle coverage, CLI/UI polish follow-up

### 2026-03-25 (CLI event lifecycle metadata contract baseline)
- `ham events --json` / `ham logs --json` 경로가 daemon-backed `lifecycle_status` / `lifecycle_mode` 필드를 그대로 유지하도록 tests 를 보강해, CLI automation path 도 event lifecycle metadata contract 를 읽을 수 있게 고정했다.
- filtering/tail limiting helper 가 lifecycle metadata 를 떨어뜨리지 않는지, newline-delimited event JSON line output 이 lifecycle metadata 를 계속 포함하는지 Go tests 로 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle detail, CLI/UI polish follow-up

### 2026-03-25 (CLI event presentation summary contract baseline)
- `ham events --json` / `ham logs --json` 경로가 daemon-backed `presentation_summary` 필드를 그대로 유지하도록 tests를 보강해, CLI automation path 도 concise lifecycle summary hint 를 읽을 수 있게 고정했다.
- filtering/tail limiting helper 가 `presentation_summary` 를 떨어뜨리지 않는지, newline-delimited event JSON line output 이 summary hint 필드를 계속 포함하는지 Go tests 로 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle metadata, CLI/UI polish follow-up

### 2026-03-25 (CLI event presentation hint contract baseline)
- `ham events --json` / `ham logs --json` 경로가 daemon-backed `presentation_label` / `presentation_emphasis` 필드를 그대로 유지하도록 테스트를 보강해, CLI automation path 도 event presentation hint contract 를 읽을 수 있게 고정했다.
- filtering/tail limiting helper 가 presentation hint 를 떨어뜨리지 않는지, newline-delimited event JSON line output 이 hint 필드를 계속 포함하는지 Go tests 로 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle summary, CLI/UI polish follow-up

### 2026-03-25 (CLI status attention subtitle contract baseline)
- `ham status --json` 이 daemon-backed attention subtitle map 도 함께 내보내도록 정리해, automation path 에서도 urgent row subtitle contract 를 읽을 수 있게 만들었다.
- human `ham status` output 은 기존 terse summary/attention line 형태를 유지하고, richer subtitle data 는 additive JSON field 로만 노출되게 분리했다.
- Go tests 로 status JSON output 이 `attention_subtitles` 를 포함하면서 human summary wording 을 섞지 않는지 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle summary, CLI/UI polish follow-up

### 2026-03-25 (CLI status attention contract baseline)
- `ham status --json` 이 daemon-backed attention summary fields (`attention_count`, `attention_breakdown`, `attention_order`) 를 함께 내보내도록 정리해, automation path 도 richer attention contract 를 읽을 수 있게 만들었다.
- human status output wording은 그대로 유지하고, JSON path 만 additive summary fields 를 받도록 분리해 existing CLI scanability와 automation contract 를 함께 유지했다.
- Go tests 로 status JSON output 이 attention fields 를 포함하면서도 human summary wording 을 섞지 않는지 고정했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: richer lifecycle coverage, daemon-backed lifecycle summary, CLI/UI polish follow-up

### 2026-03-25 (severity-aware feed ordering baseline)
- recent event feed ordering 을 severity-first, recency-second 로 정리해 warning/positive/info 계열 event 가 작은 recent-event window 에서 더 빠르게 보이도록 만들었다.
- `MenuBarViewModel.recentEvents` 가 `AgentEventPresenter` ordering helper 를 사용하도록 연결하고, Swift tests 로 warning event 가 informational row 앞에 오는 ordering 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: runtime lifecycle transition baseline, stronger feed semantics, richer attached shell-state fidelity

### 2026-03-25 (runtime lifecycle transition baseline)
- attached disconnect/reconnect 와 observed status updates 가 shared `Status changed to …` summary wording 을 사용하도록 정리해 lifecycle feed phrasing 을 일관화했다.
- Go tests 로 observed/attached lifecycle event summary wording 을 고정해, event taxonomy 뿐 아니라 transition copy 도 drift 하지 않게 했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: stronger feed semantics, richer attached shell-state fidelity, lower-latency visual polish

### 2026-03-25 (runtime coordinator baseline)
- registry mutation/save/event-append 경로의 중복을 줄이기 위해 shared mutation/persistence helpers (`mutateAgent`, `saveAgentsAndEvents`) 를 추가했다.
- notification policy update, role update, removal, attached refresh, observed refresh 가 같은 persistence/event append boundary 를 재사용하도록 정리했다.
- managed / attached / observed registration 도 shared `registerAgent` helper 로 정리해 registration path duplication 까지 줄였다.
- 기존 Go/Swift verification 을 다시 돌려 refactor 후 behavior 가 유지되는지 확인했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: runtime transition consistency baseline, stronger feed semantics, lower-latency visual polish

### 2026-03-25 (runtime coordinator follow-up)
- `RefreshAttached` 와 observed read/refresh paths 가 shared apply/persist helper 를 더 직접 재사용하도록 정리해 runtime-side mutation boundaries를 한 단계 더 통일했다.
- attached refresh no-op persistence test 와 snapshot/list-driven observed persistence tests 를 추가해 helper reuse 범위가 실제 behavior로 고정되도록 만들었다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: runtime lifecycle coverage follow-up, stronger feed semantics, lower-latency visual polish

### 2026-03-25 (runtime transition consistency baseline)
- observed refresh가 poll path뿐 아니라 list/snapshot read path에서도 같은 apply-and-persist helper 를 타도록 정리해 lifecycle evidence 가 호출 경로에 따라 달라지지 않게 했다.
- snapshot-driven observed refresh regression test 를 추가했고, explicit poll/list/snapshot 경로가 모두 `agent.status_updated` event 를 남기는지 보호했다.
- unchanged observed/attached refresh 가 extra lifecycle event 나 redundant save 를 만들지 않는지까지 regression coverage 를 넓혔다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: runtime lifecycle coverage follow-up, stronger feed semantics, lower-latency visual polish

### 2026-03-25 (richer attached shell-state fidelity follow-up)
- attached shell-state follow-up 으로 stale disconnect metadata 를 정리해, attached session 이 끊기면 tty/cwd/activity/pid/command 정보가 그대로 남아 misleading 하지 않게 만들었다.
- iTerm adapter 쪽도 shell-only noise 를 더 줄여 shell command 는 숨기고 foreground tool 신호가 있으면 그쪽을 우선 사용하도록 보정했다.
- Go tests 로 disconnect 시 stale shell-state clearing 과 shell-noise normalization behavior 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: runtime coordinator follow-up, stronger feed semantics, lower-latency visual polish

### 2026-03-25 (runtime coordinator follow-up: no-op refresh consistency)
- attached refresh도 observed refresh와 같은 `applyRefreshedAgents` no-op guard 를 재사용하도록 정리해, 실제 변화가 없을 때는 불필요한 persistence/write 를 건너뛰게 만들었다.
- Go regression test 로 unchanged attached refresh 가 extra save 를 만들지 않는지 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: runtime lifecycle coverage follow-up, stronger feed semantics, lower-latency visual polish

### 2026-03-25 (richer lifecycle coverage follow-up)
- `agent.removed` event 도 removed agent 의 mode/status/reason/confidence 를 함께 보존하도록 정리해, tracking removal 이후에도 downstream consumer 가 마지막 lifecycle context 를 잃지 않게 만들었다.
- Go regression test 로 removed event 가 lifecycle metadata/detail 을 유지하는지 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: daemon-backed lifecycle detail follow-up, stronger feed semantics, lower-latency visual polish

### 2026-03-25 (removed-event lifecycle detail follow-up)
- daemon removal event 의 `presentation_summary` 가 generic `Tracking stopped.` 대신 마지막 lifecycle status/reason 을 반영한 detail(`Stopped tracking while ...`)을 제공하게 정리했고, `waiting_input` 같은 raw status 는 `waiting for input` 으로 humanize 했다.
- human `ham events` / `ham logs` 도 같은 daemon-backed removal detail 을 그대로 보여주도록 유지했다.
- Go regression tests 로 removed event summary generation과 human CLI rendering 을 함께 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/cmd/ham ./go/internal/runtime` ✅

### 2026-03-25 (observed inference keyword refinement baseline)
- observed inference 가 generic `?` heuristic 만 보지 않고 `waiting for input`, `need input`, `all tests passed`, `task complete` 같은 explicit 문구를 더 직접 해석하도록 정리했다.
- observed mode confidence 는 여전히 low-to-mid 범위에 머물게 두되, explicit signal 에는 generic keyword보다 높은 confidence/reason/summary 를 부여했다.
- Go regression tests 로 explicit input/completion signal detection 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed inference precedence guard baseline)
- observed inference 가 `0 failed`, `no error`, `not completed` 같은 obvious negation 문맥에 덜 흔들리도록 precedence guard 를 추가했다.
- explicit completion/input/error phrase 는 계속 우선 처리하되, negated generic substring 은 thinking fallback 을 방해하지 않게 정리했다.
- Go regression tests 로 `0 failed` / `no error` / `not completed` / `don't need input` false positive guard 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed inference latest-line precedence baseline)
- observed inference 가 mixed log 전체를 동일 가중치로 보지 않고 최신 non-empty line 을 먼저 해석하게 정리했다.
- 그래서 오래된 `error`/`waiting for input` line 이 더 최신 `all tests passed` / `don't need input anymore` line 을 덮지 않게 만들었다.
- Go regression tests 로 latest-line precedence 와 mixed-log fallback 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed inference continuation-line guard baseline)
- 최신 line 이 explicit signal 은 아니어도 `continuing`, `still working`, `processing` 같은 continuation phrase 면 stale full-log signal fallback 을 억제하도록 정리했다.
- 그래서 오래된 waiting/error line 뒤에 이어진 neutral progress line 이 있으면 recent-activity 기반 thinking fallback 이 유지된다.
- Go regression test 로 continuation-line guard 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed continuation summary baseline)
- continuation phrase 로 thinking fallback 에 들어간 경우 generic freshness 문장 대신 `Continuation-like output detected.` / `Observed continuing output.` 을 보여주게 정리했다.
- 그래서 observed thinking 상태가 단순 최근 출력과 실제 continuation line 을 더 잘 구분한다.
- Go regression tests 로 continuation summary 와 generic freshness fallback 분리를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed tool-read inference baseline)
- observed 로그의 explicit tool-like line 을 `running_tool`, reading/analyzing line 을 `reading` 상태로 추론하는 baseline 을 추가했다.
- 그래서 observed mode 도 generic thinking 전에 spec 상태 집합 일부를 더 직접 반영할 수 있게 됐다.
- Go regression tests 로 tool/read inference baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (tool-read event presentation baseline)
- daemon event presentation hint 와 Swift presenter 가 `running_tool` / `reading` 상태를 각각 `Running Tool` / `Reading` 으로 더 직접 보여주게 정리했다.
- 그래서 observed tool/read inference 가 activity feed 에서 generic `Status` 로 다시 뭉개지지 않게 됐다.
- Go/Swift regression tests 로 tool-read event presentation baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/runtime` ✅

### 2026-03-25 (thinking-sleeping event presentation baseline)
- daemon event presentation hint 와 Swift presenter 가 `thinking` / `sleeping` 상태도 각각 `Thinking` / `Sleeping` 으로 직접 보여주게 정리했다.
- 그래서 observed recent-activity / idle transition 이 feed 에서 generic `Status` 로 보이지 않게 됐다.
- Go/Swift regression tests 로 thinking-sleeping event presentation baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/runtime` ✅

### 2026-03-25 (humanized status label baseline)
- human CLI 와 Swift status display 가 raw underscore status 대신 `needs input`, `running tool` 같은 더 사람 친화적인 wording 을 사용하게 정리했다.
- JSON/status contracts 는 그대로 두고, human-facing text 만 바꿔 operator scanability 를 높였다.
- Go/Swift regression tests 로 humanized status label baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/cmd/ham` ✅

### 2026-03-25 (attention subtitle humanization baseline)
- daemon-generated attention subtitle 도 raw `waiting_input` 대신 `needs input` 같은 humanized status wording 을 사용하게 정리했다.
- Swift attention subtitle path 와 daemon-provided urgent subtitle wording 이 다시 정렬되도록 테스트 기대값도 함께 갱신했다.
- Go/Swift regression tests 로 attention subtitle humanization baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/runtime ./go/cmd/ham` ✅

### 2026-03-25 (notification fallback humanization baseline)
- notification candidate 가 summary 없이 fallback body 를 만들 때도 raw underscore status 대신 humanized status wording 을 사용하게 정리했다.
- 그래서 `waiting_input at /tmp/app` 같은 표현이 `needs input at /tmp/app` 으로 더 자연스럽게 보이게 됐다.
- Swift regression test 로 notification fallback humanization baseline 을 보호했다.
- 검증:
  - `swift test --filter StatusChangeNotificationEngineTests --disable-sandbox` ✅

### 2026-03-25 (human attention breakdown wording baseline)
- human `ham status` 의 attention breakdown line 이 raw `waiting_input` 대신 더 읽기 쉬운 `needs_input` wording 을 쓰도록 정리했다.
- JSON `attention_breakdown.waiting_input` contract 는 그대로 유지해서 automation path 를 깨지 않게 했다.
- Go regression tests 로 human attention breakdown wording baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/cmd/ham` ✅

### 2026-03-25 (observed thinking phrase inference baseline)
- observed 로그의 explicit `thinking` / `planning` / `investigating` 류 line 이 generic recent-output fallback 전에 `thinking` 상태로 직접 추론되게 정리했다.
- 그래서 continuation phrase 와 plain freshness fallback 사이에 더 설명적인 thinking-like heuristic layer 가 생겼다.
- Go regression test 로 observed thinking phrase inference baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed status summary alignment baseline)
- observed `agent.status_updated` event summary 가 raw reason 보다 `LastUserVisibleSummary` 를 우선 사용하게 정리했다.
- 그래서 event feed / CLI event row wording 이 `Observed question-like output.` / `Observed error-like output.` 같이 observed inference surface 와 더 직접적으로 맞춰졌다.
- Go regression tests 로 observed status summary alignment baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/runtime` ✅

### 2026-03-25 (observed sleeping phrase inference baseline)
- observed 로그의 explicit `idle` / `paused` / `waiting for changes` 류 문구가 age-based staleness fallback 전에 직접 `sleeping` 으로 추론되게 정리했다.
- 그래서 최근에 갱신된 로그라도 명시적으로 쉬는 상태를 말하면 generic freshness/thinking 으로 보지 않고 sleeping-like state 로 반영된다.
- Go regression test 로 observed sleeping phrase inference baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed booting phrase inference baseline)
- observed 로그의 explicit `starting up` / `initializing` / `booting` 류 문구가 thinking/freshness fallback 전에 직접 `booting` 으로 추론되게 정리했다.
- 그래서 observed mode 도 spec 상태 집합의 `booting` 을 좀 더 직접 반영할 수 있게 됐다.
- Go regression test 로 observed booting phrase inference baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed idle phrase inference baseline)
- observed 로그의 explicit `ready` / `idle` / `standing by` 류 문구가 `sleeping` stale fallback 대신 직접 `idle` 로 추론되게 정리했다.
- 그래서 observed mode 에서 explicit idle wording 과 paused/stale sleeping wording 이 더 잘 분리된다.
- Go regression tests 로 observed idle phrase inference baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (observed disconnected phrase inference baseline)
- observed 로그의 explicit `disconnected` / `offline` / `session lost` 류 문구가 file-missing fallback 전에 직접 `disconnected` 로 추론되게 정리했다.
- disconnected negation guard 는 `connected` substring false-positive 를 피하도록 더 좁게 다듬었다.
- Go regression test 로 observed disconnected phrase inference baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/inference` ✅

### 2026-03-25 (booting event presentation baseline)
- daemon event presentation hint 와 Swift presenter 가 `booting` 상태도 `Booting` 으로 직접 보여주게 정리했다.
- 그래서 observed booting inference 가 activity feed 에서 generic `Status` 로 다시 뭉개지지 않게 됐다.
- Go/Swift regression tests 로 booting event presentation baseline 을 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./go/internal/runtime` ✅
  - `swift test --filter EventPresentationTests --disable-sandbox` ✅
