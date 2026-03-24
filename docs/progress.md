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
- Swift tests 로 event summary grouping 과 view-model recent-event summary surface 를 보호했다.
- 검증:
  - `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test ./...` ✅
  - `swift build --disable-sandbox` ✅
  - `swift test --disable-sandbox` ✅
- 다음 우선순위 후보: lower-latency visual polish, stronger feed semantics, richer attached shell-state fidelity
