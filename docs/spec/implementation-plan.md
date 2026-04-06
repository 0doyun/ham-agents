# ham-agents 구현 플랜

> Step 3 산출물 | 2026-04-06 작성 | docs/spec/ 기획서 기반

---

## Build & Test Convention

모든 빌드/테스트 명령은 **레포 루트** (`/Users/User/projects/ham-agents/`) 에서 실행한다. go.mod 이 루트에 위치하며 모듈 경로는 `github.com/ham-agents/ham-agents` 이다.

| 작업 | 명령 |
|------|------|
| 빌드 | `go build ./go/cmd/ham ./go/cmd/hamd` |
| 단위 테스트 | `go test ./... -count=1 -short` |
| 레이스 테스트 | `go test ./... -race -count=1` |

**주의**: `./go/...` 는 잘못된 패턴이다. 루트 기준 `./...` 이 정답.

---

## 목차

1. [Phase 분류 및 우선순위](#1-phase-분류-및-우선순위)
2. [Phase 1 상세 태스크](#2-phase-1-상세-태스크)
3. [Phase 2 상세 태스크](#3-phase-2-상세-태스크)
4. [Phase 3 상세 태스크](#4-phase-3-상세-태스크)
5. [에이전트 팀 구성 가이드](#5-에이전트-팀-구성-가이드)
6. [실행 프롬프트](#6-실행-프롬프트)

---

## 1. Phase 분류 및 우선순위

### Phase 1: Mission Control MVP (CLI + 메뉴바 확장)

**실행 순서**: P1-0 → P1-1 → (P1-2 ∥ P1-3) → 조건부 P1-4

| 단계 | 기능 | 예상 커밋 수 | 범위 |
|------|------|-------------|------|
| P1-0 | 신뢰성 기반 다지기 | 4-5 | 잔존 버그 수정, Swift/Go enum 동기화, contract test |
| P1-1 | 이벤트 스키마 확장 + Artifact Capture | 3-4 | Event 구조체 확장, ArtifactStore 신규, hook handler 수정 |
| P1-2 | 실시간 Session Graph | 2-3 | SessionGraph 타입, CLI --graph, 메뉴바 트리 뷰 |
| P1-3 | Notification Inbox (읽기 전용) | 3-4 | InboxItem/InboxManager, IPC 커맨드 2개, CLI/메뉴바 UI |
| P1-4 | 비용/토큰 텔레메트리 v1 (조건부) | 2-3 | ADR-3 조사 결과에 따라 시나리오 분기 (A/B/C). Scenario C 시 Phase 2 이관 |

#### Scope Gate (P1-4 Cost Telemetry)

**목적**: P1-4 를 Phase 1 에 포함할지, Phase 2 로 이관할지를 결정론적으로 판정한다.

- **결정 주체**: Phase 1 리드 (ham-agents 프로젝트 메인테이너)
- **트리거 아티팩트**: `docs/decisions/ADR-3-cost-telemetry.md` 가 `status: accepted` 로 커밋될 때 게이트 판정
- **데드라인**: Phase 1 킥오프 후 1주 이내. 판정이 그 전에 나오지 않으면 자동으로 Scenario C (Phase 2 이관) 로 fallback
- **판정 분기**:
  - **Scenario A — hook 에서 cost 데이터 확보 가능**: P1-4 Phase 1 포함, 예상 +3-4 커밋
  - **Scenario B — transcript 파싱으로 확보 가능**: P1-4 Phase 1 포함, 예상 +5-6 커밋 (난이도 상향)
  - **Scenario C — 현재 hook/transcript 로 불가능**: P1-4 를 Phase 2 로 이관, Phase 1 커밋 -2 차감
- **종속성**: 이 판정이 Phase 1 commit 총량, Phase 2 commit 총량, ADR-3 deadline 세 가지를 동시에 확정한다

**판정 상태**: PENDING (ADR-3 미작성)

**커밋 수 재산정 (Ralph Round 2)**: 기존 16-22 → **12-17** (P1-5 이관으로 4-5 커밋 감소). P1-4 Scenario C 확정 시 추가 -2 커밋.

### Phase 2: Terminal IDE (ham Studio)

**실행 순서**: Event Broadcast → Studio Window → Team Orchestrator → Playbooks → Git/CI → Review Loop

| 태스크 | 내용 | 예상 커밋 |
|-------|------|----------|
| P2-0 Event Broadcast (P1-5 이관) | EventBus 내부 fan-out | 2-3 |
| P2-1 Embedded PTY Runtime | hamd PTY 할당 + NDJSON 스트림 (ADR-2) + SwiftTerm 통합 | 6-8 |
| P2-2 Session Launcher | Studio New Session UI + workspace/model/playbook 선택 | 2-3 |
| P2-3 Approval Interception | PTY 블록 + approve 모달 + CommandAnswerPermission | 3-4 |
| P2-4 Agent Team Orchestrator | 기존 round 2 범위 | 3-4 |
| P2-5 Playbooks | 기존 round 2 범위 | 2-3 |
| P2-6 Git/CI webhook | 기존 round 2 범위 | 2-3 |
| P2-7 Review Loop | 기존 round 2 범위 | 2-3 |

**총 예상**: 20-27 커밋 (P1-5 이관으로 +2-3)

**커밋 수 재산정 (Ralph Round 3)**: 라운드 2 `20-27` → **22-31 커밋**. 내역:
- 라운드 2 기준 20-27 유지
- +2-3 from P2-1 Embedded PTY Runtime (hamd PTY 할당, NDJSON 스트림, SwiftTerm 통합)
- +0-1 from P2-3 Approval Interception (P2-1 위에 얹히므로 증분 작음)
- 총 +2-4, 신규 범위 `22-31 커밋`

### Phase 3: AgentOps Platform

**실행 순서**: Debugger → Policy → Memory → Maintenance → Marketplace

| 단계 | 기능 | 예상 커밋 수 | 범위 |
|------|------|-------------|------|
| P3-0 | Embedded DB 전환 | 3-4 | SQLite(modernc.org/sqlite) 도입, 기존 JSONL 마이그레이션 |
| P3-1 | AI Agent Debugger | 4-5 | SessionTrace, TraceBuilder, CLI replay, Studio UI |
| P3-2 | Org Policy Engine | 3-4 | PolicySet YAML, policy_engine, 위반 감지, CLI |
| P3-3 | Persistent Memory Graph | 3-4 | MemoryNode, memory_collector, CLAUDE.md 연동 |
| P3-4 | Autonomous Maintenance | 2-3 | MaintenanceJob, 템플릿 3개, CLI |
| P3-5 | Pack Marketplace | 2-3 | PackManifest, pack_manager, CLI install/remove |

**총 예상**: 17-23 커밋

---

## 2. Phase 1 상세 태스크

### P1-0: 신뢰성 기반 다지기

#### P1-0-A: Go 잔존 버그 수정

**변경 파일 목록:**
- `go/internal/ipc/server.go` — M-3 (이중 RecordHookSessionSeen) 수정, M-5 (요청 크기 제한 1MB) 추가
- `go/cmd/hamd/main.go` — M-2 (pollRuntimeState 에러 로깅), M-4 (err 변수 섀도잉) 수정
- `go/cmd/ham/pty.go` — H-2 (SIGWINCH 고루틴 릭: `signal.Stop` 후 `close(sigwinch)`)

**변경 내용:**
- `server.go`: `dispatch()` 함수의 `hook.session-start` 케이스에서 `RecordHookSessionSeen` 이중 호출 제거. `handleConnection()`에 `io.LimitReader(conn, 1<<20)` 적용
- `main.go`: `pollRuntimeState()` 반환 에러를 `log.Printf`로 기록. `err` 섀도잉 변수명 수정
- `pty.go`: SIGWINCH 시그널 핸들러에 `defer signal.Stop(sigwinch)` 추가

**테스트 계획:**
- `go/internal/ipc/server_test.go` — 1MB 초과 요청 시 에러 응답 테스트
- `go/internal/runtime/registry_test.go` — 동시 hook 이벤트 레이스 테스트 (goroutine 100개)

**빌드/테스트 검증:**
```bash
go test ./... -race -count=1
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- `go test ./... -race` PASS, 데이터 레이스 0건
- M-2, M-3, M-4, M-5, H-2 버그 모두 해결 확인

---

#### P1-0-B: Swift 버그 수정 + IPC enum 동기화

**변경 파일 목록:**
- `Sources/HamCore/DaemonIPC.swift` — DaemonCommand에 6개 케이스 추가 + `unknown` fallback
- `Sources/HamAppServices/DaemonClient.swift` — H-1 수정: write 후 `shutdown(fd, SHUT_WR)`
- `Sources/HamNotifications/UserNotificationSink.swift` — H-6 수정: `[weak self]` + Task 핸들, H-7 수정: `interactionHandler`에 NSLock
- `Sources/HamAppServices/MenuBarViewModel.swift` — H-8 수정: deinit 본문 제거, M-6 수정: heartbeat 중복 방지

**변경 내용:**
- `DaemonIPC.swift`: `DaemonCommand` enum에 `registerManaged`, `managedStop`, `managedExited`, `agentsRename`, `agentsOpenTarget`, `tmuxSessions` 추가. `init(from:)` 디코더에 unknown fallback 케이스 추가
- `DaemonClient.swift`: `UnixSocketDaemonTransport.send()` 내 write 완료 후 `Darwin.shutdown(fd, Int32(SHUT_WR))` 호출
- `UserNotificationSink.swift`: notification Task에 `[weak self]` 캡처. `interactionHandler` 프로퍼티에 NSLock 보호 추가
- `MenuBarViewModel.swift`: `deinit` 본문을 `stop()` 메서드로 이동. heartbeat 이벤트 발송 전 중복 체크 분기 추가

**테스트 계획:**
- `Tests/HamCoreTests/DaemonIPCTests.swift` — unknown command 디코딩 테스트
- `Tests/HamAppServicesTests/DaemonClientTests.swift` — shutdown(SHUT_WR) 호출 확인 mock 테스트

**빌드/테스트 검증:**
```bash
swift build --disable-sandbox
swift test --disable-sandbox
```

**완료 조건:**
- `swift build --disable-sandbox` 컴파일 성공
- `swift test --disable-sandbox` PASS
- DaemonCommand가 Go IPC Command 상수와 동기화 (hook 계열 제외)

---

#### P1-0-C: Hook Contract Test + 이벤트 스토어 테스트

**변경 파일 목록:**
- `go/internal/ipc/contract_test.go` — **[신규]** hook 커맨드 직렬화/역직렬화 왕복 테스트
- `go/internal/store/events_test.go` — truncate 경계 조건 테스트
- `go/internal/runtime/managed_state_test.go` — RecordHook* 메서드별 상태 전이 검증

**변경 내용:**
- `contract_test.go`: 27개 hook 커맨드 각각에 대해 `Request` 직렬화 → JSON → 역직렬화 왕복 테스트. 모든 필수 필드가 보존되는지 검증
- `events_test.go`: 10,001번째 Append 시 truncate 발동 확인, truncate 후 최근 10,000건만 남는지 검증
- `managed_state_test.go`: table-driven 테스트로 각 RecordHook* → 기대 Agent.Status 매핑 검증

**테스트 계획:**
- 위 파일 자체가 테스트 코드

**빌드/테스트 검증:**
```bash
go test ./go/internal/ipc/ -run TestContract -v
go test ./go/internal/store/ -run TestTruncate -v
go test ./go/internal/runtime/ -run TestRecordHook -v
```

**완료 조건:**
- 27개 hook contract test 전체 PASS
- truncate 경계 테스트 PASS
- RecordHook* 상태 전이 테스트 PASS (최소 10개 hook 커버)

---

### P1-1: 이벤트 스키마 확장 + Artifact Capture

#### P1-1-A: Event 구조체 확장 (Go + Swift)

**변경 파일 목록:**
- `go/internal/core/agent.go` — Event 구조체에 10개 필드 추가
- `Sources/HamCore/DaemonPayloads.swift` — AgentEventPayload에 대응 옵셔널 필드 추가

**변경 내용:**
- `agent.go`: Event 구조체에 다음 필드 추가 (모두 `omitempty`):
  - `SessionID string` — 세션 컨텍스트
  - `ParentAgentID string` — 부모 에이전트 ID
  - `TaskName string`, `TaskDesc string` — 태스크 컨텍스트
  - `ArtifactType string`, `ArtifactRef string`, `ArtifactData string` — Artifact 캡처
  - `ToolName string`, `ToolInput string`, `ToolType string`, `ToolDuration int` (json: `tool_duration_ms`) — 도구 컨텍스트
- `DaemonPayloads.swift`: `AgentEventPayload`에 `sessionID: String?`, `parentAgentID: String?`, `toolName: String?`, `toolDurationMs: Int?` 등 옵셔널 필드 추가. CodingKeys에 snake_case 매핑

**테스트 계획:**
- `go/internal/core/agent_test.go` — Event JSON 직렬화/역직렬화 테스트 (새 필드 포함/비포함 양방향)
- `Tests/HamCoreTests/AgentEventPayloadTests.swift` — 새 필드 있는 JSON / 없는 JSON 디코딩 테스트

**빌드/테스트 검증:**
```bash
go test ./go/internal/core/ -v
swift test --disable-sandbox --filter HamCoreTests
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- 기존 events.jsonl 파일이 새 코드로 정상 로딩 (새 필드 zero value)
- Go/Swift 양쪽 빌드 성공
- JSON 왕복 테스트 PASS

---

#### P1-1-B: Hook Handler에서 새 필드 채우기

**변경 파일 목록:**
- `go/internal/runtime/managed_state.go` — RecordHookToolStart, RecordHookToolDone, RecordHookTaskCreated, RecordHookAgentSpawned 수정
- `go/internal/ipc/server.go` — prepareHookRequest에서 SessionID를 이벤트에 전달

**변경 내용:**
- `managed_state.go`:
  - `RecordHookToolStart`: 이벤트에 `ToolName`, `ToolInput` (4KB truncate), `ToolType` (ClassifyToolType 결과) 설정
  - `RecordHookToolDone`: 이벤트에 `ToolDuration` (ToolStart ~ ToolDone 시간차 ms) 계산하여 설정
  - `RecordHookTaskCreated`: 이벤트에 `TaskName`, `TaskDesc` 설정
  - `RecordHookAgentSpawned`: 이벤트에 `ParentAgentID` (호출 컨텍스트의 Agent.ID) 설정
- `server.go`: `prepareHookRequest`에서 `request.SessionID`를 후속 Event 생성에 전달하도록 파이프라인 수정

**테스트 계획:**
- `go/internal/runtime/managed_state_test.go` — RecordHookToolStart 후 이벤트의 ToolName/ToolInput 검증, RecordHookToolDone 후 ToolDuration > 0 검증, RecordHookAgentSpawned 후 ParentAgentID != "" 검증

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestRecordHook -v
go test ./... -race
```

**완료 조건:**
- hook 이벤트에 ToolName, SessionID, ParentAgentID가 정상 기록됨
- 기존 테스트 회귀 없음

---

#### P1-1-C: ArtifactStore 신규 구현

**변경 파일 목록:**
- `go/internal/store/artifacts.go` — **[신규]** ArtifactStore 인터페이스 + FileArtifactStore
- `go/internal/store/events.go` — Append 시 ArtifactData 크기 판단 로직 추가

**변경 내용:**
- `artifacts.go`: `ArtifactStore` 인터페이스 (`Save`, `Load`, `Prune`), `FileArtifactStore` 구현체. 저장 경로: `~/Library/Application Support/ham-agents/artifacts/{agent_id}/{event_id}.json`. 4KB 이하 인라인, 4KB~1MB 파일 저장, 1MB 초과 truncate
- `events.go`: `Append()` 함수에서 `event.ArtifactData` 길이가 4KB 초과이면 `ArtifactStore.Save()`로 위임, `event.ArtifactRef`에 파일 경로 설정, `event.ArtifactData` 비우기

**테스트 계획:**
- `go/internal/store/artifacts_test.go` — **[신규]** 인라인 저장 (< 4KB), 파일 저장 (4KB-1MB), truncate (> 1MB), Prune (LRU 정리) 테스트

**빌드/테스트 검증:**
```bash
go test ./go/internal/store/ -run TestArtifact -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- Artifact 4KB/1MB 경계 테스트 PASS
- Prune 테스트 PASS
- hamd 빌드 성공

---

### P1-2: 실시간 Session Graph

#### P1-2-A: SessionGraph 데이터 모델 + Go 구현

**변경 파일 목록:**
- `go/internal/core/graph.go` — **[신규]** SessionNode, SessionGraph 타입 + BuildSessionGraph 함수
- `go/internal/ipc/ipc.go` — Request에 `Graph bool` 필드 추가
- `go/internal/ipc/server.go` — status 핸들러에서 graph 옵션 처리

**변경 내용:**
- `graph.go`: `SessionNode` (Agent, Children, BlockReason, Depth), `SessionGraph` (Roots, TotalCount, BlockedCount, GeneratedAt). `BuildSessionGraph(snapshot RuntimeSnapshot) SessionGraph` 함수: agents 순회 → SubAgents.AgentID로 자식 매칭 → root 식별 → BlockReason 설정
- `ipc.go`: `Request` 구조체에 `Graph bool` 필드 추가
- `server.go`: `CommandStatus` 핸들러에서 `req.Graph == true`이면 `BuildSessionGraph` 호출하여 Response에 포함

**테스트 계획:**
- `go/internal/core/graph_test.go` — **[신규]** 1 root + 2 children, orphan 에이전트, blocking reason 설정, 빈 snapshot 케이스

**빌드/테스트 검증:**
```bash
go test ./go/internal/core/ -run TestBuildSessionGraph -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- 3+ 에이전트 시나리오에서 트리 구조 정확히 구성
- root/child/orphan 정확 분류

---

#### P1-2-B: CLI --graph 플래그 + Swift 트리 렌더링

**변경 파일 목록:**
- `go/cmd/ham/parse.go` — `ham status --graph` 플래그 추가, 트리 ASCII 출력
- `Sources/HamCore/Agent.swift` — SessionNode, SessionGraph Codable 구조체
- `Sources/HamAppServices/MenuBarViewModel.swift` — agents 배열을 트리로 변환
- `apps/macos/HamMenuBarApp/Sources/MenuBarViews.swift` — 에이전트 리스트를 들여쓰기 트리 렌더링

**변경 내용:**
- `parse.go`: `status` 서브커맨드에 `--graph` 플래그 추가. `SessionGraph` 수신 후 `+- agent_name [status]` 형식으로 트리 ASCII 출력
- `Agent.swift`: `SessionNode` (agent, children, blockReason, depth), `SessionGraph` (roots, totalCount, blockedCount, generatedAt) Codable 구조체
- `MenuBarViewModel.swift`: `agents` 배열을 `SubAgents` 필드 기반으로 트리 구조로 변환하는 `buildAgentTree()` 메서드
- `MenuBarViews.swift`: `AgentListCard`를 `DisclosureGroup` 또는 indent 기반 트리로 렌더링. depth별 좌측 패딩 적용

**테스트 계획:**
- `go/cmd/ham/parse_test.go` — `--graph` 출력 포맷 검증
- `Tests/HamCoreTests/SessionGraphTests.swift` — SessionGraph JSON 디코딩 테스트

**빌드/테스트 검증:**
```bash
go test ./go/cmd/ham/ -run TestStatusGraph -v
swift build --disable-sandbox
swift test --disable-sandbox
```

**완료 조건:**
- `ham status --graph` 실행 시 트리 ASCII 출력
- 메뉴바에서 parent-child 에이전트가 들여쓰기로 표시
- Go/Swift 양쪽 빌드 성공

---

### P1-3: Notification Inbox (읽기 전용)

#### P1-3-A: InboxItem 모델 + InboxManager (Go)

**변경 파일 목록:**
- `go/internal/core/inbox.go` — **[신규]** InboxItem, InboxItemType 타입
- `go/internal/runtime/inbox.go` — **[신규]** InboxManager: Event → InboxItem 변환, ring buffer 100개, Read/MarkRead
- `go/internal/ipc/ipc.go` — `CommandInboxList`, `CommandInboxMarkRead` 상수 + Request/Response 필드
- `go/internal/ipc/server.go` — inbox.list, inbox.mark-read 핸들러
- `go/cmd/hamd/main.go` — InboxManager 초기화, Registry에 콜백 등록

**변경 내용:**
- `inbox.go` (core): `InboxItem` 구조체 (ID, AgentID, AgentName, Type, Summary, ToolName, OccurredAt, Read, Actionable). `InboxItemType` 상수: permission_request, notification, task_complete, error, stop
- `inbox.go` (runtime): `InboxManager` — Event 수신 콜백에서 Type에 따라 InboxItem 생성. ring buffer (최근 100개). `~/Library/Application Support/ham-agents/inbox.json` 영속화. `List(typeFilter, unreadOnly)`, `MarkRead(id)` 메서드
- `ipc.go`: 2개 커맨드 상수 추가. Request에 `TypeFilter`, `UnreadOnly`, `InboxItemID` 필드 추가. Response에 `InboxItems`, `UnreadCount` 필드 추가
- `server.go`: `CommandInboxList` → `InboxManager.List()`, `CommandInboxMarkRead` → `InboxManager.MarkRead()` 핸들러

**테스트 계획:**
- `go/internal/runtime/inbox_test.go` — **[신규]** hook.permission-request → InboxItem 생성, ring buffer 101번째 삽입 시 oldest 삭제, MarkRead 후 UnreadCount 감소, JSON 영속화/복원

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestInbox -v
go test ./go/internal/ipc/ -run TestInboxCommand -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- 6종 hook → InboxItem 변환 정상 동작
- ring buffer 100개 제한 동작
- IPC inbox.list/inbox.mark-read 정상 응답

---

#### P1-3-B: CLI `ham inbox` + Swift 메뉴바 Inbox UI

**변경 파일 목록:**
- `go/cmd/ham/parse.go` — `ham inbox` 커맨드 추가 (--mark-read, --type 옵션)
- `Sources/HamCore/DaemonIPC.swift` — `inboxList`, `inboxMarkRead` 커맨드, InboxItem Codable
- `Sources/HamCore/DaemonPayloads.swift` — InboxItem 페이로드
- `Sources/HamAppServices/DaemonClient.swift` — `fetchInbox()`, `markInboxRead()` 메서드
- `Sources/HamAppServices/MenuBarViewModel.swift` — InboxViewModel 소유, unreadCount 노출
- `apps/macos/HamMenuBarApp/Sources/MenuBarViews.swift` — Inbox 섹션 (뱃지 + 리스트 + 클릭 → OpenTarget)

**변경 내용:**
- `parse.go`: `ham inbox` 서브커맨드. `--mark-read`로 전체 읽음 처리, `--type permission_request` 필터
- Swift IPC: `DaemonCommand`에 `inboxList`, `inboxMarkRead` 케이스 추가. `InboxItemPayload` Codable 구조체. `HamDaemonClient`에 `fetchInbox()` → `inboxList` IPC, `markInboxRead(id:)` → `inboxMarkRead` IPC 메서드
- `MenuBarViewModel.swift`: 5초 refresh 사이클에 `fetchInbox()` 포함. `@Published var unreadCount: Int`
- `MenuBarViews.swift`: 에이전트 리스트 상단에 Inbox 섹션 추가. unread 뱃지 (빨간 원 + 숫자). InboxItem 리스트 (아이콘 + 에이전트명 + 요약 + 시간). 항목 클릭 시 `openTarget`으로 터미널 이동

**테스트 계획:**
- `Tests/HamCoreTests/InboxPayloadTests.swift` — InboxItem JSON 디코딩
- `Tests/HamAppServicesTests/DaemonClientTests.swift` — fetchInbox 요청 형식 검증

**빌드/테스트 검증:**
```bash
go test ./go/cmd/ham/ -run TestInbox -v
swift build --disable-sandbox
swift test --disable-sandbox
```

**완료 조건:**
- `ham inbox` 실행 시 InboxItem 목록 출력
- 메뉴바에 unread 뱃지 표시
- 항목 클릭 시 터미널 이동 동작

---

### P1-4: 비용/토큰 텔레메트리 v1

#### P1-4-A: ADR-3 데이터 소스 조사 + 구현

**변경 파일 목록 (시나리오 A/B 공통):**
- `go/internal/core/cost.go` — **[신규]** CostRecord 구조체
- `go/internal/store/cost.go` — **[신규]** FileCostStore (JSONL)
- `go/internal/runtime/cost.go` — **[신규]** CostTracker: 세션 종료 시 데이터 수집
- `go/internal/ipc/ipc.go` — `CommandCostSummary` 상수
- `go/internal/ipc/server.go` — cost.summary 핸들러
- `go/cmd/ham/parse.go` — `ham cost` 커맨드

**변경 내용:**
- 먼저 `~/.claude/` 디렉토리에서 토큰/비용 관련 파일 조사 (시나리오 A 가능성 확인)
- 시나리오 A (토큰 획득 가능): `CostRecord` (AgentID, SessionID, ProjectPath, TokensIn, TokensOut, EstimatedUSD, RecordedAt)
- 시나리오 B (시간만 가능): `CostRecord`에서 TokensIn/Out 대신 DurationSeconds, EstimatedTokens (추정치) 사용
- 시나리오 C (아무것도 없음): P1-4를 Phase 2로 이관. ADR-3 문서화만 수행
- `cost.go` (runtime): CostTracker는 RecordHookSessionEnd 콜백에서 세션 duration 계산 + (가능하면) 토큰 수집 → CostRecord 생성 → FileCostStore 저장

**테스트 계획:**
- `go/internal/store/cost_test.go` — CostRecord JSONL 저장/로딩
- `go/internal/runtime/cost_test.go` — CostTracker 세션 종료 시 CostRecord 생성

**빌드/테스트 검증:**
```bash
go test ./go/internal/store/ -run TestCost -v
go test ./go/internal/runtime/ -run TestCost -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- ADR-3 조사 결과가 문서화됨
- (시나리오 A/B) `ham cost` 실행 시 세션별 비용 요약 출력
- (시나리오 C) P1-4 이관 결정 문서화

---

#### P1-4-B: Swift 비용 표시 (시나리오 A/B인 경우)

**변경 파일 목록:**
- `Sources/HamCore/DaemonIPC.swift` — `costSummary` 커맨드
- `Sources/HamCore/DaemonPayloads.swift` — CostSummaryPayload
- `Sources/HamAppServices/DaemonClient.swift` — `fetchCostSummary()` 메서드
- `Sources/HamAppServices/MenuBarViewModel.swift` — 오늘 총 비용 표시 (opt-in)
- `apps/macos/HamMenuBarApp/Sources/MenuBarViews.swift` — SummaryBadge에 비용 표시 추가

**빌드/테스트 검증:**
```bash
swift build --disable-sandbox
swift test --disable-sandbox
```

**완료 조건:**
- 메뉴바 SummaryBadge에 오늘 예상 비용 표시 (설정에서 활성화 시)

---

## 3. Phase 2 상세 태스크

### P2-0. Event Broadcast (P1-5 에서 이관)

> 이 태스크는 Ralph 라운드 2 에서 Phase 1 P1-5 에서 Phase 2 초입 P2-0 으로 이관되었다. 기획 원본은 docs/spec/mission-control.md P1-5 참조.

#### P1-5-A: EventBus 구현 + Registry 리팩토링

**변경 파일 목록:**
- `go/internal/runtime/eventbus.go` — **[신규]** EventBus (Publish, Subscribe, Unsubscribe)
- `go/internal/runtime/registry.go` — `recordEvent` → `EventBus.Publish` 교체. EventBus 필드 추가
- `go/internal/runtime/events.go` — FollowEvents를 EventBus subscription 기반으로 재작성
- `go/internal/store/events.go` — eventStore.Append를 EventBus subscriber로 등록
- `go/internal/runtime/inbox.go` — InboxManager를 EventBus subscriber로 전환
- `go/cmd/hamd/main.go` — EventBus 초기화, subscriber 등록

**변경 내용:**
- `eventbus.go`: `EventBus` 구조체 (sync.RWMutex + map[string]chan Event). `Publish(event Event)`: RLock → 모든 subscriber에 non-blocking send (채널 full이면 drop + warning). `Subscribe(id string) <-chan Event`: buffered channel (256). `Unsubscribe(id string)`: remove + close
- `registry.go`: `Registry` 구조체에 `eventBus *EventBus` 필드 추가. `appendEvent()` 내부의 `eventStore.Append()` 직접 호출을 `eventBus.Publish()` 호출로 교체
- `events.go`: `FollowEvents`가 내부적으로 `eventBus.Subscribe` → channel 읽기로 전환. 기존 파일 폴링 로직 제거
- `main.go`: `NewEventBus()` → `eventStore` subscriber 등록 → `inboxManager` subscriber 등록 → `NewRegistry(eventBus)` 순서

**테스트 계획:**
- `go/internal/runtime/eventbus_test.go` — **[신규]** Publish → 3 subscriber 동시 수신, subscriber full → drop + 에러 없음, Unsubscribe 후 수신 안함
- `go/internal/runtime/events_test.go` — FollowEvents가 EventBus 기반으로 동작하는지 통합 테스트

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestEventBus -v
go test ./... -race
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- EventBus를 통해 eventStore, inboxManager, followEvents 모두 정상 수신
- 기존 `ham events --follow` 동작 변경 없음
- `go test -race` PASS

---

### P2-1. Embedded PTY Runtime (신규 — Round 3)

**의존성**: P1-0 Registry 락, P1-1 SessionEvent 스키마, ADR-2 (tech-migration.md)

**목표**: hamd 가 PTY master 를 소유하고 Claude Code 를 spawn. NDJSON 스트림으로 Swift ham Studio 가 PTY 데이터를 수신 → SwiftTerm 렌더.

**변경 파일**:
- `go/internal/runtime/managed.go` — ManagedService.Start 에서 openPTY 호출, managedProcess 에 ptmx/subs 필드 추가
- `go/internal/runtime/pty_alloc.go` (신규) — 기존 go/cmd/ham/pty.go 의 openPTY 패턴을 재사용 가능하게 추출
- `go/internal/ipc/ipc.go` — Command 상수 3 개 추가 (`CommandFollowPTY`, `CommandWritePTY`, `CommandResizePTY`)
- `go/internal/ipc/server.go` — dispatch 에 3 개 case 추가, handleFollowPTY 는 CommandFollowEvents 패턴 재사용
- `go/internal/core/agent.go` — 기존 `SessionTTY` 필드를 managed 모드에서도 populate
- `Sources/HamCore/DaemonIPC.swift` — DaemonCommand enum 에 `ptyFollow`, `ptyWrite`, `ptyResize` 추가 (16 → 19)
- `Sources/HamApp/PTY/PtyClient.swift` (신규)
- `Sources/HamApp/PTY/PtyTabView.swift` (신규 — SwiftTerm NSViewRepresentable)
- `Sources/HamApp/StudioWindow.swift` (신규 또는 확장) — 탭 컨테이너

**테스트**:
- Go: `go test ./go/internal/runtime -run TestPTYHost` (ptmx 할당 + subs fan-out unit test)
- Go: `go test ./go/internal/ipc -run TestFollowPTY` (NDJSON stream contract)
- Swift: SwiftTerm 통합은 수동 smoke test (Studio 탭 열기 → Claude Code 세션 시작 → 입력/출력 rendering 확인)

**빌드**:
```bash
go build ./go/cmd/ham ./go/cmd/hamd
# Swift 측: xcodebuild 또는 swift build (HamApp 타겟에 SwiftTerm SPM dependency 추가)
```

**완료 조건**:
- [ ] hamd managed 모드가 PTY 할당
- [ ] Swift Studio 탭이 SwiftTerm 으로 PTY 출력 렌더
- [ ] 사용자 입력이 CommandWritePTY 경유로 ptmx 에 write
- [ ] TIOCSWINSZ 리사이즈 동작
- [ ] Studio 크래시 후 재시작 시 resume_from_seq 로 재구독 성공
- [ ] 기존 `ham run <provider>` CLI 로컬 PTY 경로도 계속 동작 (회귀 없음)
- [ ] `go test ./... -race -count=1` 통과
- [ ] `go build ./go/cmd/ham ./go/cmd/hamd` 성공

---

### P2-2: Session Launcher

#### P2-2-A: Studio New Session UI (신규 — Round 3)

**의존성**: P2-1 Embedded PTY Runtime

**목표**: ham Studio 에서 새 세션을 시작할 때 workspace/model/playbook 을 선택하는 UI.

**변경 파일 목록:**
- `apps/macos/HamMenuBarApp/Sources/SessionLauncherView.swift` — **[신규]** New Session 모달 (workspace picker, model selector, playbook dropdown)
- `Sources/HamAppServices/SessionLauncherModel.swift` — **[신규]** SessionLauncherModel (IPC CommandStartSession 호출)
- `Sources/HamCore/DaemonIPC.swift` — `startSession` 커맨드 추가

**빌드/테스트 검증:**
```bash
swift build --disable-sandbox
swift test --disable-sandbox
```

**완료 조건:**
- "New Session" 클릭 시 launcher 모달 열림
- workspace/model/playbook 선택 후 세션 시작 → P2-1 PTY 탭에 연결

---

### P2-3: Approval Interception

#### P2-3-A: PTY 블록 + CommandAnswerPermission (신규 — Round 3)

**의존성**: P2-1 Embedded PTY Runtime

**목표**: hamd 가 permission 요청 감지 시 PTY 입력을 블록하고 Swift 승인 모달을 트리거. 사용자 응답을 `CommandAnswerPermission` 으로 전달.

**변경 파일 목록:**
- `go/internal/runtime/managed.go` — PTY 출력에서 permission 패턴 감지, 블록 상태 설정
- `go/internal/ipc/ipc.go` — `CommandAnswerPermission` 상수 추가
- `go/internal/ipc/server.go` — `handleAnswerPermission` 핸들러 추가
- `Sources/HamCore/DaemonIPC.swift` — `answerPermission` 커맨드 추가
- `Sources/HamApp/PTY/ApprovalModalView.swift` — **[신규]** 승인 모달 (Yes/No/Always/Never)

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestApproval -v
go build ./go/cmd/ham ./go/cmd/hamd
swift build --disable-sandbox
```

**완료 조건:**
- [ ] permission 요청 감지 시 PTY 입력 블록
- [ ] Swift 승인 모달 표시
- [ ] 사용자 응답이 CommandAnswerPermission 경유로 hamd 에 전달
- [ ] "Always" 선택 시 동일 패턴 자동 승인

---

### P2-4: Agent Team Orchestrator

> 기존 round 2 P2-2 범위. Round 3 에서 P2-4 로 재번호 부여.

#### P2-4-A: TeamOrchestratorState + Git Adapter

**변경 파일 목록:**
- `go/internal/core/team.go` — TeamOrchestratorState, TaskContract, WorktreeInfo, MergeGate 타입 추가
- `go/internal/runtime/orchestrator.go` — **[신규]** Orchestrator (팀 상태 관리, concurrency budget)
- `go/internal/adapters/git.go` — **[신규]** WorktreeScanner (`git worktree list` 파싱, conflict 탐지)
- `go/internal/ipc/ipc.go` — CommandTeamOrchestrate, CommandTeamTaskList, CommandTeamMergeGate
- `go/internal/ipc/server.go` — 핸들러 3개

**테스트 계획:**
- `go/internal/adapters/git_test.go` — `git worktree list` 출력 파싱 테스트
- `go/internal/runtime/orchestrator_test.go` — concurrency budget 초과 시 거부, task contract 상태 전이

**빌드/테스트 검증:**
```bash
go test ./go/internal/adapters/ -run TestGit -v
go test ./go/internal/runtime/ -run TestOrchestrator -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- worktree 목록 스캔 동작
- concurrency budget 제한 동작
- IPC 3개 커맨드 응답 정상

---

#### P2-4-B: Studio Team UI

**변경 파일 목록:**
- `apps/macos/HamMenuBarApp/Sources/StudioSidebarView.swift` — 팀 트리 섹션 (lead/worker 아이콘, task progress bar)
- `apps/macos/HamMenuBarApp/Sources/StudioInspectorView.swift` — 팀 상세 섹션 (worktree, merge gate, concurrency)
- `Sources/HamAppServices/TeamOrchestratorModel.swift` — **[신규]** TeamOrchestratorModel
- `Sources/HamCore/DaemonIPC.swift` — 3개 커맨드 + payload 타입

**빌드/테스트 검증:**
```bash
swift build --disable-sandbox
swift test --disable-sandbox
```

**완료 조건:**
- Studio 사이드바에서 팀 트리 구조 표시
- 인스펙터에서 worktree 상태 + merge gate 표시

---

### P2-5: Playbooks / Recipes

> 기존 round 2 P2-3 범위. Round 3 에서 P2-5 로 재번호 부여.

#### P2-5-A: Playbook 스키마 + Runner (Go)

**변경 파일 목록:**
- `go/internal/core/playbook.go` — **[신규]** Playbook, PlaybookStep, PlaybookExecution 타입
- `go/internal/runtime/playbook_runner.go` — **[신규]** PlaybookRunner (step 순회, approval gate)
- `go/internal/store/playbook_store.go` — **[신규]** PlaybookStore (YAML 로드, 검색 경로: `.ham/playbooks/` + `~/.ham/playbooks/`)
- `go/internal/ipc/ipc.go` — CommandPlaybookList, CommandPlaybookRun, CommandPlaybookStatus
- `go/internal/ipc/server.go` — 핸들러 3개
- `go/cmd/ham/parse.go` — `ham playbook list/run/status` 커맨드

**테스트 계획:**
- `go/internal/store/playbook_store_test.go` — YAML 파싱, 검색 경로 탐색
- `go/internal/runtime/playbook_runner_test.go` — 3-step playbook 실행, approval gate에서 정지

**빌드/테스트 검증:**
```bash
go test ./go/internal/store/ -run TestPlaybook -v
go test ./go/internal/runtime/ -run TestPlaybook -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- `ham playbook list` 로 `.ham/playbooks/*.yaml` 목록 출력
- `ham playbook run <name>` 으로 managed agent 시작 + step 순차 실행

---

#### P2-5-B: Studio Playbook UI

**변경 파일 목록:**
- `apps/macos/HamMenuBarApp/Sources/StudioPlaybookView.swift` — **[신규]** playbook 목록 + 실행 UI
- `apps/macos/HamMenuBarApp/Sources/PlaybookExecutionView.swift` — **[신규]** 단계별 진행률
- `Sources/HamAppServices/PlaybookModel.swift` — **[신규]** PlaybookModel
- `Sources/HamCore/DaemonIPC.swift` — 3개 커맨드 + payload

**빌드/테스트 검증:**
```bash
swift build --disable-sandbox
swift test --disable-sandbox
```

**완료 조건:**
- Studio에서 playbook 목록 표시
- playbook 실행 시 단계별 진행률 표시

---

### P2-6: Git/CI/Issue 연동

> 기존 round 2 P2-4 범위. Round 3 에서 P2-6 으로 재번호 부여.

#### P2-6-A: GitHub Webhook + EventTrigger (Go)

**변경 파일 목록:**
- `go/internal/core/external_event.go` — **[신규]** ExternalEvent, ExternalEventSource, EventTriggerRule
- `go/internal/adapters/github_webhook.go` — **[신규]** GitHubWebhookHandler
- `go/internal/runtime/webhook_server.go` — **[신규]** WebhookServer (localhost HTTP)
- `go/internal/runtime/event_trigger.go` — **[신규]** EventTriggerEngine
- `go/internal/ipc/ipc.go` — CommandExternalEvents, CommandTriggerRuleList, CommandTriggerRuleCreate

**테스트 계획:**
- `go/internal/adapters/github_webhook_test.go` — PR/CI webhook payload 파싱
- `go/internal/runtime/event_trigger_test.go` — 규칙 매칭 + playbook 트리거

**빌드/테스트 검증:**
```bash
go test ./go/internal/adapters/ -run TestGitHub -v
go test ./go/internal/runtime/ -run TestEventTrigger -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- GitHub webhook 수신 + ExternalEvent 변환
- EventTriggerRule 매칭 → playbook 실행 트리거

---

### P2-7: Review Loop

> 기존 round 2 P2-5 범위. Round 3 에서 P2-7 로 재번호 부여.

#### P2-7-A: Checkpoint + Review Queue (Go)

**변경 파일 목록:**
- `go/internal/core/review.go` — **[신규]** Checkpoint, ReviewItem, ReviewStatus
- `go/internal/runtime/checkpoint_manager.go` — **[신규]** agent done 이벤트 시 git checkpoint 생성
- `go/internal/runtime/review_queue.go` — **[신규]** ReviewQueue (review item CRUD, status 전환)
- `go/internal/adapters/git.go` — `CreateCheckpoint()`, `RewindToCheckpoint()` 메서드 추가
- `go/internal/ipc/ipc.go` — CommandReviewList, CommandReviewApprove, CommandCheckpointRewind

**테스트 계획:**
- `go/internal/runtime/checkpoint_manager_test.go` — done 이벤트 → checkpoint 생성
- `go/internal/runtime/review_queue_test.go` — status 전이 (pending → approved → merged)

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestCheckpoint -v
go test ./go/internal/runtime/ -run TestReviewQueue -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- agent done → 자동 checkpoint 생성
- review approve → ready-for-merge 전환

---

## 4. Phase 3 상세 태스크

### P3-0: Embedded DB 전환

#### P3-0-A: SQLite 도입 + 기존 데이터 마이그레이션

**변경 파일 목록:**
- `go.mod` — `modernc.org/sqlite` 의존성 추가
- `go/internal/store/db.go` — **[신규]** DB 초기화, 스키마 마이그레이션
- `go/internal/store/db_event_store.go` — **[신규]** SQLite 기반 EventStore
- `go/internal/store/db_agent_store.go` — **[신규]** SQLite 기반 AgentStore
- `go/internal/store/migration.go` — **[신규]** JSONL → SQLite 데이터 마이그레이션

**테스트 계획:**
- `go/internal/store/db_test.go` — CRUD, 시간 범위 쿼리, 세션별 필터링
- `go/internal/store/migration_test.go` — JSONL → SQLite 마이그레이션 정합성

**빌드/테스트 검증:**
```bash
go test ./go/internal/store/ -run TestDB -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- SQLite DB 파일 `~/Library/Application Support/ham-agents/ham.db` 생성
- 기존 events.jsonl → SQLite 마이그레이션 성공
- 세션별/시간범위 쿼리 동작

---

### P3-1: AI Agent Debugger

#### P3-1-A: SessionTrace + TraceBuilder (Go)

**변경 파일 목록:**
- `go/internal/core/debugger.go` — **[신규]** SessionTrace, ToolCallNode, BreakpointRule, SessionDiff
- `go/internal/runtime/trace_builder.go` — **[신규]** 이벤트 로그 → SessionTrace 변환
- `go/internal/store/trace_store.go` — **[신규]** SessionTrace 영속화 (SQLite)
- `go/internal/ipc/ipc.go` — debug.trace, debug.trace.list, debug.compare, debug.breakpoint.* 커맨드
- `go/cmd/ham/parse.go` — `ham debug replay/compare/breakpoint` CLI

**테스트 계획:**
- `go/internal/runtime/trace_builder_test.go` — 10개 이벤트 → SessionTrace 구성, tool-call 체인 정확성

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestTraceBuilder -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- `ham debug replay <session-id>` 실행 시 타임라인 출력
- tool-call 체인이 트리 구조로 표시

---

#### P3-1-B: Studio Debugger UI (Swift)

**변경 파일 목록:**
- `Sources/HamCore/DebuggerPayloads.swift` — **[신규]** SessionTrace, ToolCallNode 디코딩
- `Sources/HamAppServices/DebuggerViewModel.swift` — **[신규]** 리플레이 ViewModel
- `apps/macos/HamMenuBarApp/Sources/DebuggerViews.swift` — **[신규]** Studio 디버거 탭

**빌드/테스트 검증:**
```bash
swift build --disable-sandbox
swift test --disable-sandbox
```

**완료 조건:**
- Studio에서 세션 트레이스 타임라인 표시
- step-through로 tool-call 탐색 가능

---

### P3-2: Org Policy Engine

#### P3-2-A: Policy 로딩 + 평가 엔진 (Go)

**변경 파일 목록:**
- `go/internal/core/policy.go` — **[신규]** PolicySet, PolicyRule, PolicyViolation, PolicyScope
- `go/internal/runtime/policy_engine.go` — **[신규]** EventBus subscriber, 규칙 평가, 위반 감지
- `go/internal/runtime/policy_loader.go` — **[신규]** YAML 로딩, 상속 병합 (org → team → repo)
- `go/internal/store/policy_store.go` — **[신규]** PolicyViolation 영속화
- `go/cmd/ham/parse.go` — `ham policy list/check/violations` CLI

**테스트 계획:**
- `go/internal/runtime/policy_engine_test.go` — tool_name 매칭, input 패턴 매칭, 위반 감지 → InboxItem

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestPolicy -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- `.ham/policies/*.yaml` 로딩 동작
- tool_name + input 패턴 매칭으로 위반 감지
- `ham policy violations` 출력

---

### P3-3: Persistent Memory Graph

#### P3-3-A: Memory 모델 + Collector (Go)

**변경 파일 목록:**
- `go/internal/core/memory.go` — **[신규]** MemoryNode, MemoryEdge, MemoryScope
- `go/internal/runtime/memory_collector.go` — **[신규]** EventBus subscriber, compact summary → 메모리 후보
- `go/internal/runtime/memory_graph.go` — **[신규]** 그래프 구성, 검색
- `go/internal/store/memory_store.go` — **[신규]** MemoryNode 영속화 (SQLite FTS5)
- `go/cmd/ham/parse.go` — `ham memory list/add/search/promote` CLI

**테스트 계획:**
- `go/internal/runtime/memory_collector_test.go` — 세션 종료 → 메모리 후보 생성
- `go/internal/store/memory_store_test.go` — FTS5 검색 테스트

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestMemory -v
go test ./go/internal/store/ -run TestMemory -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- `ham memory add "tip"` → 저장
- `ham memory search "keyword"` → FTS5 검색 결과

---

### P3-4: Autonomous Maintenance

#### P3-4-A: MaintenanceJob + 기본 템플릿 (Go)

**변경 파일 목록:**
- `go/internal/core/maintenance.go` — **[신규]** MaintenanceJob, MaintenanceRun, MaintenanceSchedule
- `go/internal/runtime/maintenance_runner.go` — **[신규]** 작업 실행 관리 (Playbook 기반)
- `go/internal/store/maintenance_store.go` — **[신규]** 실행 이력 저장
- `go/cmd/ham/parse.go` — `ham maintain list/run/history` CLI

**테스트 계획:**
- `go/internal/runtime/maintenance_runner_test.go` — 작업 실행 + 이력 저장

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestMaintenance -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- `ham maintain run dependency-sweep` 실행 시 managed agent 시작
- 실행 이력 저장 + `ham maintain history` 조회

---

### P3-5: Pack Marketplace

#### P3-5-A: PackManifest + Manager (Go)

**변경 파일 목록:**
- `go/internal/core/pack.go` — **[신규]** PackManifest, PackRegistry, PackSource
- `go/internal/runtime/pack_manager.go` — **[신규]** Pack 설치/제거/목록
- `go/internal/store/pack_store.go` — **[신규]** PackRegistry 영속화 (`~/.ham/packs/registry.json`)
- `go/cmd/ham/parse.go` — `ham pack install/remove/list/create` CLI

**테스트 계획:**
- `go/internal/runtime/pack_manager_test.go` — 로컬 디렉토리에서 pack 설치 → 제거

**빌드/테스트 검증:**
```bash
go test ./go/internal/runtime/ -run TestPack -v
go build ./go/cmd/ham ./go/cmd/hamd
```

**완료 조건:**
- `ham pack install ./my-pack` → playbook/policy 파일 로딩
- `ham pack list` → 설치된 pack 목록 출력

---

## 5. 에이전트 팀 구성 가이드

### Phase 1 팀 구성

#### 참여 에이전트

| 에이전트 | 모델 | 역할 | 담당 태스크 |
|----------|------|------|-------------|
| **go-backend** | opus | Go 코드 변경 | P1-0-A, P1-1-A(Go), P1-1-B, P1-1-C, P1-2-A, P1-3-A, P1-4-A |
| **swift-frontend** | opus | Swift 코드 변경 | P1-0-B, P1-1-A(Swift), P1-2-B(Swift), P1-3-B(Swift), P1-4-B |
| **test-engineer** | opus | 테스트 작성 | P1-0-C, 각 태스크별 테스트 코드 |
| **architect** | opus | 스키마 리뷰 (읽기 전용) | P1-1 Event 스키마 리뷰 |
| **code-reviewer** | opus | 품질 검증 (읽기 전용) | 각 태스크 완료 후 Go-Swift 동기화 확인 |
| **devops** | haiku | 빌드 검증 | 각 커밋 후 `go build` + `swift build` 실행 |

#### 태스크 분배

**순차 실행 (의존성 있음):**
```
P1-0-A (go-backend) → P1-0-B (swift-frontend) → P1-0-C (test-engineer)
    ↓
P1-1-A (go-backend + swift-frontend 병렬) → P1-1-B (go-backend) → P1-1-C (go-backend)
    ↓
P1-2-A (go-backend) ─┬─ P1-2-B (swift-frontend)     ← 병렬 가능
P1-3-A (go-backend) ─┘─ P1-3-B (swift-frontend)     ← 병렬 가능
    ↓
P1-4-A (go-backend) → P1-4-B (swift-frontend)
```

**병렬 가능 구간:**
- P1-0-A (Go 버그) + P1-0-B (Swift 버그): 독립적이므로 병렬
- P1-2 (Session Graph) + P1-3 (Inbox): P1-1 완료 후 병렬
- P1-1-A의 Go 파트 + Swift 파트: 스키마 확정 후 병렬

#### Coordination 포인트

1. **P1-0 완료 후**: architect가 Go-Swift enum 동기화 리뷰. code-reviewer가 버그 수정 검증
2. **P1-1 스키마 확정**: architect가 Event 스키마 리뷰 → go-backend, swift-frontend 동시 구현 시작
3. **P1-3 InboxItem 스키마**: go-backend가 InboxItem 정의 → swift-frontend가 Codable 구현

---

### Phase 2 팀 구성

#### 참여 에이전트

| 에이전트 | 역할 | 담당 태스크 |
|----------|------|-------------|
| **go-backend** | Go 코드 변경 | P2-0-A (EventBus), P2-1-A (ManagedService PTY + IPC commands), P2-3-A (Permission interception + CommandAnswerPermission), P2-4-A, P2-5-A, P2-6-A, P2-7-A |
| **swift-frontend** | Swift UI 구현 | P2-1-B (PTY UI + SwiftTerm NSViewRepresentable), P2-2-A (Session Launcher UI), P2-3-B (Approval modal), P2-4-B, P2-5-B |
| **ui-designer** | Studio 레이아웃 설계 | P2-4-B 팀 UI, P2-5-B playbook UI |
| **test-engineer** | 테스트 | 각 태스크별 테스트 |
| **architect** | 설계 리뷰 | P2-1 설계 검증 (ADR-2 구현 확인), P2-5 Playbook 스키마, P2-6 webhook 보안 |
| **code-reviewer** | 품질 검증 | Studio 코드 리뷰 |
| **devops** | 빌드 검증 | 각 커밋 후 빌드, SwiftTerm SPM dependency spike |

#### 태스크 분배

**순차 실행:**
```
P2-0-A (go-backend)  ← EventBus 구현 (P1-5 이관)
    ↓
[SwiftTerm SPM spike — devops + architect]
    ↓
P2-1-A (go-backend: PTY 할당 + IPC) ∥ P2-1-B (swift-frontend: PtyClient + PtyTabView)
    ↓
P2-2-A (swift-frontend: Session Launcher) ∥ P2-3-A (go-backend: Permission interception)
P2-3-B (swift-frontend: Approval modal)
    ↓
P2-4-A (go-backend) + P2-4-B (swift-frontend)  ← 스키마 확정 후 병렬
P2-5-A (go-backend) + P2-5-B (swift-frontend)  ← 스키마 확정 후 병렬
    ↓
P2-6-A (go-backend) → P2-7-A (go-backend)
```

---

### Phase 3 팀 구성

#### 참여 에이전트

| 에이전트 | 역할 | 담당 태스크 |
|----------|------|-------------|
| **go-backend** | Go 코드 변경 | P3-0-A, P3-1-A, P3-2-A, P3-3-A, P3-4-A, P3-5-A |
| **swift-frontend** | Swift UI | P3-1-B, Studio 디버거/정책/메모리 뷰 |
| **architect** | DB 스키마 설계 | P3-0 SQLite 스키마, P3-2 정책 DSL |
| **test-engineer** | 테스트 | 각 태스크별 |
| **code-reviewer** | 품질 검증 | SQLite 마이그레이션 정합성 |
| **devops** | 빌드 + 의존성 관리 | `modernc.org/sqlite` 의존성 검증 |

---

## 6. 실행 프롬프트

### Phase 1 실행 프롬프트

```
docs/spec/implementation-plan.md의 Phase 1 (Mission Control MVP) 태스크를 구현해라.

## 브랜치
dev/phase-1 브랜치에서 작업. main에 직접 커밋 금지.

## 에이전트 팀 구성
.claude/agents/ 에이전트 정의 참고:
- go-backend (opus): Go 코드 변경 전담
- swift-frontend (opus): Swift 코드 변경 전담
- test-engineer (opus): 테스트 코드 전담
- architect (opus): 스키마 리뷰 (읽기 전용)
- code-reviewer (opus): 완료 후 품질 검증 (읽기 전용)
- devops (haiku): 빌드 검증

## 실행 순서

### Step 1: P1-0 신뢰성 기반 (4-5 커밋)
1. go-backend: go/internal/ipc/server.go M-3, M-5 수정. go/cmd/hamd/main.go M-2, M-4 수정. go/cmd/ham/pty.go H-2 수정
2. swift-frontend: Sources/HamCore/DaemonIPC.swift enum 동기화 (6개 추가 + unknown). DaemonClient.swift H-1. UserNotificationSink.swift H-6, H-7. MenuBarViewModel.swift H-8, M-6
3. test-engineer: go/internal/ipc/contract_test.go (27 hook 왕복). go/internal/store/events_test.go (truncate). go/internal/runtime/managed_state_test.go (상태 전이)

### Step 2: P1-1 이벤트 스키마 확장 (3-4 커밋)
1. architect: Event 스키마 10개 신규 필드 리뷰 (docs/spec/mission-control.md 섹션 6 참조)
2. go-backend + swift-frontend 병렬: core/agent.go Event 확장 + DaemonPayloads.swift 대응
3. go-backend: managed_state.go hook handler에서 새 필드 채우기 + server.go SessionID 파이프라인
4. go-backend: store/artifacts.go ArtifactStore 신규

### Step 3: P1-2 + P1-3 병렬 (5-7 커밋)
1. go-backend: core/graph.go SessionGraph + ipc 확장 (P1-2)
2. go-backend: core/inbox.go + runtime/inbox.go + ipc 확장 (P1-3)
3. swift-frontend: Agent.swift SessionGraph Codable + 메뉴바 트리 렌더링 (P1-2)
4. swift-frontend: DaemonIPC inbox 커맨드 + 메뉴바 Inbox UI (P1-3)

### Step 4: P1-4 비용 텔레메트리 (2-3 커밋)
1. go-backend: ~/.claude/ 디렉토리 조사 → 시나리오 결정
2. go-backend: core/cost.go + store/cost.go + runtime/cost.go + ipc + CLI
3. swift-frontend: DaemonIPC costSummary + 메뉴바 비용 표시 (시나리오 A/B)

## 빌드/테스트 검증 (매 커밋 후)
go test ./... -race -count=1
go build ./go/cmd/ham ./go/cmd/hamd
swift build --disable-sandbox
swift test --disable-sandbox

## 완료 조건
- [ ] go test ./... -race PASS (데이터 레이스 0건)
- [ ] swift test --disable-sandbox PASS
- [ ] go build ./go/cmd/ham ./go/cmd/hamd 성공
- [ ] swift build --disable-sandbox 성공
- [ ] DaemonCommand enum이 Go IPC Command와 동기화됨 (hook 계열 제외)
- [ ] 27개 hook contract test PASS
- [ ] Event 스키마에 SessionID, ToolName, ParentAgentID 필드 존재
- [ ] ham status --graph 실행 시 트리 출력
- [ ] ham inbox 실행 시 알림 목록 출력
```

---

### Phase 2 실행 프롬프트

```
docs/spec/implementation-plan.md의 Phase 2 (Terminal IDE - ham Studio) 태스크를 구현해라.

## 브랜치
dev/phase-2 브랜치에서 작업. dev/phase-1이 main에 머지된 후 시작.

## 에이전트 팀 구성
.claude/agents/ 에이전트 정의 참고:
- go-backend (opus): Go 코드 변경 전담
- swift-frontend (opus): Swift UI 구현 전담
- ui-designer (opus): Studio 레이아웃 + 픽셀 아트 일관성
- test-engineer (opus): 테스트 코드 전담
- architect (opus): ADR-2 구현 검증, Playbook 스키마 + Webhook 보안 리뷰
- code-reviewer (opus): Studio 코드 품질 검증
- devops (haiku): 빌드 검증

## 실행 순서

### Step 0: P2-0 EventBus (2-3 커밋)
1. go-backend: runtime/eventbus.go 신규, registry.go 리팩토링, events.go EventBus 기반 재작성

### Step 0a: SwiftTerm SPM dependency spike (P2-1 전 선행)
1. devops: Package.swift 에 SwiftTerm SPM dependency 추가 후 `swift build --disable-sandbox` 검증
2. architect: ADR-2 (tech-migration.md) 구현 가능성 확인

### Step 1: P2-1 Embedded PTY Runtime (6-8 커밋)
1. go-backend: runtime/pty_alloc.go 신규 (openPTY 추출), runtime/managed.go PTY 할당 통합
2. go-backend: ipc/ipc.go CommandFollowPTY/CommandWritePTY/CommandResizePTY 추가, ipc/server.go 핸들러
3. swift-frontend: DaemonIPC.swift ptyFollow/ptyWrite/ptyResize 추가
4. swift-frontend: PTY/PtyClient.swift 신규, PTY/PtyTabView.swift 신규 (SwiftTerm NSViewRepresentable)
5. swift-frontend: StudioWindow.swift 탭 컨테이너

### Step 1b: P2-2 Session Launcher (2-3 커밋)
1. swift-frontend: SessionLauncherView.swift 신규 + SessionLauncherModel.swift 신규
2. swift-frontend: DaemonIPC.swift startSession 커맨드 추가

### Step 1c: P2-3 Approval Interception (3-4 커밋)
1. go-backend: runtime/managed.go permission 패턴 감지 + 블록 로직
2. go-backend: ipc/ipc.go CommandAnswerPermission + ipc/server.go 핸들러
3. swift-frontend: DaemonIPC.swift answerPermission 추가 + PTY/ApprovalModalView.swift 신규

### Step 2: P2-4 Team Orchestrator (3-4 커밋)
1. go-backend: core/team.go 타입 확장 + runtime/orchestrator.go + adapters/git.go WorktreeScanner
2. swift-frontend + ui-designer: Studio sidebar 팀 트리 + inspector 팀 상세

### Step 3: P2-5 Playbooks (2-3 커밋)
1. architect: Playbook YAML 스키마 리뷰
2. go-backend: core/playbook.go + runtime/playbook_runner.go + store/playbook_store.go + CLI
3. swift-frontend: StudioPlaybookView + PlaybookExecutionView

### Step 4: P2-6 Git/CI 연동 (2-3 커밋)
1. architect: Webhook 보안 (localhost only, 인증) 리뷰
2. go-backend: core/external_event.go + adapters/github_webhook.go + runtime/webhook_server.go + event_trigger.go

### Step 5: P2-7 Review Loop (2-3 커밋)
1. go-backend: core/review.go + runtime/checkpoint_manager.go + review_queue.go + adapters/git.go
2. swift-frontend: StudioInspectorView review section + CheckpointTimelineView

## 빌드/테스트 검증 (매 커밋 후)
go test ./... -race -count=1
go build ./go/cmd/ham ./go/cmd/hamd
swift build --disable-sandbox
swift test --disable-sandbox

## 완료 조건
- [ ] go test ./... -race PASS
- [ ] swift test --disable-sandbox PASS
- [ ] hamd managed 모드가 PTY 할당 (go test ./go/internal/runtime -run TestPTYHost PASS)
- [ ] Swift Studio 탭이 SwiftTerm 으로 PTY 출력 렌더 (smoke test)
- [ ] CommandFollowPTY/CommandWritePTY/CommandResizePTY IPC 동작
- [ ] Session Launcher 모달에서 workspace/model/playbook 선택 후 세션 시작
- [ ] permission 요청 시 승인 모달 표시 + CommandAnswerPermission 전달
- [ ] 팀 트리에서 lead/worker 구분 표시
- [ ] ham playbook list/run 동작
- [ ] Studio에서 playbook 실행 진행률 표시
- [ ] GitHub webhook 수신 + ExternalEvent 변환 (localhost)
- [ ] agent done → 자동 checkpoint 생성
```

---

### Phase 3 실행 프롬프트

```
docs/spec/implementation-plan.md의 Phase 3 (AgentOps Platform) 태스크를 구현해라.

## 브랜치
dev/phase-3 브랜치에서 작업. dev/phase-2가 main에 머지된 후 시작.

## 에이전트 팀 구성
.claude/agents/ 에이전트 정의 참고:
- go-backend (opus): Go 코드 변경 전담 (SQLite, 디버거, 정책, 메모리, 유지보수, Pack)
- swift-frontend (opus): Swift UI 전담 (Studio 디버거/정책/메모리 탭)
- architect (opus): SQLite 스키마 설계, 정책 DSL 리뷰
- test-engineer (opus): 테스트 + 마이그레이션 정합성
- code-reviewer (opus): SQLite 쿼리 품질, 보안 검증
- devops (haiku): modernc.org/sqlite 의존성 + 빌드 검증

## 실행 순서

### Step 0: P3-0 Embedded DB 전환 (3-4 커밋)
1. devops: go.mod에 modernc.org/sqlite 추가, 빌드 검증
2. architect: SQLite 스키마 설계 (events, agents, traces, policies, memory 테이블)
3. go-backend: store/db.go + db_event_store.go + db_agent_store.go + migration.go
4. test-engineer: JSONL → SQLite 마이그레이션 정합성 테스트

### Step 1: P3-1 AI Agent Debugger (4-5 커밋)
1. go-backend: core/debugger.go + runtime/trace_builder.go + store/trace_store.go + IPC + CLI
2. swift-frontend: DebuggerPayloads.swift + DebuggerViewModel.swift + DebuggerViews.swift

### Step 2: P3-2 Org Policy Engine (3-4 커밋)
1. architect: 정책 YAML 스키마 + 상속 규칙 리뷰
2. go-backend: core/policy.go + runtime/policy_engine.go + policy_loader.go + store/policy_store.go + CLI

### Step 3: P3-3 Persistent Memory Graph (3-4 커밋)
1. go-backend: core/memory.go + runtime/memory_collector.go + memory_graph.go + store/memory_store.go + CLI

### Step 4: P3-4 Autonomous Maintenance (2-3 커밋)
1. go-backend: core/maintenance.go + runtime/maintenance_runner.go + store/maintenance_store.go + CLI

### Step 5: P3-5 Pack Marketplace (2-3 커밋)
1. go-backend: core/pack.go + runtime/pack_manager.go + store/pack_store.go + CLI

## 빌드/테스트 검증 (매 커밋 후)
go test ./... -race -count=1
go build ./go/cmd/ham ./go/cmd/hamd
swift build --disable-sandbox
swift test --disable-sandbox

## 완료 조건
- [ ] go test ./... -race PASS
- [ ] swift test --disable-sandbox PASS
- [ ] SQLite DB 파일 생성 + JSONL 마이그레이션 성공
- [ ] ham debug replay <session-id> 실행 시 타임라인 출력
- [ ] Studio 디버거 탭에서 세션 트레이스 표시
- [ ] .ham/policies/*.yaml 로딩 + 위반 감지 동작
- [ ] ham memory add/search 동작 (FTS5 검색)
- [ ] ham maintain run <job> 동작
- [ ] ham pack install/list 동작
- [ ] 기존 Phase 1/2 기능 회귀 없음
```
