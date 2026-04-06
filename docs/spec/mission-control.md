# Mission Control MVP 기능 명세서

> Phase 1 | 2026-04-06 작성 | ham-agents v2.0

---

## 목차

1. [개요](#1-개요)
2. [아키텍처 제약 조건](#2-아키텍처-제약-조건)
3. [Phase 1 ADR 상태표](#3-phase-1-adr-상태표)
4. [기능 의존성 그래프](#4-기능-의존성-그래프)
5. [P1-0. 신뢰성 기반 다지기](#5-p1-0-신뢰성-기반-다지기)
6. [P1-1. 이벤트 스키마 확장 + Artifact Capture](#6-p1-1-이벤트-스키마-확장--artifact-capture)
7. [P1-2. 실시간 Session Graph](#7-p1-2-실시간-session-graph)
8. [P1-3. Notification Inbox (읽기 전용)](#8-p1-3-notification-inbox-읽기-전용)
9. [P1-4. 비용/토큰 텔레메트리 v1](#9-p1-4-비용토큰-텔레메트리-v1)
10. [P1-5. 이벤트 브로드캐스트 기반](#10-p1-5-이벤트-브로드캐스트-기반)
11. [Claude Code 참조 스펙](#11-claude-code-참조-스펙)
12. [경쟁 제품 비교](#12-경쟁-제품-비교)

---

## 1. 개요

ham-agents는 macOS 메뉴바 앱으로 Claude Code 세션을 픽셀 햄스터로 시각화하는 도구다. Mission Control MVP는 이 앱을 **Claude Code AgentOps 플랫폼**으로 진화시키는 첫 단계다.

### 제품 축: Observe + Direct + Govern

ham-agents 는 AgentOps 플랫폼으로서 세 가지 축을 모두 제공한다:

- **Observe (관찰)** — 세션 트리, 이벤트 타임라인, 상태 아이콘. Phase 1 Mission Control MVP 의 주 범위
- **Direct (지시)** — Studio 탭의 내장 PTY 에서 에이전트에 직접 타이핑, 중단, 피드백. Phase 2 에서 enable
- **Govern (통제)** — hook.permission-request 를 PTY 층에서 가로채 approve/deny. Phase 2 Approval Interception + Phase 3 Policy Engine 이 이 축 담당

경쟁 제품 중 LangSmith / AgentOps.ai / Helicone 은 Observe 만 제공한다. ham-agents 는 Direct + Govern 까지 커버하는 첫 Claude Code 네이티브 플랫폼이다.

> Phase 2 미리보기: embedded PTY 런타임이 Direct + Govern 축을 실제 동작 가능하게 만드는 전환점이다. 기술 결정은 `tech-migration.md` ADR-2, UX 는 `ham-studio.md` P2-1 ~ P2-3 참조.

**Phase 1 범위**: CLI (`ham`) + 메뉴바 확장. ham Studio (별도 창/웹 UI)는 Phase 2로 미룸.

> **Phase 1 MVP 범위**: `P1-0 (Reliability) + P1-1 (Event schema) + P1-2 (Session graph) + P1-3 (Notification inbox) + 조건부 P1-4 (Cost telemetry)`. P1-5 는 Phase 2 이관.

**실행 순서**: P1-0 → P1-1 → (P1-2 ∥ P1-3) → 조건부 P1-4

---

## 왜 ham-agents 를 써야 하는가

### 시나리오 1: 멀티세션 운영

개발자가 iTerm 탭 3개에서 Claude Code 세션을 동시에 돌리고 있다. 각각 다른 저장소 — API 서버 리팩토링, 프론트엔드 컴포넌트 추출, 인프라 스크립트 수정 — 를 맡고 있다. 에이전트가 blocked 상태인지, 사용자 입력을 기다리는지, 툴 실행 중인지 한 눈에 확인하려면 세 개의 탭을 일일이 클릭해야 한다. ham-agents 메뉴바를 보면 세션별 상태가 픽셀 아이콘으로 표시된다. 어떤 세션이 `waiting_input` 상태인지, 어떤 세션이 `running_tool` 중인지 클릭 없이 파악한다. `ham status --graph` 로 parent-child 트리 전체를 CLI 에서도 확인할 수 있다.

### 시나리오 2: 승인 워크플로 (approval loop)

Claude Code 가 destructive 한 bash 명령(`rm -rf build/`, `git push --force`) 이나 새 파일 쓰기를 시도한다. 기존에는 터미널을 직접 보고 있지 않으면 에이전트가 무엇을 하는지 알 방법이 없었다. ham-agents 는 `hook.permission-request` 이벤트를 수신하여 Notification Inbox 에 즉시 아이템을 추가하고 메뉴바 배지를 업데이트한다. 사용자는 다른 작업을 하다가도 배지를 보고 메뉴바를 열어 요청 내용을 확인한다. approve/deny 는 Phase 2 Approval Inbox 에서 처리하며, Phase 1 에서는 읽기 전용으로 요청 이력을 기록한다.

### 시나리오 3: 팀 관찰 (team observation)

여러 사람이 같은 저장소에서 Claude Code Agent Teams 를 돌린다. lead 에이전트가 task 를 분배하고 worker 에이전트들이 병렬 작업을 수행한다. `ham status` CLI 와 메뉴바 트리 뷰에서 각 worker 의 상태, 소속 팀, 현재 실행 중인 tool 을 한 화면에서 본다. `hook.task-created` / `hook.task-completed` 이벤트가 Inbox 에 기록되어 "어떤 서브태스크가 완료됐는지" 흐름을 놓치지 않는다. 로컬 우선 아키텍처이므로 외부 서버 없이 팀 전체의 에이전트 활동이 각자의 머신에서 관측된다.

### 시나리오 4: 멀티 세션 직접 지시 (Phase 2 preview)

개발자가 Studio 에서 3 개 탭으로 Claude Code 를 병렬 운영한다. 탭 A 에서 리팩토링 진행 중 "잠깐, 이 방향 말고 adapter 패턴으로 가봐" 라고 직접 타이핑한다. 탭 B 에서 Claude 가 `git push --force` 를 시도하는 순간 Studio 가 PTY 를 블록하고 approve 모달을 띄운다. 사용자는 Deny 를 누른다. 탭 C 는 조용히 테스트를 돌리고 있고, 완료되면 메뉴바 배지로만 알려준다.

이 시나리오는 Phase 1 MVP (관찰) 로는 절반만 동작한다 — Phase 2 embedded PTY + Approval Interception 이 enable 되어야 전체 플로우가 완성된다.

---

## ADR-1: Unified SessionEvent Schema (Phase 1 → Phase 3)

**Status**: Confirmed (2026-04-06)
**Scope**: docs/spec/mission-control.md, docs/spec/tech-migration.md, docs/spec/agentops-platform.md
**Replaces**: 이전 버전의 분리된 Event 확장(P1-1)과 SessionEvent(tech-migration.md 별도 정의)

### Decision
ham-agents는 단일 통합 SessionEvent 스키마를 사용한다. core.Event 구조체를 Phase 1에서 확장하며, Phase 3에서 필요한 필드도 omitempty 로 미리 추가한다. Phase 3에서 스키마를 재차 갈아엎지 않는다.

### Schema (Go)

```go
// go/internal/core/event.go — Phase 1에서 확장
type Event struct {
    // --- 기존 필드 (변경 없음) ---
    ID        string    `json:"id"`
    AgentID   string    `json:"agent_id"`
    Type      string    `json:"type"`
    Timestamp time.Time `json:"timestamp"`
    Data      string    `json:"data,omitempty"`

    // --- Phase 1 신규 (P1-1에서 추가) ---
    SessionID      string `json:"session_id,omitempty"`
    ParentAgentID  string `json:"parent_agent_id,omitempty"`
    TaskID         string `json:"task_id,omitempty"`
    TaskName       string `json:"task_name,omitempty"`
    TaskDesc       string `json:"task_desc,omitempty"`
    ArtifactType   string `json:"artifact_type,omitempty"`
    ArtifactRef    string `json:"artifact_ref,omitempty"`
    ToolName       string `json:"tool_name,omitempty"`
    ToolInput      string `json:"tool_input,omitempty"`    // truncated, path-only for file tools
    ToolType       string `json:"tool_type,omitempty"`
    ToolDurationMs int64  `json:"tool_duration_ms,omitempty"`

    // --- Phase 3에서 사용, Phase 1에 omitempty 로 미리 추가 ---
    Source          string  `json:"source,omitempty"`           // hook | transcript | manual | inferred
    Confidence      float64 `json:"confidence,omitempty"`       // 0.0 ~ 1.0
    ConfidenceModel string  `json:"confidence_model,omitempty"` // "rule-v1" 등 식별자
    Cost            float64 `json:"cost,omitempty"`             // USD or token-equivalent. Phase 1에서는 0
    ApprovalState   string  `json:"approval_state,omitempty"`   // pending | approved | denied | auto
    Payload         []byte  `json:"payload,omitempty"`          // 구조화되지 않은 추가 데이터 (JSON bytes)
}
```

### Rationale
- 단일 스키마여야 Phase 1 데이터로 Phase 3 기능이 동작한다
- omitempty 사용으로 기존 consumer는 영향 없음
- Cost/Confidence/ApprovalState는 Phase 1에서 항상 zero-value. Phase 3에서 채움
- Source 필드로 "이 이벤트가 hook에서 왔는지, transcript 파싱에서 왔는지" 구분 가능

### Constraints (honesty section)
- hook 으로는 Cost 를 추출할 수 없음 → Phase 1 에서는 Cost 필드가 존재하지만 값은 항상 0
- parent_agent_id 는 현재 hook 에서 직접 제공되지 않음 → SubAgents 추론 경로로만 채움 (current-state.md 참조)
- ToolInput 은 파일 경로 계열 툴(Edit/Write/Read)에서 경로만 기록. 본문은 저장하지 않음
- Payload 는 "확장 슬롯" 이며 Phase 1 에서는 미사용

### Migration
Phase 1 시작 시점에 go/internal/core/event.go 를 위 스키마로 확장. 기존 events.jsonl 은 추가 필드 누락 상태로 남으며 consumer 는 omitempty 로 정상 디코딩. 별도 migration script 불필요.

---

## 2. 아키텍처 제약 조건

이 명세서의 모든 설계는 아래 제약을 준수한다.

| 제약 | 설명 | 영향 |
|------|------|------|
| IPC: 요청-응답 전용 | Unix 소켓, 연결당 1회 JSON Request → JSON Response. 스트리밍 없음 | WebSocket/SSE 불가. 실시간성은 long-polling에 의존 |
| Long-polling 한계 | `events.follow`: 200ms 폴링 간격, 60초 max wait. 유일한 준실시간 경로 | 최소 200ms 지연. 60초마다 재연결 필요 |
| Hook 단방향 | Claude Code → hamd 방향만 가능. 역방향 채널 없음 | hamd에서 Claude Code 세션에 명령 전송 불가 |
| Hook에 토큰/비용 없음 | 어떤 hook payload에도 token count, cost 필드 없음 | P1-4 텔레메트리의 데이터 소스가 불확실 |
| Hook에 파일 경로만 | diff 내용, 파일 본문은 hook payload에 포함되지 않음 | Artifact 캡처 시 별도 파일 읽기 필요 |
| Hook에 parent_id 없음 | SubagentStart/Stop에 부모 에이전트 ID가 없음 | parent-child 관계는 hamd 내부 추적에 의존 |
| Hook 출력 10,000자 제한 | hook handler가 반환하는 stdout 최대 크기 | 대용량 응답 불가 |
| 수정 완료된 버그 | C-1 (TOCTOU), C-2 (Event ID), H-4 (IPC read deadline), H-5 (이벤트 로그 성장), H-9 (FollowEvents wait cap) | 이 항목들은 P1-0 범위에서 제외 |

---

## 3. Phase 1 ADR 상태표

| # | 결정 사항 | 관련 기능 | 상태 | 비고 |
|---|----------|----------|------|------|
| ADR-1 | SessionEvent 스키마: `core.Event` additive 확장 | P1-1, P1-5 | **확정** | 기존 JSONL 하위 호환. 모든 새 필드 `omitempty` |
| ADR-2 | Approval 경로: 외부 permission 승인 API 가용성 | P1-3 | **미정 -- 조사 필요** | Phase 1은 읽기 전용으로 스코프 축소 |
| ADR-3 | 비용 데이터 소스: Claude Code 토큰/비용 노출 경로 | P1-4 | **미정 -- 조사 필요** | 시나리오 A/B/C 분기 |
| ADR-4 | Artifact 저장: 인라인 4KB / 파일 1MB / 총 500MB | P1-1 | **확정** | LRU 정리, Privacy 설정 연동 |
| ADR-5 | UI 표면: Phase 1은 CLI + 메뉴바, Studio는 Phase 2 | P1-2 | **확정** | |
| ADR-6 | Swift/Go IPC 동기화: 수동, Phase 2에서 코드 생성 검토 | P1-0 | **확정** | hook 커맨드는 Go 전용 |

---

## 4. 기능 의존성 그래프

```
P1-0 신뢰성 기반
 │
 ▼
P1-1 이벤트 스키마 확장 + Artifact
 │
 ├──────────────┐
 ▼              ▼
P1-2 Session   P1-3 Notification
Graph          Inbox
 │              │
 └──────┬───────┘
        ▼
  조건부 P1-4 비용/토큰 텔레메트리
```

> **P1-5 는 Phase 2 초입 과제로 분리.** Phase 1 에서는 기존 long-polling 방식을 유지한다.

**의존성 상세:**
- P1-1은 P1-0 완료 후 시작 (안정적인 IPC/레지스트리 필수)
- P1-2와 P1-3은 P1-1의 확장된 Event 스키마 사용 (병렬 진행 가능)
- P1-4는 ADR-3 조사 결과에 의존 (데이터 소스 확정 후 시작). Scenario C 시 Phase 2 이관
- P1-5는 Phase 2 초입으로 이관. Phase 1 크리티컬 패스에서 제외

---

## 5. P1-0. 신뢰성 기반 다지기

### 5-1. 기능 설명 + 사용자 시나리오

**무엇을 하나**: 잔존 CRITICAL/HIGH 버그 수정, Swift/Go IPC enum 동기화, 계약 테스트 추가.

**누가 쓰나**: 모든 ham-agents 사용자. 현재 발생 가능한 크래시/데이터 손실을 제거.

**시나리오**:
- 사용자가 Claude Code 세션 3개를 동시 실행. 동시 hook 이벤트가 registry를 손상시키지 않아야 함 (C-1 이미 수정)
- Swift 메뉴바 앱이 Go에만 존재하는 IPC 커맨드를 수신해도 크래시하지 않아야 함 (H-10)
- 데몬이 24시간 이상 실행되어도 이벤트 로그가 무한 성장하지 않아야 함 (H-5 이미 수정)

### 5-2. 필요한 데이터

| 데이터 | 현재 상태 | 필요한 조치 |
|--------|----------|------------|
| Go IPC Command 상수 (52개) | `go/internal/ipc/ipc.go:22-74` 에 정의 | 변경 없음 |
| Swift DaemonCommand enum (16개) | `Sources/HamCore/DaemonIPC.swift:3-20` | 6개 추가 + unknown fallback |
| Hook contract schema | 암묵적. 테스트 없음 | contract test 추가 |
| 동시성 안전성 | C-1 수정 완료, 추가 레이스 미검증 | race test 추가 |

### 5-3. Go 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `go/internal/ipc/contract_test.go` | **[신규]** 모든 hook 커맨드에 대한 Request 직렬화/역직렬화 왕복 테스트 |
| `go/internal/runtime/registry_test.go` | 동시 hook 이벤트 레이스 테스트 (goroutine 100개 동시 mutateAgent) |
| `go/internal/store/events_test.go` | truncate 경계 조건 테스트, 10,000건 초과 시 pruning 검증 |
| `go/internal/ipc/server.go` | M-3 (이중 RecordHookSessionSeen) 수정, M-5 (요청 크기 제한 1MB) 추가 |
| `go/cmd/hamd/main.go` | M-2 (pollRuntimeState 에러 로깅), M-4 (err 변수 섀도잉) 수정 |

**미수정 HIGH 버그 처리 계획:**

| 버그 | 상태 | 조치 |
|------|------|------|
| H-1. 소켓 half-close 누락 | 미수정 | Swift 측에서 `shutdown(fd, SHUT_WR)` 추가 |
| H-2. SIGWINCH 고루틴 릭 | 미수정 | `signal.Stop` 후 `close(sigwinch)` |
| H-3. stdin-to-ptmx 고루틴 릭 | 미수정 | CLI 종료 시 문제없으므로 LOW로 재분류 |
| H-6. UserNotificationSink Task 미관리 | 미수정 | `[weak self]` 캡처 + Task 핸들 저장 |
| H-7. interactionHandler 동기화 없음 | 미수정 | NSLock 보호 추가 |
| H-8. MenuBarViewModel deinit 위반 | 미수정 | `stop()` 메서드로 Task 취소 이동 |
| H-10. Swift/Go enum 불일치 | 미수정 | 아래 상세 참조 |
| H-11. 크로스 프로세스 파일 잠금 | 미수정 | `flock(2)` advisory locking |

### 5-4. Swift 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `Sources/HamCore/DaemonIPC.swift` | DaemonCommand에 6개 케이스 추가: `registerManaged`, `managedStop`, `managedExited`, `agentsRename`, `agentsOpenTarget`, `tmuxSessions`. `unknown` fallback case 추가 |
| `Sources/HamAppServices/DaemonClient.swift` | H-1 수정: write 후 `shutdown(fd, SHUT_WR)` 호출. C-5 이미 수정됨 |
| `Sources/HamNotifications/UserNotificationSink.swift` | H-6 수정: `[weak self]` + Task 핸들 저장. H-7 수정: `interactionHandler`에 NSLock |
| `Sources/HamAppServices/MenuBarViewModel.swift` | H-8 수정: deinit 본문 제거, `stop()` 메서드에서 Task 취소. M-6 수정: heartbeat 중복 방지 분기 |

### 5-5. IPC 변경사항

IPC 프로토콜 자체는 변경 없음. Swift enum 확장만 수행.

**동기화 대상 커맨드 (Go → Swift 추가):**

```
register.managed    — managed 에이전트 등록
managed.stop        — managed 에이전트 중지
managed.exited      — managed 에이전트 종료 알림
agents.rename       — 에이전트 이름 변경
agents.open_target  — 에이전트 터미널 열기
tmux.sessions       — tmux 세션 목록
```

**동기화하지 않는 커맨드 (Go 전용, hook.* 계열):**
- `hook.session-start`, `hook.session-end`, `hook.tool-start`, `hook.tool-done` 등 27개 hook 커맨드
- Hook 커맨드는 Claude Code → hamd 전용이므로 Swift 앱에서 발신할 일 없음

### 5-6. 선행 작업 / 의존성

- 없음. 즉시 시작 가능
- C-1, C-2, H-4, H-5, H-9는 이미 수정 완료

### 5-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 이유 | 대안 |
|------------|------|------|
| IPC enum 자동 동기화 | Go와 Swift 사이에 코드 생성 파이프라인 없음 | Phase 1: 수동 동기화. Phase 2: protobuf 또는 코드 생성 검토 (ADR-6) |
| H-3 stdin-to-ptmx 완전 수정 | `os.Stdin.Read()` 블로킹은 Go 런타임 한계 | CLI 종료 시 프로세스와 함께 정리되므로 실질적 영향 없음. LOW로 재분류 |
| H-11 크로스 프로세스 잠금 완전 보장 | `flock(2)`는 advisory lock이라 협력적이지 않은 프로세스는 무시 가능 | 현실적으로 hamd + Swift 앱만 접근하므로 advisory lock으로 충분 |

---

## 6. P1-1. 이벤트 스키마 확장 + Artifact Capture

### 6-1. 기능 설명 + 사용자 시나리오

**무엇을 하나**: `core.Event` 구조체(현재 12개 필드)를 additive하게 확장하여 세션 컨텍스트, 태스크 컨텍스트, artifact, 도구 컨텍스트를 캡처한다.

**누가 쓰나**: 
- CLI 사용자: `ham logs`에서 도구 호출 상세, artifact 내용 확인
- 메뉴바 사용자: 에이전트 상세 뷰에서 최근 도구 활동 확인
- 시스템 내부: P1-2 (Session Graph), P1-3 (Inbox), P1-5 (EventBus)의 데이터 소스

**시나리오**:
- 개발자가 `ham logs api-refactor`를 실행하면 tool 호출 이력이 도구명, 입력, 소요시간과 함께 표시됨
- SubAgent가 생성되면 ParentAgentID가 기록되어 P1-2에서 트리 구성에 사용됨
- 4KB 초과 artifact(diff 출력 등)는 별도 파일로 저장되고 `ham logs --artifact <event-id>`로 조회 가능

### 6-2. 필요한 데이터

| 데이터 | 현재 상태 | 소스 |
|--------|----------|------|
| SessionID | hook payload의 `ipc.Request.SessionID`에 존재하나 Event에 미저장 | `go/internal/ipc/server.go` prepareHookRequest |
| ParentAgentID | `Agent.SubAgents[]`에 자식 목록은 있으나, Event에 부모 ID 미기록 | RecordHookAgentSpawned에서 부모 Agent context 사용 |
| TaskName, TaskDesc | hook.task-created에서 수신하나 Event에 미저장 | `go/internal/runtime/managed_state.go` RecordHookTaskCreated |
| ToolName | Agent.RecentTools에 저장하나 Event에는 Summary 문자열만 | RecordHookToolStart |
| ToolInput | `ToolInputPreview` (문자열 미리보기)만 저장. 전체 JSON 미캡처 | Claude Code hook payload의 `tool_input` 필드 |
| ToolDuration | RecordHookToolDone에서 계산하나 Event에 미저장 | ToolStart/ToolDone 시간 차이 |
| Artifact 내용 | 미캡처 | hook payload에 파일 경로만 제공. 내용은 별도 읽기 필요 |

### 6-3. Go 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `go/internal/core/agent.go` | Event 구조체에 10개 필드 추가 (모두 `omitempty`). 아래 스키마 참조 |
| `go/internal/runtime/managed_state.go` | RecordHookToolStart: ToolName, ToolInput, ToolType 채우기. RecordHookToolDone: ToolDuration 계산. RecordHookTaskCreated: TaskName, TaskDesc. RecordHookAgentSpawned: ParentAgentID |
| `go/internal/ipc/server.go` | prepareHookRequest: SessionID를 Event에 전달 |
| `go/internal/store/artifacts.go` | **[신규]** ArtifactStore 인터페이스 + FileArtifactStore 구현 |
| `go/internal/store/events.go` | Append 시 ArtifactData 크기 판단 → 4KB 초과면 ArtifactStore에 위임, ArtifactRef 설정 |

**확장 Event 스키마:**

> **참고**: P1-1에서 추가하는 필드 목록과 Phase 3 선탑재 필드는 **ADR-1** (이 문서 상단)에 단일 정의된다. 아래는 P1-1 구현 범위 요약이며, 권위 있는 스키마는 ADR-1을 따른다.

P1-1에서 추가하는 필드: `SessionID`, `ParentAgentID`, `TaskID`, `TaskName`, `TaskDesc`, `ArtifactType`, `ArtifactRef`, `ToolName`, `ToolInput`, `ToolType`, `ToolDurationMs`

Phase 1에 omitempty로 선탑재 (Phase 3에서 채움): `Source`, `Confidence`, `ConfidenceModel`, `Cost`, `ApprovalState`, `Payload`

전체 Go 스키마 → **ADR-1 Schema (Go)** 참조.

**Artifact 저장 정책:**

| 조건 | 저장 위치 | 필드 |
|------|----------|------|
| 4KB 이하 | Event.ArtifactData에 인라인 | ArtifactData |
| 4KB 초과, 1MB 이하 | `~/Library/Application Support/ham-agents/artifacts/{agent_id}/{event_id}.json` | ArtifactRef (경로) |
| 1MB 초과 | truncate + `[truncated]` 마커 후 1MB로 저장 | ArtifactRef |
| artifacts 디렉토리 총합 | 500MB 초과 시 LRU 정리 | Privacy 설정의 `eventHistoryRetentionDays` 적용 |

**FileArtifactStore 인터페이스:**

```go
type ArtifactStore interface {
    Save(agentID, eventID string, data []byte) (ref string, err error)
    Load(ref string) ([]byte, error)
    Prune(maxTotalBytes int64, retentionDays int) error
}
```

### 6-4. Swift 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `Sources/HamCore/Agent.swift` | AgentEvent 구조체에 대응 필드 추가 (sessionID, parentAgentID, taskName, toolName 등). 모두 Optional |
| `Sources/HamAppServices/EventPresentation.swift` | 새 필드 활용: tool 이벤트에 도구명+소요시간 표시, artifact 있으면 `[artifact]` 마커 |
| `apps/macos/HamMenuBarApp/Sources/MenuBarViews.swift` | 에이전트 상세 뷰에서 최근 tool 활동을 도구명+타입으로 표시 |

### 6-5. IPC 변경사항

| 변경 | 상세 |
|------|------|
| `events.follow` Response | Event JSON에 새 필드 포함 (omitempty이므로 하위 호환) |
| `events.list` Response | 동일 |
| 신규 커맨드 없음 | Artifact 조회는 CLI에서 직접 파일 읽기 (IPC 불필요) |

### 6-6. 선행 작업 / 의존성

- **P1-0 완료 필수**: 안정적인 registry + IPC가 전제
- ADR-1 (확정): additive 확장, 기존 JSONL 하위 호환
- ADR-4 (확정): artifact 저장 정책

### 6-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 이유 | 대안 |
|------------|------|------|
| 전체 tool_input JSON 캡처 | Claude Code hook payload에 `tool_input`이 full JSON으로 제공되나, 크기가 수 MB 가능 (Write 도구의 파일 내용 등) | ToolInputPreview (기존)를 유지하되, 4KB 이하만 ToolInput에 저장. 초과분은 artifact로 분리 |
| Diff 내용 캡처 | Hook payload에 파일 경로만 포함, diff 본문 없음 | ArtifactType="diff"로 표시만 하고, 실제 diff는 git을 통해 조회하도록 안내. Phase 2에서 PreToolUse hook에서 파일 스냅샷 → PostToolUse에서 diff 계산 검토 |
| ParentAgentID 직접 수신 | Hook payload의 SubagentStart에 parent_id 필드 없음 | hamd 내부에서 추적: SubagentStart를 수신한 에이전트가 parent. `RecordHookAgentSpawned`의 호출 컨텍스트에서 parent AgentID를 Event에 기록 |

---

## 7. P1-2. 실시간 Session Graph

### 7-1. 기능 설명 + 사용자 시나리오

**무엇을 하나**: Claude Code 세션들을 parent-child 트리로 시각화. 각 노드에 상태, blocking reason, confidence를 표시한다.

**누가 쓰나**:
- CLI 사용자: `ham status --graph`로 전체 세션 구조 한눈에 파악
- 메뉴바 사용자: 에이전트 리스트가 flat 목록 대신 트리로 표시

**시나리오**:
- 개발자가 lead agent를 실행하고 3개의 sub-agent가 자동 생성됨. `ham status --graph`로 트리 구조를 확인하고, 어떤 sub-agent가 permission 대기 중인지 즉시 파악
- 메뉴바에서 에이전트를 클릭하면 해당 에이전트와 그 sub-agent들이 들여쓰기된 트리로 표시

**CLI 출력 예시:**
```
$ ham status --graph
SESSION GRAPH (3 agents, 1 blocked)  2026-04-05 15:30:00

+-  api-refactor [thinking] confidence:0.92
|  +- test-writer [running_tool: Bash] confidence:1.00
|  +- doc-updater [waiting_input] confidence:0.45  >> permission: Write
+- release-notes [done] confidence:1.00
```

### 7-2. 필요한 데이터

| 데이터 | 현재 상태 | 소스 |
|--------|----------|------|
| Agent 목록 + 상태 | `RuntimeSnapshot.Agents` (Agent 36개 필드) | `go/internal/runtime/registry.go` Snapshot() |
| SubAgent 관계 | `Agent.SubAgents []SubAgentInfo` (AgentID, Role, Status 등) | RecordHookAgentSpawned/Finished에서 갱신 |
| Blocking reason | Agent.Status + Agent.ErrorType + Agent.LastNotificationType | 기존 필드 조합으로 도출 |
| Confidence | Agent.StatusConfidence (float64) | inference 엔진 또는 hook 기반 설정 |
| P1-1 확장 필드 | Event.ParentAgentID, Event.SessionID | P1-1 완료 후 사용 가능 |

### 7-3. Go 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `go/internal/core/graph.go` | **[신규]** SessionNode, SessionGraph 타입 + `BuildSessionGraph(snapshot RuntimeSnapshot) SessionGraph` 변환 함수 |
| `go/cmd/ham/cmd_status.go` (또는 `parse.go` 내 status 핸들러) | `--graph` 플래그 추가. SessionGraph를 받아 트리 렌더링 |
| `go/internal/ipc/ipc.go` | `CommandStatusGraph` 상수 추가 (또는 기존 `status` 응답에 graph 필드 포함) |
| `go/internal/ipc/server.go` | status 핸들러에서 graph 옵션 처리 |

**SessionGraph 데이터 모델:**

```go
type SessionNode struct {
    Agent       Agent         `json:"agent"`
    Children    []SessionNode `json:"children,omitempty"`
    BlockReason string        `json:"block_reason,omitempty"` // "permission_request" | "waiting_input" | "error" | "disconnected"
    Depth       int           `json:"depth"`
}

type SessionGraph struct {
    Roots        []SessionNode `json:"roots"`
    TotalCount   int           `json:"total_count"`
    BlockedCount int           `json:"blocked_count"`
    GeneratedAt  time.Time     `json:"generated_at"`
}
```

**BuildSessionGraph 로직:**
1. `RuntimeSnapshot.Agents`를 순회
2. `Agent.SubAgents`의 AgentID로 자식 에이전트 매칭 (SubAgentInfo.AgentID → Agent lookup)
3. parent가 없는 에이전트 = Root 노드
4. Status가 `waiting_input` / `error` / `disconnected`이면 BlockReason 설정
5. StatusConfidence < 0.5이면 BlockReason에 "(low confidence)" 추가

### 7-4. Swift 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `Sources/HamCore/Agent.swift` | SessionNode, SessionGraph Codable 구조체 추가 (Go 대응) |
| `Sources/HamAppServices/MenuBarViewModel.swift` | agents 배열을 트리로 변환하는 로직 추가. SubAgents 필드 기반 parent-child 그룹핑 |
| `apps/macos/HamMenuBarApp/Sources/MenuBarViews.swift` | 에이전트 리스트를 들여쓰기된 트리로 렌더링. DisclosureGroup 또는 indent 기반 |

### 7-5. IPC 변경사항

| 변경 | 상세 |
|------|------|
| 옵션 A: 기존 `status` 확장 | Request에 `graph: true` 필드 추가. Response에 `SessionGraph` 포함 |
| 옵션 B: 별도 `status.graph` 커맨드 | 새 커맨드 추가. 기존 status 변경 없음 |
| **권장: 옵션 A** | Graph는 Snapshot의 뷰 변환이므로 별도 커맨드 불필요. Request에 옵션 필드 추가가 단순 |

### 7-6. 선행 작업 / 의존성

- **P1-1 완료 권장**: ParentAgentID가 Event에 기록되어야 정확한 트리 구성 가능
- P1-1 없이도 `Agent.SubAgents` 배열로 기본 트리 구성은 가능 (degraded mode)
- SubAgents 배열은 최대 20개로 제한됨 (RecordHookAgentSpawned에서 cap)

### 7-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 이유 | 대안 |
|------------|------|------|
| 실시간 그래프 업데이트 | IPC가 요청-응답 전용. push 불가 | CLI: 실행 시점 스냅샷. 메뉴바: 기존 5초 폴링 + 15초 event follow로 갱신 |
| 크로스 세션 부모-자식 (다른 터미널의 agent team) | SubagentStart hook은 같은 세션 내 subagent만 보고. 별도 터미널의 teammate는 TeammateIdle로만 감지 | TeammateIdle의 TeamRole로 "같은 팀" 표시는 가능하나 정확한 트리 관계는 불가. 같은 팀 에이전트를 그룹으로 묶되, parent-child 아닌 peer 관계로 표시 |
| Blocking reason의 자동 해소 감지 | Permission 승인 후 Claude Code가 별도 hook을 보내지 않음 (다음 tool 시작으로 간접 감지) | 다음 RecordHookToolStart 수신 시 BlockReason 자동 클리어 |

---

## 8. P1-3. Notification Inbox (읽기 전용)

### 8-1. 기능 설명 + 사용자 시나리오

**무엇을 하나**: PermissionRequest, Notification, TaskComplete, Error, Stop을 하나의 수신함으로 통합. 메뉴바에서 알림 목록을 보고, 해당 에이전트의 터미널로 점프한다.

**누가 쓰나**:
- 멀티 세션 사용자: 어떤 세션이 주의를 필요로 하는지 한 곳에서 확인
- CLI 사용자: `ham inbox`로 최근 알림 확인

**시나리오**:
- 3개 세션을 동시 실행 중. 메뉴바의 뱃지에 "2"가 표시됨. 클릭하면 "api-refactor: permission request (Write tool)"과 "test-runner: error (rate_limit)"이 보임. 각 항목을 클릭하면 해당 에이전트의 터미널로 이동
- `ham inbox`에서 최근 알림 목록 출력. `ham inbox --mark-read`로 전체 읽음 처리

**Phase 1 한계**: 읽기 전용. 승인/거절 버튼 없음. 터미널 이동만 가능.

### 8-2. 필요한 데이터

| 데이터 | 현재 상태 | 소스 |
|--------|----------|------|
| Permission request | RecordHookPermissionRequest에서 Agent 상태 변경 + Event 기록 | hook.permission-request |
| Notification 유형 | RecordHookNotification에서 NotificationType 기록 | hook.notification |
| Task 완료 | RecordHookTaskCompleted에서 TeamTaskCompleted++ | hook.task-completed |
| Error 정보 | RecordHookStopFailure에서 ErrorType 설정 | hook.stop-failure |
| Stop 정보 | RecordHookStop에서 LastAssistantMessage 저장 | hook.stop |
| Agent OpenTarget | Agent.OpenTarget (URL 또는 workspace 경로) | 기존 필드 |

### 8-3. Go 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `go/internal/core/inbox.go` | **[신규]** InboxItem 구조체. InboxItemType 상수 |
| `go/internal/runtime/inbox.go` | **[신규]** InboxManager: Event 스트림에서 InboxItem 생성. 최근 100개 유지. Read/MarkRead 메서드 |
| `go/internal/ipc/ipc.go` | `CommandInboxList`, `CommandInboxMarkRead` 상수 추가. Request/Response에 inbox 필드 |
| `go/internal/ipc/server.go` | inbox.list: InboxManager.List() 호출. inbox.mark-read: InboxManager.MarkRead(id) 호출 |
| `go/cmd/ham/parse.go` | `ham inbox` 커맨드 추가. `--mark-read`, `--type` 필터 옵션 |

**InboxItem 모델:**

```go
type InboxItem struct {
    ID         string    `json:"id"`
    AgentID    string    `json:"agent_id"`
    AgentName  string    `json:"agent_name"`
    Type       string    `json:"type"`       // "permission_request" | "notification" | "task_complete" | "error" | "stop"
    Summary    string    `json:"summary"`
    ToolName   string    `json:"tool_name,omitempty"`
    OccurredAt time.Time `json:"occurred_at"`
    Read       bool      `json:"read"`
    Actionable bool      `json:"actionable"` // Phase 1: 항상 false
}
```

**Hook → InboxItem 매핑:**

| Hook Command | InboxItem.Type | Summary 생성 |
|---|---|---|
| `hook.permission-request` | `permission_request` | "Approve {ToolName}?" |
| `hook.permission-denied` | `permission_request` | "Permission denied: {ToolName}" |
| `hook.notification` | `notification` | NotificationType 필드 |
| `hook.task-completed` | `task_complete` | TaskName 필드 |
| `hook.stop-failure` | `error` | ErrorType 필드 (rate_limit, auth_failed 등) |
| `hook.stop` | `stop` | LastAssistantMessage (최대 200자 truncate) |

**InboxManager 동작:**
- 생성 시 Registry에서 Event 스트림 구독 (현재: recordEvent 호출 시 콜백. P1-5 이후: EventBus 구독)
- Event.Type이 위 매핑에 해당하면 InboxItem 생성
- 메모리에 최근 100개 유지 (ring buffer)
- 영속화: `~/Library/Application Support/ham-agents/inbox.json` (앱 재시작 시 복원)

### 8-4. Swift 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `Sources/HamCore/DaemonIPC.swift` | `inboxList`, `inboxMarkRead` 커맨드 추가 |
| `Sources/HamCore/DaemonPayloads.swift` | InboxItem Codable 구조체 추가 |
| `Sources/HamAppServices/InboxViewModel.swift` | **[신규]** InboxViewModel: 폴링 주기에 맞춰 inbox 갱신, unreadCount 계산, markRead 호출 |
| `Sources/HamAppServices/MenuBarViewModel.swift` | InboxViewModel 소유. unreadCount를 뱃지 수로 노출 |
| `apps/macos/HamMenuBarApp/Sources/MenuBarViews.swift` | Inbox 섹션 추가: unread 뱃지, InboxItem 리스트, 클릭 시 OpenTarget으로 이동 |

### 8-5. IPC 변경사항

**신규 커맨드:**

```
inbox.list
  Request:  { command: "inbox.list", type_filter?: string, unread_only?: bool }
  Response: { items: InboxItem[], unread_count: int }

inbox.mark-read
  Request:  { command: "inbox.mark-read", inbox_item_id?: string }  // id 없으면 전체 읽음
  Response: { success: bool }
```

### 8-6. 선행 작업 / 의존성

- **P1-1 완료 필수**: 확장된 Event에서 ToolName, TaskName 등을 InboxItem에 활용
- P1-1 없이도 기존 Event.Summary에서 InboxItem 생성은 가능 (degraded mode)

### 8-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 이유 | 대안 |
|------------|------|------|
| Permission 승인/거절 | Claude Code에 외부에서 permission을 승인하는 공개 API가 확인되지 않음 (ADR-2 미정) | Phase 1: 읽기 전용. "Go to terminal" 버튼으로 사용자가 직접 터미널에서 승인. Phase 2: API 확인 후 Approval Inbox로 업그레이드 |
| 실시간 알림 push | IPC 요청-응답 전용. 서버 → 클라이언트 push 불가 | 메뉴바: 기존 폴링 주기 (5초 refresh)에 inbox 갱신 포함. macOS 알림은 기존 HamNotifications 경로 활용 |
| Inbox 항목 100개 초과 보관 | 메모리 + 파일 크기 제한 | 100개 ring buffer. 오래된 항목은 events.jsonl에 이력으로 남아있으므로 `ham logs`로 조회 가능 |
| Permission request의 정확한 해소 시점 | Claude Code가 "permission granted" hook을 보내지 않음 | 다음 tool 시작 시 해당 permission_request InboxItem을 자동으로 "resolved"로 표시 |

---

## 9. P1-4. 비용/토큰 텔레메트리 v1

**Scope Gate**: ADR-3 (비용 데이터 소스 조사) 는 Phase 1 시작과 동시 병렬 진행한다. 결과에 따른 분기:
- Scenario A (hook 파싱 가능): Phase 1 포함
- Scenario B (transcript 파싱 가능): Phase 1 포함, 구현 난이도 상향
- Scenario C (데이터 없음): Phase 1 에서 제외, Phase 2 이관. 결정 데드라인: Phase 1 시작 후 1주 내.

### 9-1. 기능 설명 + 사용자 시나리오

**무엇을 하나**: Claude Code 세션별 토큰 사용량/비용을 추적하여 `ham cost` CLI와 메뉴바에 표시한다.

**누가 쓰나**:
- 비용 관리가 필요한 개발자/팀 리드
- 세션별 효율을 비교하고 싶은 사용자

**시나리오 (데이터 확보 시)**:
- `ham cost`로 오늘/이번 주/이번 달 세션별 비용 확인
- `ham cost --project ./my-app`으로 프로젝트별 비용 집계
- 메뉴바에 오늘 총 예상 비용 표시 (opt-in)

### 9-2. 필요한 데이터

**핵심 문제: Claude Code는 hook payload에 토큰/비용 정보를 포함하지 않는다.**

| 조사 경로 | 가능성 | 상세 |
|----------|--------|------|
| Hook payload 내 토큰 수 | **없음** | 27개 hook 이벤트 중 어떤 것도 token_count, cost 필드를 포함하지 않음 |
| `~/.claude/` 디렉토리 파일 | **조사 필요** | statsig, usage, billing 관련 파일 존재 여부 확인 필요 |
| `claude --usage` CLI | **조사 필요** | 사용량 조회 CLI 명령 존재 여부 확인 필요 |
| Anthropic API 대시보드 | **간접 가능** | 조직 단위 비용은 확인 가능하나 세션별 세분화 불가 |
| 세션 시작/종료 시간 | **확보 가능** | SessionStart/SessionEnd hook에서 duration 계산 가능 |

### 9-3. 시나리오별 구현 분기

#### 시나리오 A: 토큰 수 직접 획득 가능

`~/.claude/` 파일이나 CLI에서 세션별 토큰 수를 읽을 수 있는 경우.

**Go 변경사항:**

| 파일 | 변경 내용 |
|------|----------|
| `go/internal/core/cost.go` | **[신규]** CostRecord 구조체 |
| `go/internal/store/cost.go` | **[신규]** FileCostStore (JSONL 기반) |
| `go/internal/runtime/cost.go` | **[신규]** CostTracker: 세션 종료 시 토큰 데이터 수집 + CostRecord 생성 |
| `go/internal/ipc/ipc.go` | `CommandCostSummary` 상수 추가 |
| `go/cmd/ham/parse.go` | `ham cost` 커맨드 추가 |

```go
type CostRecord struct {
    AgentID      string    `json:"agent_id"`
    SessionID    string    `json:"session_id"`
    ProjectPath  string    `json:"project_path"`
    TokensIn     int64     `json:"tokens_in"`
    TokensOut    int64     `json:"tokens_out"`
    EstimatedUSD float64   `json:"estimated_usd"`
    RecordedAt   time.Time `json:"recorded_at"`
}
```

**Swift 변경사항:**

| 파일 | 변경 내용 |
|------|----------|
| `Sources/HamCore/DaemonIPC.swift` | `costSummary` 커맨드 추가 |
| `Sources/HamAppServices/MenuBarViewModel.swift` | 오늘 총 비용 표시 (opt-in) |

#### 시나리오 B: 세션 시간만 확보 가능

토큰 데이터 없음. Duration 기반 추정만 가능.

**Go 변경사항:**

| 파일 | 변경 내용 |
|------|----------|
| `go/internal/core/cost.go` | CostRecord에서 TokensIn/Out 제거. DurationSeconds, EstimatedTokens (추정) 추가 |
| 나머지 | 시나리오 A와 동일하되 추정치 표시 |

**추정 로직**: 모델별 평균 토큰 소비율 (예: claude-sonnet-4-20250514 ~1000 tokens/min active) x 활성 시간. 정확도 낮음을 UI에 명시.

#### 시나리오 C: 아무 데이터도 없음

**조치**: P1-4를 Phase 2로 이관. ADR-3에 조사 결과 문서화.

### 9-4. IPC 변경사항 (시나리오 A/B)

```
cost.summary
  Request:  { command: "cost.summary", project_path?: string, since?: string }
  Response: { records: CostRecord[], total_usd: float64, period: string }
```

### 9-5. 선행 작업 / 의존성

- **ADR-3 조사 완료 필수**: 데이터 소스가 확정되어야 시나리오 결정 가능
- P1-1 완료 권장: SessionID가 Event에 기록되어야 세션별 집계 정확

### 9-6. 구현 불가능한 부분과 대안

| 불가능한 것 | 이유 | 대안 |
|------------|------|------|
| 실시간 토큰 스트리밍 | Hook에 토큰 정보 없음. 모델 응답 중간에 관측 불가 | 세션 종료 후 사후 집계 |
| 정확한 비용 계산 | 모델별 단가가 변동. 캐시 히트/미스에 따라 실제 과금 다름 | "예상 비용"으로 표시. 정확한 비용은 Anthropic 대시보드 참조 안내 |
| 세션 내 세부 breakdown | API 호출 단위 토큰 수 불가 | 세션 단위 총합만 제공 |
| Anthropic API 직접 조회 | 사용자 API 키 필요. 보안/프라이버시 문제 | 로컬 데이터만 사용. API 연동은 Phase 3에서 opt-in으로 검토 |

---

## 10. P1-5. Event Broadcast (Phase 2 이관)

> **Deferred to Phase 2 (Ralph Round 2 재검토 결정)**: 이 항목은 사용자 가치 0인 내부 리팩터링이며 Phase 1 MVP 범위에서 제외되어 Phase 2 초입으로 이동한다. 기존 long-polling 방식을 Phase 1 에서 유지한다.

### 10-1. 기능 설명 + 사용자 시나리오

**무엇을 하나**: hamd 내부를 이벤트 발행(pub-sub) 중심으로 리팩토링. registry.recordEvent를 EventBus.Publish로 교체한다.

**누가 쓰나**: 직접적인 사용자 기능은 아님. P1-2 (Session Graph), P1-3 (Inbox), 향후 Phase 2 (Studio SSE) 등 내부 subscriber의 기반.

**시나리오**:
- Hook 이벤트 수신 → EventBus.Publish → agentReducer(상태 갱신), eventStore(로그 저장), inboxManager(알림 생성), followSubscriber(long-poll 응답) 모두 동시 처리
- 새로운 subscriber 추가 시 기존 코드 변경 없이 EventBus.Subscribe만 호출

### 10-2. 필요한 데이터

| 데이터 | 현재 상태 | 목표 상태 |
|--------|----------|----------|
| Event 발행 | `registry.recordEvent()` → `eventStore.Append()` 직접 호출 | `EventBus.Publish(event)` → fan-out |
| Event 구독 | `FollowEvents`가 `eventStore.EventsAfterID()`를 200ms 폴링 | FollowEvents가 EventBus subscriber로 전환 |
| Inbox 연동 | 없음 (P1-3에서 신규) | InboxManager가 EventBus subscriber |

### 10-3. Go 변경사항

| 파일 | 변경 내용 |
|------|----------|
| `go/internal/runtime/eventbus.go` | **[신규]** EventBus 구현: Publish(event), Subscribe(id, chan Event), Unsubscribe(id). sync.RWMutex + map[string]chan Event |
| `go/internal/runtime/registry.go` | `recordEvent` 메서드를 `EventBus.Publish`로 교체. EventBus 초기화를 Registry 생성 시 수행 |
| `go/internal/runtime/events.go` | FollowEvents를 EventBus subscription 기반으로 재작성. 기존 파일 폴링 제거 |
| `go/internal/runtime/inbox.go` | InboxManager를 EventBus subscriber로 등록 (P1-3 연동) |
| `go/internal/store/events.go` | eventStore.Append를 EventBus subscriber로 등록 (기존 직접 호출 제거) |

**현재 구조:**
```
hook → IPC server → registry.mutateAgent()
                       ├→ store.SaveAgents()     (스냅샷)
                       └→ registry.recordEvent()
                            └→ eventStore.Append()  (로그)
```

**목표 구조:**
```
hook → IPC server → registry.Publish(event)
                       ├→ agentReducer       → store.SaveAgents()
                       ├→ eventStore         → Append() (로그)
                       ├→ inboxManager       → Process() (알림)
                       └→ followSubscribers  → FollowEvents 응답
```

**EventBus 인터페이스:**

```go
type EventBus struct {
    mu          sync.RWMutex
    subscribers map[string]chan Event
}

func (b *EventBus) Publish(event Event)                    // fan-out to all subscribers
func (b *EventBus) Subscribe(id string) <-chan Event       // buffered channel (size 256)
func (b *EventBus) Unsubscribe(id string)                  // remove + close channel
```

**Publish 동작:**
- RLock으로 subscriber map 순회
- 각 subscriber channel에 non-blocking send (채널 full이면 drop + warning 로그)
- subscriber가 느려도 Publish가 블로킹되지 않음

### 10-4. Swift 변경사항

| 파일 | 변경 내용 |
|------|----------|
| 없음 | EventBus는 hamd 내부 리팩토링. IPC 프로토콜과 Swift 앱에 변경 없음 |

Swift 메뉴바 앱은 기존과 동일하게 `events.follow` IPC를 통해 이벤트를 수신한다. EventBus는 hamd 서버 내에서 FollowEvents 핸들러가 subscription 기반으로 전환되는 것이므로 외부 인터페이스 변경 없음.

### 10-5. IPC 변경사항

| 변경 | 상세 |
|------|------|
| `events.follow` | 외부 프로토콜 변경 없음. 내부 구현만 파일 폴링 → EventBus subscription으로 전환 |
| 응답 지연 특성 변화 | 기존: 200ms 간격 파일 폴링으로 최대 200ms 지연. 전환 후: EventBus에서 즉시 전달되므로 지연 감소 |

### 10-6. 선행 작업 / 의존성

- **P1-2, P1-3 안정화 후 수행 권장**: subscriber가 충분히 테스트된 후 내부 배관 교체
- P1-1 완료 필수: 확장된 Event 스키마가 EventBus를 통해 전파

### 10-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 이유 | 대안 |
|------------|------|------|
| SSE/WebSocket 기반 실시간 push | IPC가 Unix 소켓 요청-응답 전용 | Phase 1: EventBus는 hamd 내부 전용. 외부 클라이언트는 기존 long-polling 유지. Phase 2: ham Studio에서 SSE 엔드포인트 추가 검토 |
| subscriber 장애 시 backpressure | non-blocking send이므로 느린 subscriber는 이벤트 유실 | buffered channel (256)으로 버스트 흡수. 유실 시 warning 로그. FollowEvents는 eventStore.EventsAfterID() fallback으로 누락 복구 가능 |
| EventBus 영속화 | 메모리 전용이므로 hamd 재시작 시 진행 중인 subscription 유실 | eventStore (JSONL)가 ground truth. 재시작 후 EventsAfterID로 따라잡기 |

---

## 11. Claude Code 참조 스펙

### Hook 시스템

| 항목 | 상세 | 소스 |
|------|------|------|
| Hook 이벤트 수 | 27개 (ham-agents는 26개 등록, Setup 포함 시 27개) | [Claude Code Hooks 문서](https://docs.anthropic.com/en/docs/claude-code/hooks) |
| Handler 유형 | command, http, prompt, agent (4가지) | 동일 |
| Hook output 상한 | 10,000자 | 동일 |
| HookInput 구조 | `{ session_id, transcript_path, conversation_id, ... }` + event-specific fields | 동일 |
| HookJSONOutput | `{ continue, decision, hookSpecificOutput }` | 동일 |

### Hook 이벤트별 payload 상세

| Hook | 핵심 데이터 필드 | ham-agents 캡처 상태 |
|------|-----------------|-------------------|
| PreToolUse | tool_name, tool_input (full JSON) | tool_name만 (tool_input preview) |
| PostToolUse | tool_name, tool_result | tool_name만 |
| PostToolUseFailure | tool_name, error_type, is_timeout, is_interrupt | generic error만 |
| SubagentStart | agent_type, session_id | agent_type |
| SubagentStop | agent_transcript_path, description | description만 |
| Notification | type, message | type만 |
| PermissionRequest | tool_name, tool_use_id | tool_name만 |
| PermissionDenied | tool_name, tool_use_id, description | tool_name만 |
| TeammateIdle | teammate_name, session_id | teammate_name |
| TaskCreated | task_id, task_description, assignee | task_description |
| TaskCompleted | task_id | task_id |
| SessionStart | session_id, source | session_id만 (source 미캡처) |
| SessionEnd | session_id, reason | session_id만 |
| Stop | (없음) | last_assistant_message |
| StopFailure | error_type | error_type |

### Agent Teams 관련 Hook

| Hook | 역할 | Phase 1 활용 |
|------|------|-------------|
| TeammateIdle | teammate가 작업 완료 후 대기 | P1-2 Session Graph에서 팀 그룹핑 |
| TaskCreated | lead가 task 생성 | P1-3 Inbox에서 task_complete 알림 |
| TaskCompleted | teammate가 task 완료 | P1-3 Inbox에서 task_complete 알림 |

---

## 12. 경쟁 제품 비교

### 경쟁 제품 비교 (2026-04 기준)

| 축 | Cursor | Windsurf | Warp | Claude Code Agent Teams | ham-agents |
|---|--------|----------|------|-------------------------|-----------|
| 제품 타입 | AI-first IDE | AI-first IDE | AI-native terminal | CLI 내장 teams | 터미널 무관 control plane |
| Claude Code 연동 | 없음 (자체 agent) | 없음 (Cascade) | 가능 (PTY) | 네이티브 | hook 네이티브 |
| 멀티세션 관찰 | Cloud Agents 대시보드 | 제한적 | Oz Cloud 에이전트 | 공유 task list | 트리 + 메뉴바 ambient |
| 로컬 우선 | ✗ (클라우드) | ✗ (클라우드) | 부분 | ✓ | ✓ |
| 승인/차단 흐름 | Bugbot autofix | Checkpoints | 없음 | mailbox 수동 | notification policy + approval (P1-3) |
| 터미널 종속 | IDE 내장 | IDE 내장 | Warp 전용 | CLI 전용 | iTerm/tmux/Warp/Ghostty |
| 가격 모델 | 유료 구독 | 유료 구독 | 무료 + Pro | Claude 구독 포함 | 오픈소스 |
| Direct / Govern capability | Background Agents 는 관찰 중심 + bugbot 자동 수정. 직접 제어는 IDE 에디터 context 에 묶임 | Cascade workflows 로 자동 실행, 중간 개입은 checkpoint 기반 | Oz Cloud Agent 직접 지시 가능하지만 Warp 터미널 전용 | lead 가 task 를 분배, worker 는 mailbox 수신, approval 은 수동 CLI | Studio 탭의 embedded PTY + Approval Interception 으로 **직접 제어 + 실시간 차단 둘 다 커버** |
| 출처 | [cursor.com/changelog](https://cursor.com/changelog) | [windsurf.com/cascade](https://windsurf.com/cascade) | [warp.dev/oz](https://www.warp.dev/oz) | [code.claude.com/docs/en/agent-teams](https://code.claude.com/docs/en/agent-teams) | — |

### ham-agents 차별점

1. **vs Cursor**: IDE 에 갇히지 않고 멀티 터미널 세션을 메뉴바 ambient 로 관찰. Cursor 는 "작업을 맡기는 도구", ham 은 "작업 중인 에이전트를 운영하는 도구"
2. **vs Windsurf**: Windsurf 가 편집 루프를 최적화한다면, ham 은 실행 루프를 최적화. 에이전트가 실제로 어떻게 행동했고 왜 그런 결정을 했는지 운영성과 재현성으로 차별화
3. **vs Warp**: Warp 전용이 아니라 iTerm/tmux/Warp/Ghostty 모두 지원. Warp 가 terminal replacement 라면, ham 은 terminal-agnostic mission control
4. **vs Claude Code Agent Teams**: CLI 내장 teams 를 대체하지 않고, 그 위에 관찰/정책 레이어 제공. hook 네이티브 연동으로 teams 의 task/mailbox 흐름을 외부에서 가시화

### Observability tools (참고)

기존 AgentOps/LangSmith/Helicone 계열은 ham-agents 의 주요 포지셔닝 축이 아니다. 이들은 범용 LLM observability 플랫폼으로, Claude Code hook 네이티브 연동이 없고 클라우드 전송이 필수다. **Observe-only, Direct/Govern 미제공**. 참고용으로만 유지한다.

| 기능 | ham-agents (Phase 1 목표) | AgentOps.ai | LangSmith | Helicone |
|------|--------------------------|-------------|-----------|----------|
| 세션 모니터링 | CLI + 메뉴바 트리 | 웹 대시보드 | 웹 대시보드 | 웹 대시보드 |
| 에이전트 관계 시각화 | parent-child 트리 (SubAgents) | span 트리 | trace 트리 | 없음 |
| 실시간성 | 200ms~5초 폴링 | WebSocket | 폴링 | 폴링 |
| 비용 추적 | 조사 중 (ADR-3) | API 레벨 자동 | API 레벨 자동 | 프록시 레벨 자동 |
| 알림/Inbox | 읽기 전용 Inbox | 웹훅 | 웹훅 | 웹훅 |
| 로컬/프라이버시 | 완전 로컬 (데이터 외부 전송 없음) | 클라우드 전송 | 클라우드 전송 | 프록시 경유 |
| Claude Code 전용 | O (hook 네이티브 연동) | X (범용) | X (범용) | X (범용) |
| macOS 네이티브 | O (메뉴바 앱) | X | X | X |

### 관련 도구 참고

| 도구 | 참고할 점 |
|------|----------|
| [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) | hook 시스템, `--print` 모드, MCP 통합 |
| [Warp Terminal](https://www.warp.dev/) | 터미널 내 AI 통합 UX. 명령 블록 단위 시각화 참고 |
| [Cursor](https://cursor.sh/) | 에디터 내 에이전트 세션 관리 UX 참고 |
| [Cline (VS Code)](https://github.com/cline/cline) | 에이전트 비용 추적 (API 프록시 방식) 참고 |

---

## Phase 2 Preview

Phase 1 Mission Control MVP 는 관찰 기반 foundation 을 다진다. Phase 2 에서는 ham Studio 윈도우가 primary UX 가 되고, 각 탭 안에 embedded PTY 로 Claude Code 가 직접 돌아간다. 기술 전환은 `tech-migration.md` ADR-2 (PTY Transport) 에서 정의하고, 기능은 `ham-studio.md` P2-1 (Embedded PTY Runtime), P2-2 (Session Launcher), P2-3 (Approval Interception) 에서 자세히 다룬다.

Phase 1 에서 쌓는 것들 중 Phase 2 와 직접 연결되는 항목:
- P1-1 SessionEvent 스키마 (ADR-1) — Phase 2 PTY 라인 tee 가 이 스키마로 흘러감
- P1-3 Notification Inbox — Phase 2 에서 approval modal 의 기반
- P1-0 Registry 락 + contract test — Phase 2 multi-tab 동시성의 foundation

---

## Rollback Plan

각 Phase 출시 후 실패 시 복구 전략.

### Phase 1 Rollback
- 증상 분류: (a) hamd 크래시 (b) events.jsonl 손상 (c) UI 행
- 복구 단계:
  1. `launchctl stop com.hamagents.hamd` 로 데몬 정지
  2. `~/.ham/state/managed-agents.json`, `~/.ham/state/events.jsonl` 백업
  3. 이전 바이너리 (`~/.ham/bin/hamd.previous`) 로 rollback
  4. `ham status` 로 헬스체크
- 데이터 손실 범위: events.jsonl 복구 불가 시, 세션 그래프는 재구축되지만 P1-3 inbox 항목은 소실 가능

### Phase 2 Rollback
- Studio 윈도우 크래시 → MenuBarExtra 재시작으로 복구. Studio 설정은 UserDefaults 에 저장되어 비파괴적
- webhook 서버 (P2-4) 오동작 → 설정에서 webhook 서버만 비활성화

### Phase 3 Rollback
- SQLite 전환 (P3-0) 실패 → events.jsonl 원본은 별도 보관, SQLite 는 버리고 JSONL 로 복귀
- Policy Engine 오알림 폭주 → policy YAML 를 빈 파일로 교체, hamd 재시작

---

## E2E Test Strategy

메뉴바 클릭 → IPC → hook → UI 갱신 전체 루프 검증.

### 테스트 시나리오
1. **Agent spawn loop**: `ham run ... --managed` → hook.session-start → 메뉴바 Agent 추가 → 메뉴바 클릭으로 agent 선택 → detail view 렌더 확인
2. **Tool execution loop**: agent 에서 tool 실행 → hook.tool-start → UI status "running_tool" → hook.tool-done → UI status "idle"
3. **Notification loop**: hook.notification 발생 → P1-3 inbox 에 아이템 추가 → 메뉴바 배지 증가 → 클릭으로 inbox 열람 → 읽음 처리
4. **Approval loop (Phase 2)**: hook.permission-request → 메뉴바 알림 → approve 액션 → hamd 로 전달 → Claude 에 응답 (Claude Code permission 시스템 경유)
5. **Crash recovery loop**: hamd kill → UI 는 disconnected 상태 표시 → hamd 재시작 → 이전 상태 복원 확인

### 도구
- Go: `go test ./go/internal/ipc -run TestE2E` (contract + golden)
- Swift: XCUITest 로 메뉴바 상호작용 (별도 테스트 타겟)
- 공통: 테스트용 mock hook 송신 스크립트

---

## 부록: 용어 정리

| 용어 | 설명 |
|------|------|
| hamd | ham-agents 백그라운드 데몬. Go로 구현. IPC 서버 + 상태 엔진 |
| hook | Claude Code가 특정 이벤트 발생 시 외부로 알리는 메커니즘. `ham hook <type>` CLI로 수신 |
| mutateAgent | hamd의 상태 변경 패턴. Lock → Load → Mutate → Save → Unlock |
| managed agent | `ham run`으로 직접 실행한 에이전트. PTY 관리 포함 |
| attached agent | `ham attach`로 기존 터미널 세션에 연결한 에이전트 |
| observed agent | `ham observe`로 트랜스크립트를 감시하는 에이전트 |
| FollowEvents | `events.follow` IPC 커맨드. 200ms 폴링, 60초 max wait의 long-polling |
| EventBus | P1-5에서 도입할 hamd 내부 pub-sub 시스템 |
| InboxManager | P1-3에서 도입할 알림 수신함 관리자 |
| ArtifactStore | P1-1에서 도입할 artifact 파일 저장소 |
| SessionGraph | P1-2에서 도입할 에이전트 트리 뷰 모델 |
