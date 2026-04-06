# AgentOps Platform 기능 명세서

> Phase 3 | 2026-04 작성 | ham-agents v3.0 Vision
>
> **이 문서는 설계 명세서이며 코드를 포함하지 않는다.**

---

## 목차

1. [개요](#1-개요)
2. [아키텍처 제약 요약](#2-아키텍처-제약-요약)
3. [기능 1: AI Agent Debugger](#3-기능-1-ai-agent-debugger)
4. [기능 2: Org Policy Engine](#4-기능-2-org-policy-engine)
5. [기능 3: Persistent Memory Graph](#5-기능-3-persistent-memory-graph)
6. [기능 4: Autonomous Maintenance](#6-기능-4-autonomous-maintenance)
7. [기능 5: Pack Marketplace](#7-기능-5-pack-marketplace)
8. [저장소 전략](#8-저장소-전략)
9. [Claude Code 통합 포인트](#9-claude-code-통합-포인트)
10. [경쟁 제품 참조](#10-경쟁-제품-참조)

---

## 1. 개요

Phase 3 AgentOps Platform은 ham을 관측 도구에서 **에이전트 운영 플랫폼**으로 전환하는 단계다. Phase 1(Mission Control MVP)의 이벤트 스키마 확장과 EventBus, Phase 2(Terminal IDE)의 Studio UI와 Playbook 인프라를 기반으로 구축한다.

**Phase 3의 핵심 전제:**
- Phase 1의 `core.Event` 확장 필드(SessionID, ToolName, ToolInput, ArtifactRef 등)가 안정적으로 수집되고 있어야 한다.
- Phase 1의 EventBus(fan-out subscriber 모델)가 동작해야 한다.
- Phase 2의 ham Studio UI가 존재해야 한다 (디버거, 정책 UI의 표면).
- Phase 2의 Playbook/Recipe 포맷이 정의되어 있어야 한다.

### Schema Dependency
Phase 3 기능들은 mission-control.md **ADR-1** 에 정의된 통합 SessionEvent 스키마를 전제한다.
Phase 1 에서 omitempty 로 미리 추가된 필드(Source, Confidence, ConfidenceModel, Cost, ApprovalState, Payload)를 Phase 3 에서 채워 사용한다.

> **라운드 3 업데이트 (2026-04-06)**: Phase 2 embedded PTY 도입으로 Alert Policy Engine (P3-2) 의 Realism Check 가 "tool blocking 불가" → "managed 탭에서 차단 가능" 으로 전환됐다. 다른 Phase 3 기능 (Event Timeline Viewer, Memory Graph, Pack Registry) 의 Realism Check 는 변경 없음.

---

## 왜 ham-agents 를 써야 하는가

### 시나리오 1: 운영 사고 사후 분석 (post-mortem)

야간에 무인 실행된 Claude Code 세션이 예상치 못한 파일을 삭제하고 종료됐다. 다음 날 아침 개발자는 `ham debug replay <session-id>` 를 실행한다. AI Agent Debugger 가 events.jsonl (Phase 3 에서 SQLite 로 전환됨) 에서 SessionTrace 를 재구성해 tool-call 체인을 시간순으로 보여준다. "왜 이 rm 명령을 실행했는가?" — 직전 assistant message, 해당 시점의 permission 상태, breakpoint 조건 매칭 여부가 모두 표시된다. 스크린 녹화나 별도 로그 없이도 에이전트의 의사결정 경로를 재현할 수 있다.

### 시나리오 2: 정책 기반 거버넌스 운영

엔지니어링 팀이 프로덕션 리포에 Claude Code 를 도입하려 한다. 관리자는 `.ham/policies/safety.yaml` 에 `deny: { tools: ["Bash"], input_pattern: "DROP TABLE|force push|rm -rf" }` 를 설정한다. 에이전트가 해당 패턴의 명령을 시도하면 Org Policy Engine 이 즉시 감지하고 Notification Inbox 에 위반 기록을 남긴다. hook 단방향 제약으로 실시간 차단은 불가하지만 (알림 전달까지 약 200ms), `~/.claude/settings.json` 의 `permissions.deny` 와 사전 연동하면 해당 도구 자체를 원천 차단할 수 있다. 감사 로그는 `ham policy violations` CLI 로 조회하며 컴플라이언스 보고에 활용된다.

### 시나리오 3: 세션 간 지식 누적 운영

같은 프로젝트를 반복적으로 작업하는 팀에서, 매 세션마다 에이전트에게 "이 리포는 `CGO_ENABLED=0` 빌드 필요", "DB 마이그레이션 실패 시 rollback 절차" 를 다시 설명해야 했다. Persistent Memory Graph 는 세션 종료 시 compact summary 와 반복 tool 패턴에서 지식을 자동 추출해 저장한다. 다음 세션 시작 시 관련 메모리가 프로젝트 CLAUDE.md 에 자동 추가되어 에이전트가 반복 실수를 줄인다. 팀원이 발견한 인사이트를 `ham memory promote` 로 팀 메모리로 승격하면 전체 팀이 공유한다. Phase 2 embedded PTY + Phase 3 Policy Engine 결합으로, 위 상황에서 팀장은 prod 접근 시도를 **차단** (실시간) 하거나 감사 로그에 자동 기록한다.

---

## 2. 아키텍처 제약 요약

Phase 3 설계 시 반드시 고려해야 하는 현재 아키텍처의 한계:

| 제약 | 설명 | 영향 |
|------|------|------|
| **IPC: Unix socket request-response** | 연결당 1회 JSON 요청-응답. 클라이언트 타임아웃 3초 | 디버거 실시간 스트리밍 불가. long-polling(200ms poll, 60s max)이 유일한 준실시간 수단 |
| **Hooks: 단방향 (Claude -> hamd)** | hook payload에 토큰/비용 없음. 파일 diff 없음. parent_id 없음 | 비용 추적 불가(hook 경유). 부모-자식 관계 직접 추론 필요 |
| **저장소: 파일 기반** | JSONL append-only (events.jsonl, 10K 상한). JSON snapshot (agents, settings, teams) | 세션 간 쿼리, 시간 범위 검색, 관계 그래프 탐색 불가. Phase 3에서 embedded DB 필수 |
| **에이전트 상태: 휘발성** | `managed-agents.json`은 현재 스냅샷만 유지. 과거 세션 상태 없음 | 세션 replay는 이벤트 로그 재구성에 의존 |
| **외부 승인 API 미확인** | Claude Code에 외부에서 permission을 승인하는 공개 API가 확인되지 않음 | 디버거 breakpoint의 "일시정지 후 재개"는 permission-request hook + 사용자 수동 승인으로 우회 |

---

## 3. 기능 1: AI Agent Debugger

### 3-1. 기능 설명 + 사용자 시나리오

AI Agent Debugger는 Claude Code 세션의 실행 과정을 **사후 분석(post-mortem)** 및 **준실시간 관찰**할 수 있는 도구다.

**시나리오 A - 세션 리플레이 (사후 분석):**
개발자가 CI 연동 세션이 실패한 원인을 파악하려 한다. `ham debug replay <session-id>`를 실행하면 해당 세션의 이벤트 타임라인이 재구성되어 tool-call 순서, 각 도구의 입출력, 상태 전이, permission 요청/거절 기록을 시간순으로 볼 수 있다.

**시나리오 B - Tool-call step-through:**
Studio UI에서 특정 세션을 열고 "step" 모드로 전환하면 각 tool invocation을 하나씩 탐색할 수 있다. 각 단계에서 도구명, 입력 프리뷰, 소요 시간, 결과(성공/실패/중단)를 확인한다.

**시나리오 C - Breakpoint (조건부 일시정지):**
정책 파일에 `breakpoint: tool_name == "Bash" && input contains "rm -rf"`를 설정하면, 해당 조건이 매칭될 때 에이전트가 `waiting_input` 상태에서 사용자에게 알림을 보낸다. 실제 "일시정지"는 Claude Code의 permission-request 메커니즘을 활용한다.

**시나리오 D - Compare runs:**
같은 Playbook을 두 번 실행한 결과를 비교한다. 이벤트 시퀀스, 사용된 도구, 소요 시간, 실패 지점을 나란히 diff로 표시한다.

**시나리오 E - "왜 이 명령을 실행했는가" 추적:**
특정 tool-call을 선택하면 직전의 assistant message(Phase 1에서 `LastAssistantMessage`로 저장), 해당 시점의 context(compact 이력), permission 상태를 보여준다.

### 3-2. 필요한 데이터

**현재 있는 것:**
| 데이터 | 소스 | 위치 |
|--------|------|------|
| 이벤트 로그 | `events.jsonl` | 최대 10K 엔트리. EventType 14종 |
| 도구 호출 기록 | `hook.tool-start` / `hook.tool-done` / `hook.tool-failed` | Event에 ToolName, ToolInputPreview 포함 (Phase 1 확장 후) |
| 에이전트 상태 전이 | `agent.status_updated` 이벤트 | Status, StatusConfidence, StatusReason |
| SubAgent 관계 | `hook.agent-spawned` / `hook.agent-finished` | SubAgentInfo (agent_id, type, status) |
| Permission 이력 | `hook.permission-request` / `hook.permission-denied` | ToolName + Description |
| Context compaction | `hook.pre-compact` / `hook.post-compact` | CompactTrigger, CompactSummary |
| 마지막 응답 | `hook.stop`, `hook.agent-finished` | LastMessage (LastAssistantMessage) |
| 트랜스크립트 | `~/.claude/projects/*/sessions/*/transcript` | Claude Code가 자체 저장하는 대화 기록 |

**새로 만들어야 하는 것:**
| 데이터 | 용도 | 설명 |
|--------|------|------|
| `SessionTrace` | 세션 리플레이 뷰 모델 | 이벤트 로그를 세션 단위로 그룹핑하고, tool-call 체인을 트리 구조로 재구성한 read model |
| `ToolCallNode` | Step-through 단위 | tool-start ~ tool-done 사이의 이벤트를 하나의 "단계"로 묶은 노드 |
| `BreakpointRule` | 조건부 일시정지 | CEL 또는 간단한 DSL로 표현한 조건식. 정책 파일(YAML)에 저장 |
| `SessionDiff` | Compare runs 결과 | 두 SessionTrace 간 정렬(alignment) 알고리즘 결과 |
| `TraceIndex` | 세션 검색/필터 | 세션 ID, 프로젝트, 시간 범위, 결과(성공/실패) 기준 인덱스. embedded DB 필요 |

### 3-3. Go 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `go/internal/core/debugger.go` | 신규 | `SessionTrace`, `ToolCallNode`, `BreakpointRule`, `SessionDiff` 타입 정의 |
| `go/internal/runtime/trace_builder.go` | 신규 | 이벤트 로그 -> SessionTrace 변환. 세션별 필터링, tool-call 체인 구성 |
| `go/internal/runtime/breakpoint.go` | 신규 | BreakpointRule 평가 엔진. EventBus subscriber로 동작. 조건 매칭 시 InboxItem 생성 |
| `go/internal/runtime/session_diff.go` | 신규 | 두 SessionTrace를 정렬하고 차이를 추출하는 diff 알고리즘 |
| `go/internal/store/trace_store.go` | 신규 | SessionTrace 영속화. 초기에는 파일 기반, 이후 embedded DB |
| `go/internal/ipc/server.go` | 수정 | 디버거 관련 dispatch 핸들러 추가 |
| `go/cmd/ham/cmd_debug.go` | 신규 | `ham debug replay`, `ham debug compare`, `ham debug breakpoint` CLI 명령 |

### 3-4. Swift 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `Sources/HamCore/DebuggerPayloads.swift` | 신규 | SessionTrace, ToolCallNode 디코딩 모델 |
| `Sources/HamCore/DaemonIPC.swift` | 수정 | 디버거 관련 DaemonCommand 케이스 추가 |
| `Sources/HamAppServices/DebuggerViewModel.swift` | 신규 | 리플레이 타임라인 ViewModel. step-through 상태 관리 |
| `Sources/HamAppServices/SessionDiffViewModel.swift` | 신규 | Compare runs ViewModel |
| `apps/macos/HamMenuBarApp/Sources/DebuggerViews.swift` | 신규 | Studio 내 디버거 탭 (타임라인 뷰, step-through 뷰, diff 뷰) |

### 3-5. IPC 변경사항

| 커맨드 | Request 필드 | Response 필드 | 설명 |
|--------|-------------|---------------|------|
| `debug.trace` | `session_id` | `SessionTrace` | 세션 트레이스 조회 |
| `debug.trace.list` | `project_path`, `limit`, `offset` | `[]SessionTraceSummary` | 트레이스 목록 (요약) |
| `debug.compare` | `session_id_a`, `session_id_b` | `SessionDiff` | 두 세션 비교 |
| `debug.breakpoint.list` | - | `[]BreakpointRule` | 활성 breakpoint 목록 |
| `debug.breakpoint.set` | `BreakpointRule` | `BreakpointRule` (with ID) | breakpoint 설정 |
| `debug.breakpoint.remove` | `breakpoint_id` | - | breakpoint 삭제 |

### 3-6. 선행 작업 / 의존성

| 의존성 | Phase | 설명 |
|--------|-------|------|
| P1-1 이벤트 스키마 확장 | Phase 1 | SessionID, ToolName, ToolInput, ArtifactRef 등이 Event에 포함되어야 함 |
| P1-5 EventBus | Phase 1 | Breakpoint 엔진이 EventBus subscriber로 동작 |
| P1-3 Notification Inbox | Phase 1 | Breakpoint 트리거 시 InboxItem으로 사용자에게 알림 |
| Phase 2 ham Studio | Phase 2 | 디버거 UI의 호스트 |
| Phase 2 Playbook | Phase 2 | Compare runs에서 "같은 Playbook의 두 실행"을 비교 |

### 3-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 원인 | 대안 |
|-------------|------|------|
| **실시간 step-by-step 디버깅** | Claude Code에 외부 일시정지/재개 API 없음 | 사후 분석(리플레이) 중심으로 설계. "준실시간"은 EventBus long-polling으로 200ms 지연 허용 |
| **진짜 breakpoint (실행 중지)** | hook은 단방향. hamd에서 Claude Code 실행을 멈출 수 없음 | permission-request를 활용한 간접 일시정지: Claude Code의 allowlist에서 특정 도구를 제거하면 permission 요청이 발생하여 사실상 일시정지됨. 또는 breakpoint 조건 매칭 시 알림만 발송하고, 사용자가 터미널에서 수동 개입 |
| **tool-call의 전체 입출력 캡처** | hook payload의 tool_input은 map[string]interface{}이지만, tool output은 전달되지 않음 | Phase 1의 ArtifactRef로 tool_input만 저장. tool output은 트랜스크립트 파일(`~/.claude/projects/*/sessions/*/transcript`)을 파싱하여 보강. 트랜스크립트 포맷이 비공개이므로 베스트 에포트 |
| **parent-child 관계 완전 추적** | hook payload에 parent_id 없음 | `hook.agent-spawned`의 agent_id(부모) + description에서 자식 ID를 추론. SubAgentInfo를 활용하되 완전하지 않을 수 있음 |
| **10K 이벤트 초과 세션 분석** | events.jsonl 10K 상한에서 오래된 이벤트 삭제됨 | Phase 3 진입 전 embedded DB 전환 필수. 전환 전까지는 trace 생성 시점에 SessionTrace를 별도 파일로 스냅샷 |
| **비용/토큰 상관 분석** | hook에 토큰/비용 정보 없음 | Phase 1 ADR-3 조사 결과에 따라 결정. 트랜스크립트 파싱 또는 Anthropic API 대시보드 연동으로 우회 가능성 탐색 |

#### Realism Check

| 필요 데이터 / 기능 | 현재 확보 경로 | 실현성 |
|-------------------|---------------|--------|
| tool_name | hook.tool-start payload | ✓ |
| tool_input (전체 JSON) | hook payload에 있으나 크기 제한 있음 → 4KB 이하만 ToolInput 저장, 초과분은 ArtifactStore | ⚠ 부분 확보 |
| tool_output / tool_result | hook.tool-done 에 없음 → transcript 파싱 필요 (포맷 비공개) | ⚠ 베스트 에포트 |
| token / cost 정보 | hook payload 에 없음 → ADR-3 조사 결과 필요 | ✗ 확보 불가 |
| breakpoint 실시간 차단 | hook 단방향 → hamd 에서 Claude Code 실행 중지 불가 | ✗ (permission allowlist 제거로 간접 우회만 가능) |
| parent-child 관계 | hook payload 에 parent_id 없음 → hamd 내부 추론에 의존 | ⚠ 불완전 |
| 10K 이벤트 초과 세션 분석 | events.jsonl 10K 상한 → 오래된 이벤트 삭제됨 | ✗ Phase 3 SQLite 전환 전까지 불가 |

**결론**: AI Agent Debugger 는 "세션 리플레이 + tool-call step-through + 의사결정 맥락 표시" 는 구현 가능하다. 그러나 "실시간 breakpoint (실행 중지)" 와 "비용 상관 분석" 은 hook 제약으로 이름값을 못한다. 기능명을 **"Event Timeline Viewer (Best-effort, hook 제약 있음)"** 으로 대안 명칭을 제안하며, breakpoint 는 "조건 매칭 시 알림 발송" 수준으로 축소 범위를 명확히 한다.

### 3-9. Realistic MVP 스코프

**MVP (빌드 가능):**
- 세션 리플레이: 이벤트 로그에서 SessionTrace 구성, CLI로 타임라인 출력
- Tool-call step-through: CLI에서 화살표 키로 이전/다음 단계 이동
- "왜 이 명령을?" 표시: tool-call 직전의 LastAssistantMessage 연결

**MVP 이후 (Phase 2 Studio 필요):**
- Studio UI 내 시각적 타임라인
- Compare runs (두 세션 diff)
- Breakpoint (EventBus + 정책 파일)

---

## 4. 기능 2: Org Policy Engine

### 4-1. 기능 설명 + 사용자 시나리오

Org Policy Engine은 리포/팀/환경별로 에이전트의 행동 범위를 정의하는 **선언적 거버넌스 시스템**이다.

**시나리오 A - 위험 명령 차단:**
`.ham/policies/safety.yaml`에 `deny: { tools: ["Bash"], input_pattern: "rm -rf|DROP TABLE|force push" }`를 설정하면, 해당 패턴이 감지될 때 사용자에게 경고 알림을 보낸다.

**시나리오 B - 네트워크 접근 제어:**
정책에 `network: { allow: ["github.com", "*.internal.corp"], deny_all_others: true }`를 설정하면, WebFetch/WebSearch 도구 사용 시 허용 도메인만 통과하도록 경고한다.

**시나리오 C - 배포 경로 제한:**
`deploy: { allowed_branches: ["main", "release/*"], require_approval: true }`로 설정하면, CI 관련 Bash 명령에서 금지된 브랜치로의 배포 시도를 감지하고 차단 알림을 보낸다.

**시나리오 D - Secret 사용 정책:**
환경변수나 파일에서 API 키 패턴이 감지되면 경고를 발생시킨다. `.env`, `credentials.json` 등의 파일에 Write 도구로 접근할 때 특히 주의한다.

**시나리오 E - 팀/리포별 정책 상속:**
전사 정책(`~/.ham/policies/org.yaml`) -> 팀 정책(`~/.ham/policies/team-backend.yaml`) -> 리포 정책(`.ham/policies/repo.yaml`) 순으로 상속. 하위 정책은 상위 정책을 강화(더 제한적)만 할 수 있고 완화할 수 없다.

### 4-2. 필요한 데이터

**현재 있는 것:**
| 데이터 | 소스 | 활용 |
|--------|------|------|
| tool_name | `hook.tool-start`, `hook.permission-request` | 도구별 정책 매칭 |
| tool_input (map) | hook payload의 `ToolInputPreview` | 입력 패턴 매칭 (부분 텍스트만 가용) |
| project_path | Agent.ProjectPath | 리포별 정책 해석 |
| session_id | Agent.SessionID | 세션 단위 정책 적용 추적 |
| file_path | `hook.file-changed` | 파일 접근 정책 매칭 |
| cwd | `hook.cwd-changed` | 작업 디렉토리 기반 정책 |

**새로 만들어야 하는 것:**
| 데이터 | 용도 | 설명 |
|--------|------|------|
| `PolicySet` | 정책 정의 | YAML 기반 정책 파일. 도구 제한, 패턴 거부, 네트워크 제어, 파일 접근 제어 규칙의 집합 |
| `PolicyViolation` | 위반 기록 | 정책 위반이 감지된 시점, 에이전트, 도구, 입력, 적용된 규칙을 기록 |
| `PolicyAuditLog` | 감사 추적 | 정책 평가 이력 (허용/거부/경고). 컴플라이언스 보고에 사용 |

### 4-3. Go 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `go/internal/core/policy.go` | 신규 | `PolicySet`, `PolicyRule`, `PolicyViolation`, `PolicyScope` (org/team/repo) 타입 정의 |
| `go/internal/runtime/policy_engine.go` | 신규 | 정책 로딩 (YAML 파싱), 규칙 평가, 위반 감지. EventBus subscriber로 동작 |
| `go/internal/runtime/policy_loader.go` | 신규 | 정책 파일 탐색 (org -> team -> repo), 상속 병합 로직 |
| `go/internal/store/policy_store.go` | 신규 | PolicyViolation, PolicyAuditLog 영속화 |
| `go/internal/ipc/server.go` | 수정 | 정책 관련 dispatch 핸들러 추가 |
| `go/cmd/ham/cmd_policy.go` | 신규 | `ham policy list`, `ham policy check`, `ham policy violations` CLI 명령 |

### 4-4. Swift 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `Sources/HamCore/PolicyPayloads.swift` | 신규 | PolicySet, PolicyViolation 디코딩 모델 |
| `Sources/HamCore/DaemonIPC.swift` | 수정 | 정책 관련 DaemonCommand 케이스 추가 |
| `Sources/HamAppServices/PolicyViewModel.swift` | 신규 | 활성 정책 목록, 최근 위반 기록 ViewModel |
| `apps/macos/HamMenuBarApp/Sources/PolicyViews.swift` | 신규 | Studio 내 정책 탭 (활성 규칙, 위반 타임라인, 감사 로그) |

### 4-5. IPC 변경사항

| 커맨드 | Request 필드 | Response 필드 | 설명 |
|--------|-------------|---------------|------|
| `policy.list` | `project_path` (optional) | `[]PolicySet` | 적용 중인 정책 목록 (상속 병합 후) |
| `policy.violations` | `agent_id`, `limit`, `offset` | `[]PolicyViolation` | 정책 위반 이력 |
| `policy.check` | `tool_name`, `tool_input`, `project_path` | `PolicyCheckResult` | 특정 도구 호출에 대한 정책 평가 (dry-run) |
| `policy.audit` | `project_path`, `since`, `limit` | `[]PolicyAuditEntry` | 감사 로그 조회 |

### 4-6. 선행 작업 / 의존성

#### P3-2 Policy Secrets

- **저장 위치**: `~/.ham/policies/secrets.age` (age 암호화), 암호화 키는 keychain
- **접근 제어**: hamd 프로세스만 읽기 가능, ham CLI 는 policy dry-run 시에만 마스킹된 값 노출
- **감사 로그**: secret 사용 시마다 events.jsonl (또는 Phase 3 SQLite) 에 access audit 기록
- **회전**: 수동 회전만 지원 (자동 회전은 scope out)

| 의존성 | Phase | 설명 |
|--------|-------|------|
| P1-1 이벤트 스키마 확장 | Phase 1 | ToolName, ToolInput이 Event에 포함되어야 정책 평가 가능 |
| P1-5 EventBus | Phase 1 | Policy Engine이 EventBus subscriber로 실시간 평가 수행 |
| P1-3 Notification Inbox | Phase 1 | 정책 위반 시 InboxItem으로 알림 전달 |
| Phase 2 Playbook | Phase 2 | Playbook에 정책 참조를 포함하여 "이 플레이북은 이 정책을 준수한다" 선언 가능 |

### 4-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 원인 | 대안 |
|-------------|------|------|
| **실시간 차단 (block before execution)** | hook은 단방향. hamd가 Claude Code의 도구 실행을 사전 차단할 수 없음 | **감지 + 알림** 모델로 전환. 위반 감지 시 사용자에게 즉시 알림을 보내고, 사용자가 터미널에서 Ctrl+C로 중단. Claude Code의 `~/.claude/settings.json`의 `permissions.deny` 필드를 사전에 설정하여 특정 도구를 원천 차단하는 것은 가능 |
| **네트워크 접근 실시간 제어** | ham은 네트워크 프록시가 아님. WebFetch의 URL은 hook payload에서 확인 가능하지만 실행 후 감지됨 | `tool_input`의 URL 패턴 매칭으로 사후 감지. macOS 방화벽 규칙 자동 생성은 보안 위험이 크므로 제외 |
| **tool_input 전체 접근** | hook의 `ToolInputPreview`는 축약된 텍스트일 수 있음 | Phase 1 ArtifactStore의 인라인(4KB) 또는 파일(1MB) 저장분을 참조. 트랜스크립트에서 전체 입력 복원 시도 |
| **조직 단위 중앙 집중 정책** | ham은 로컬 머신 데몬. 중앙 서버 없음 | 정책 파일을 git 리포에 커밋하여 배포. `.ham/policies/` 디렉토리를 convention으로 정의. 향후 ham cloud 서비스 구축 시 중앙 관리 추가 |

#### Realism Check

| 필요 데이터 / 기능 | 현재 확보 경로 | 실현성 |
|-------------------|---------------|--------|
| tool_name | hook.tool-start, hook.permission-request | ✓ |
| tool_input 패턴 매칭 | hook payload ToolInputPreview (축약본) | ⚠ 전체 입력이 아닌 preview 만 가용 |
| tool blocking / real-time intercept | Phase 2 Embedded PTY + Approval Interception 에서 차단 가능 (ham-studio.md P2-3 참조) | ✓ Phase 2 Embedded PTY + Approval Interception 에서 차단 가능 (ham-studio.md P2-3 참조, **Phase 2 P2-1 spike 통과 조건부**) |
| 네트워크 접근 실시간 제어 | ham 은 네트워크 프록시가 아님 | ✗ 사후 감지만 가능 |
| 정책 상속 (org → team → repo) | 정책 파일 계층 탐색 필요 — 직접 구현 가능 | ✓ |
| 중앙 집중 정책 배포 | ham 은 로컬 데몬, 중앙 서버 없음 | ✗ git 기반 배포로만 가능 |

> **라운드 3 업데이트**: Phase 2 embedded PTY 층 도입으로 Policy Engine 이 실제 차단 기능을 갖게 됐다. 단 attached / observed legacy 모드에서는 여전히 알림만 가능 — 차단은 managed + Embedded PTY 탭에서만 동작.

**결론**: Org Policy Engine 의 핵심 제약은 **hook 단방향으로 인해 차단(block)은 불가능하고 알림(notify)만 가능**하다는 점이다. "Policy Engine" 이라는 이름은 "정책으로 에이전트를 제어한다"는 기대를 심지만, 실제 동작은 "정책 위반을 감지하고 사용자에게 알린다" 수준이다. 이를 명확히 하기 위해 기능 설명에 **(Managed PTY 탭에서 차단 가능, legacy 모드에서는 알림만)** 태그를 부착한다. `~/.claude/settings.json`의 `permissions.deny` 사전 연동으로 특정 도구 원천 차단은 별도로 가능하다.

### 4-9. Realistic MVP 스코프

**MVP (빌드 가능):**
- YAML 기반 정책 파일 로딩 (`.ham/policies/*.yaml`)
- tool_name + input 패턴 매칭으로 위반 감지
- 위반 시 Notification Inbox 알림 + CLI 경고 출력
- `ham policy list` / `ham policy violations` CLI

**MVP 이후:**
- 정책 상속 (org -> team -> repo)
- Studio UI 정책 관리 화면
- Claude Code `permissions.deny` 자동 동기화
- 감사 로그 + 컴플라이언스 리포트

#### Phase 2 Handoff

Phase 3 Policy Engine 은 Phase 2 Approval Interception (ham-studio.md P2-3) 을 extension point 로 사용한다. 구체적으로:

1. Phase 2 에서 `CommandAnswerPermission(agent_id, request_id, approved)` IPC 가 도입됨 (ADR-2)
2. Phase 3 Policy Engine 은 동일 포인트에 hook 을 달아서 YAML 룰을 평가하고 자동 응답 가능
3. 룰 매칭 실패 / timeout 시 fallback 은 Phase 2 수동 approval modal
4. Policy rule 예시:
   ```yaml
   rules:
     - match:
         tool_name: Bash
         command_regex: '^rm -rf /'
       action: deny
       reason: "dangerous command blocked by policy"
     - match:
         tool_name: Write
         path_regex: '^\.env'
       action: require_manual_approval
   ```
5. Phase 2 는 Policy Engine 없이도 수동 approval 로 충분히 동작. Phase 3 은 이를 정책 자동화로 확장하는 레이어.

---

## 5. 기능 3: Persistent Memory Graph

### 5-1. 기능 설명 + 사용자 시나리오

Persistent Memory Graph는 세션을 넘어 누적되는 **프로젝트/조직 수준의 지식 저장소**다.

**시나리오 A - 리포 관례 자동 수집:**
Claude Code가 특정 리포에서 반복적으로 사용하는 패턴(테스트 실행 명령, 린트 설정, 배포 절차)을 자동으로 감지하여 메모리에 저장한다. 다음 세션에서 같은 리포를 열면 이 관례가 자동으로 컨텍스트에 주입된다.

**시나리오 B - 장애 해결 기록:**
"DB 마이그레이션 실패 시 `rails db:rollback STEP=1` 후 재시도"와 같은 해결 패턴을 저장한다. 유사한 에러가 발생하면 과거 해결 기록을 에이전트 컨텍스트에 제안한다.

**시나리오 C - 빌드/디버그 힌트:**
"이 프로젝트는 `GOOS=darwin go build`로 빌드하면 CGO 관련 에러 발생. `CGO_ENABLED=0`을 추가해야 함"과 같은 힌트를 저장한다.

**시나리오 D - 개인 -> 조직 지식 전환:**
개인 메모리에서 3회 이상 참조된 항목을 팀 메모리로 승격 제안한다. 팀 리드가 승인하면 `.ham/memory/team/` 디렉토리에 저장되어 팀 전체가 공유한다.

**시나리오 E - 세션 간 지식 공유:**
같은 프로젝트의 다른 세션에서 발견한 인사이트가 현재 세션에 자동 주입된다. Claude Code의 `CLAUDE.md`와 유사하지만, 자동 수집/구조화/검색이 가능하다.

### 5-2. 필요한 데이터

**현재 있는 것:**
| 데이터 | 소스 | 활용 |
|--------|------|------|
| project_path | Agent.ProjectPath | 리포별 메모리 스코핑 |
| tool 사용 패턴 | Event 로그의 ToolName 빈도 | 리포 관례 추론 |
| 에러 + 해결 기록 | `hook.stop-failure` + 이후 성공 세션 | 장애 해결 패턴 추출 |
| compact summary | `hook.post-compact` CompactSummary | 세션 요약 (메모리 후보) |
| session transcript | `~/.claude/projects/*/sessions/*/transcript` | 상세 대화 기록 |
| CLAUDE.md | `~/.claude/CLAUDE.md`, 프로젝트별 CLAUDE.md | 기존 수동 메모리 |

**새로 만들어야 하는 것:**
| 데이터 | 용도 | 설명 |
|--------|------|------|
| `MemoryNode` | 지식 단위 | 제목, 내용, 태그, 소스(세션 ID), 신뢰도, 참조 횟수, 스코프(personal/team/org) |
| `MemoryEdge` | 지식 간 관계 | "relates_to", "supersedes", "contradicts" 관계 |
| `MemoryIndex` | 검색 인덱스 | 키워드, 태그, 프로젝트, 시간 기반 검색. 전문 검색 필요 시 embedded DB 필수 |
| `MemoryScope` | 스코프 정의 | personal(`~/.ham/memory/`), team(`.ham/memory/team/`), org(`.ham/memory/org/`) |

### 5-3. Go 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `go/internal/core/memory.go` | 신규 | `MemoryNode`, `MemoryEdge`, `MemoryScope` 타입 정의 |
| `go/internal/runtime/memory_collector.go` | 신규 | EventBus subscriber. 세션 종료 시 compact summary, 에러 패턴, 도구 사용 빈도에서 메모리 후보 추출 |
| `go/internal/runtime/memory_graph.go` | 신규 | 그래프 구성, 관계 추론, 검색, 스코프별 접근 제어 |
| `go/internal/runtime/memory_injector.go` | 신규 | 세션 시작 시 관련 메모리를 Claude Code의 system prompt에 주입하는 메커니즘 |
| `go/internal/store/memory_store.go` | 신규 | MemoryNode/Edge 영속화. 초기에는 JSON 파일, 이후 embedded DB |
| `go/cmd/ham/cmd_memory.go` | 신규 | `ham memory list`, `ham memory add`, `ham memory search`, `ham memory promote` CLI 명령 |

### 5-4. Swift 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `Sources/HamCore/MemoryPayloads.swift` | 신규 | MemoryNode 디코딩 모델 |
| `Sources/HamCore/DaemonIPC.swift` | 수정 | 메모리 관련 DaemonCommand 추가 |
| `Sources/HamAppServices/MemoryViewModel.swift` | 신규 | 메모리 그래프 탐색 ViewModel |
| `apps/macos/HamMenuBarApp/Sources/MemoryViews.swift` | 신규 | Studio 내 메모리 탭 (그래프 시각화, 검색, 승격 UI) |

### 5-5. IPC 변경사항

| 커맨드 | Request 필드 | Response 필드 | 설명 |
|--------|-------------|---------------|------|
| `memory.list` | `scope`, `project_path`, `tags`, `limit` | `[]MemoryNode` | 메모리 목록 |
| `memory.search` | `query`, `scope`, `project_path` | `[]MemoryNode` | 키워드 검색 |
| `memory.add` | `MemoryNode` | `MemoryNode` (with ID) | 수동 메모리 추가 |
| `memory.promote` | `node_id`, `target_scope` | `MemoryNode` | personal -> team/org 승격 |
| `memory.relate` | `source_id`, `target_id`, `relation` | `MemoryEdge` | 관계 설정 |
| `memory.inject` | `session_id`, `project_path` | `[]MemoryNode` | 세션에 주입할 메모리 조회 |

### 5-6. 선행 작업 / 의존성

| 의존성 | Phase | 설명 |
|--------|-------|------|
| P1-1 이벤트 스키마 확장 | Phase 1 | SessionID로 세션 단위 메모리 수집 |
| P1-5 EventBus | Phase 1 | memory_collector가 EventBus subscriber로 동작 |
| Phase 2 Playbook | Phase 2 | Playbook 실행 결과에서 메모리 자동 추출 |
| Phase 2 ham Studio | Phase 2 | 메모리 그래프 시각화 UI |
| Embedded DB (저장소 전략) | Phase 3 전제 | 전문 검색, 관계 그래프 탐색에 embedded DB 필요 |

### 5-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 원인 | 대안 |
|-------------|------|------|
| **자동 컨텍스트 주입 (system prompt)** | Claude Code에 외부에서 system prompt를 수정하는 API 없음 | **CLAUDE.md 연동**: 메모리에서 관련 항목을 추출하여 프로젝트별 CLAUDE.md에 자동 추가. Claude Code는 CLAUDE.md를 매 세션 시작 시 읽으므로 간접 주입 가능. 또는 Claude Code skills를 활용하여 `SKILL.md`로 메모리를 주입 |
| **세션 중 실시간 메모리 업데이트** | hook은 단방향. 실행 중인 세션에 새 메모리를 주입할 수 없음 | 세션 종료 후 다음 세션에 반영. 실행 중인 세션에는 `hook.instructions-loaded` 이벤트를 감지하여 CLAUDE.md가 다시 읽히는 시점에 맞춤 |
| **의미적 유사도 검색** | 로컬에서 임베딩 모델 실행은 리소스 부담 | 키워드 + 태그 기반 검색으로 시작. 향후 로컬 임베딩(onnxruntime 등) 또는 Claude API를 이용한 시맨틱 검색 추가 |
| **조직 단위 중앙 메모리** | ham은 로컬 데몬. 중앙 서버 없음 | git 리포에 `.ham/memory/team/`, `.ham/memory/org/` 디렉토리를 커밋하여 공유. 이 파일들을 ham이 로딩 |

#### Realism Check

| 필요 데이터 / 기능 | 현재 확보 경로 | 실현성 |
|-------------------|---------------|--------|
| 세션 종료 시 compact summary | hook.post-compact CompactSummary 필드 | ✓ |
| tool 사용 빈도 패턴 | Event 로그 ToolName 집계 | ✓ |
| 에러 + 해결 패턴 추출 | hook.stop-failure + 이후 성공 세션 상관 | ⚠ 상관 추론, 완전하지 않음 |
| 세션 중 실시간 메모리 주입 | hook 단방향 → 실행 중 system prompt 수정 불가 | ✗ 다음 세션에만 반영 가능 |
| 자동 컨텍스트 주입 (system prompt) | Claude Code 외부 API 없음 | ✗ CLAUDE.md 연동으로만 간접 가능 |
| 의미적 유사도 검색 | 로컬 임베딩 모델 없음 | ✗ 키워드+태그 검색으로 시작 |
| 조직 단위 중앙 메모리 | ham 로컬 데몬, 중앙 서버 없음 | ✗ git 기반 배포만 가능 |

**결론**: Persistent Memory Graph 는 "수동 메모리 CRUD + 프로젝트별 스코핑 + CLAUDE.md 연동" 은 구현 가능하다. "그래프" 라는 이름이 주는 관계형 지식 저장소 이미지와 달리, Phase 3 MVP 는 **태그 기반 플랫 스토어**에 가깝다. 실시간 주입과 의미 검색은 hook 제약과 로컬 리소스 한계로 MVP 범위 밖이다. 기능명 뒤에 **(Best-effort, hook 제약 있음 — 다음 세션 반영)** 태그를 부착한다.

### 5-9. Realistic MVP 스코프

**MVP (빌드 가능):**
- 수동 메모리 추가/검색 (`ham memory add`, `ham memory search`)
- 프로젝트별 메모리 스코핑 (project_path 기반)
- 세션 종료 시 compact summary를 메모리 후보로 자동 저장
- CLAUDE.md에 관련 메모리 항목 자동 추가 (opt-in)

**MVP 이후:**
- 자동 메모리 수집 (에러 패턴, 도구 사용 빈도)
- 그래프 관계 (MemoryEdge)
- personal -> team 승격 워크플로
- Studio UI 그래프 시각화

---

## 6. 기능 4: Autonomous Maintenance

### 6-1. 기능 설명 + 사용자 시나리오

Autonomous Maintenance는 사람이 없는 시간에 안전하게 반복 유지보수 작업을 수행하는 시스템이다. Claude Code의 **Scheduled Tasks**와 **Desktop/Cloud 세션**을 활용한다.

**시나리오 A - Nightly dependency sweep:**
매일 새벽 2시에 `npm audit` / `go mod tidy` / `pip check`를 실행하고, 취약점이 발견되면 자동으로 업데이트 PR을 생성한다. 결과를 Notification Inbox에 보고한다.

**시나리오 B - Flaky test triage:**
CI에서 최근 7일간 3회 이상 실패한 테스트를 식별하고, 실패 패턴을 분석하여 "환경 의존", "타이밍 이슈", "실제 버그" 등으로 분류한다. 분류 결과를 이슈로 생성하거나 Memory Graph에 저장한다.

**시나리오 C - Docs drift fix:**
코드 변경 커밋과 문서 파일의 마지막 수정일을 비교하여, 코드는 변경되었지만 관련 문서는 업데이트되지 않은 경우를 감지한다. 자동으로 문서 업데이트 PR을 생성한다.

**시나리오 D - PR babysitting:**
오픈된 PR의 CI 상태를 모니터링한다. CI 실패 시 로그를 분석하여 수정 방안을 제안하고, 리뷰어 코멘트에 자동으로 응답하거나 요청된 변경사항을 반영한다.

**시나리오 E - Release shepherding:**
릴리즈 브랜치 생성부터 체인지로그 작성, 태그 생성, 배포 확인까지의 전체 릴리즈 프로세스를 자동화한다. 각 단계 완료 시 사용자에게 확인을 요청한다.

### 6-2. 필요한 데이터

**현재 있는 것:**
| 데이터 | 소스 | 활용 |
|--------|------|------|
| project_path | Agent.ProjectPath | 프로젝트별 유지보수 대상 식별 |
| Playbook (Phase 2) | `.ham/playbooks/*.yaml` | 유지보수 작업 정의 |

**새로 만들어야 하는 것:**
| 데이터 | 용도 | 설명 |
|--------|------|------|
| `MaintenanceJob` | 작업 정의 | 작업 유형, 일정(cron), 대상 프로젝트, 실행 조건, 결과 보고 채널 |
| `MaintenanceRun` | 실행 기록 | 작업 ID, 시작/종료 시간, 결과(성공/실패/건너뜀), 생성된 PR/이슈, 비용 |
| `MaintenanceSchedule` | 스케줄 관리 | Claude Code Scheduled Tasks와의 매핑. cron 표현식, 활성/비활성 |

### 6-3. Go 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `go/internal/core/maintenance.go` | 신규 | `MaintenanceJob`, `MaintenanceRun`, `MaintenanceSchedule` 타입 정의 |
| `go/internal/runtime/maintenance_scheduler.go` | 신규 | cron 스케줄러. Claude Code Scheduled Tasks API(CronCreate/CronList/CronDelete)를 호출하여 작업 등록 |
| `go/internal/runtime/maintenance_runner.go` | 신규 | 작업 실행 관리. Claude Code 세션 생성(Desktop/Cloud), Playbook 주입, 결과 수집 |
| `go/internal/runtime/maintenance_reporter.go` | 신규 | 실행 결과를 Notification Inbox + Memory Graph에 보고 |
| `go/internal/store/maintenance_store.go` | 신규 | MaintenanceJob/Run 영속화 |
| `go/cmd/ham/cmd_maintenance.go` | 신규 | `ham maintain list`, `ham maintain run`, `ham maintain schedule`, `ham maintain history` CLI 명령 |

### 6-4. Swift 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `Sources/HamCore/MaintenancePayloads.swift` | 신규 | MaintenanceJob, MaintenanceRun 디코딩 모델 |
| `Sources/HamCore/DaemonIPC.swift` | 수정 | 유지보수 관련 DaemonCommand 추가 |
| `Sources/HamAppServices/MaintenanceViewModel.swift` | 신규 | 스케줄 관리, 실행 이력 ViewModel |
| `apps/macos/HamMenuBarApp/Sources/MaintenanceViews.swift` | 신규 | Studio 내 유지보수 탭 (스케줄 목록, 실행 이력, 결과 뷰) |

### 6-5. IPC 변경사항

| 커맨드 | Request 필드 | Response 필드 | 설명 |
|--------|-------------|---------------|------|
| `maintenance.list` | `project_path` | `[]MaintenanceJob` | 등록된 작업 목록 |
| `maintenance.schedule` | `MaintenanceJob` | `MaintenanceJob` (with ID) | 작업 등록/수정 |
| `maintenance.run` | `job_id` | `MaintenanceRun` | 수동 즉시 실행 |
| `maintenance.history` | `job_id`, `limit` | `[]MaintenanceRun` | 실행 이력 |
| `maintenance.cancel` | `run_id` | - | 실행 중인 작업 취소 |

### 6-6. 선행 작업 / 의존성

| 의존성 | Phase | 설명 |
|--------|-------|------|
| Phase 2 Playbook | Phase 2 | 유지보수 작업을 Playbook으로 정의 |
| Phase 2 Git/CI/Issue 연동 | Phase 2 | PR 생성, CI 모니터링, 이슈 생성에 필요 |
| P1-3 Notification Inbox | Phase 1 | 유지보수 결과 알림 |
| Persistent Memory Graph | Phase 3 | 유지보수 결과를 조직 지식으로 축적 |
| Claude Code Scheduled Tasks | 외부 | CronCreate/CronList/CronDelete 도구. 세션당 최대 50개 제한 |
| Claude Code Desktop/Cloud 세션 | 외부 | 무인 실행 환경 |

### 6-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 원인 | 대안 |
|-------------|------|------|
| **hamd에서 직접 Claude Code 세션 생성** | Claude Code 프로세스를 외부에서 프로그래밍적으로 시작하려면 server mode(`--spawn`) 필요. OAuth 인증 필요 | **Claude Code Scheduled Tasks 활용**: `/loop` (session), Desktop(persistent), Cloud(remote) scheduled tasks를 ham이 CronCreate 도구를 통해 등록. ham은 스케줄 매니저 역할만 수행 |
| **작업 중간 결과 실시간 모니터링** | Scheduled Task가 별도 세션에서 실행됨. hook 기반 이벤트 수신에 의존 | hook.session-start로 새 세션 감지 -> 일반 에이전트처럼 이벤트 스트림 수신. 별도 세션이므로 약간의 지연 허용 |
| **무인 실행 시 안전 보장** | 정책 위반 시 실시간 차단 불가 (hook 단방향) | **보수적 기본값**: 유지보수 작업에는 가장 제한적인 정책 자동 적용. write 권한 최소화. PR 생성은 하되 auto-merge는 금지. Org Policy Engine과 연동하여 유지보수 전용 정책 프리셋 제공 |
| **CI 로그 직접 접근** | ham은 CI 시스템에 대한 인증 정보가 없음 | Claude Code의 MCP 커넥터(GitHub, Linear 등) 활용. 또는 유지보수 Playbook에서 Claude Code가 직접 `gh run view` 등을 실행하도록 지시 |
| **세션당 50개 Scheduled Task 제한** | Claude Code 제약 | 유사 작업을 배치로 묶는 meta-task 패턴 사용. 예: "dependency-sweep"이 npm+go+pip을 모두 처리 |

### 6-8. Realistic MVP 스코프

**MVP (빌드 가능):**
- 수동 유지보수 작업 실행 (`ham maintain run dependency-sweep`)
- 작업 실행 이력 저장 및 조회
- 결과를 Notification Inbox로 보고
- 미리 정의된 작업 템플릿 3개: dependency-sweep, flaky-test-triage, docs-drift

**MVP 이후:**
- Claude Code Scheduled Tasks 연동 (자동 cron 등록)
- PR babysitting (CI 모니터링 + 자동 수정)
- Release shepherding
- Studio UI 스케줄 관리

---

## 7. 기능 5: Pack Marketplace

### 7-1. 기능 설명 + 사용자 시나리오

Pack Marketplace는 팀이 만든 **재사용 가능한 ham 확장 패키지**를 공유하는 시스템이다.

**시나리오 A - Playbook Pack 공유:**
백엔드 팀이 만든 "API 서비스 마이그레이션 플레이북"을 Pack으로 패키징하여 조직 내 다른 팀에게 공유한다. 다른 팀은 `ham pack install backend-migration`으로 설치하고 자신의 프로젝트에 맞게 파라미터를 조정한다.

**시나리오 B - Policy Pack:**
보안 팀이 만든 "SOC2 컴플라이언스 정책 팩"을 배포한다. 설치하면 `.ham/policies/` 디렉토리에 사전 정의된 정책 파일이 추가된다.

**시나리오 C - Debugger Preset Pack:**
"React 앱 디버깅 프리셋"을 설치하면, React 관련 에러 패턴에 맞는 breakpoint 규칙과 메모리 항목이 추가된다.

**시나리오 D - Dashboard Pack:**
"팀 리드용 대시보드 팩"을 설치하면, Studio UI에 팀 전체 에이전트 가동률, 비용 추이, 정책 위반 빈도를 보여주는 커스텀 대시보드 위젯이 추가된다.

### 7-2. 필요한 데이터

**현재 있는 것:**
- 없음. Pack 시스템은 Phase 3에서 완전히 새로 만드는 기능.

**새로 만들어야 하는 것:**
| 데이터 | 용도 | 설명 |
|--------|------|------|
| `PackManifest` | 패키지 정의 | `pack.yaml` — 이름, 버전, 설명, 저자, 의존성, 포함 파일 목록, 호환 ham 버전 |
| `PackContent` | 패키지 내용물 | playbook YAML, policy YAML, memory JSON, breakpoint 규칙, 대시보드 정의 파일 |
| `PackRegistry` | 레지스트리 | 설치된 Pack 목록. 설치 경로, 버전, 활성/비활성 |
| `PackSource` | 배포 소스 | git 리포 URL, 로컬 디렉토리, 또는 향후 중앙 레지스트리 |

### 7-3. Go 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `go/internal/core/pack.go` | 신규 | `PackManifest`, `PackRegistry`, `PackSource` 타입 정의 |
| `go/internal/runtime/pack_manager.go` | 신규 | Pack 설치/제거/업데이트/목록. git clone 또는 로컬 복사 |
| `go/internal/runtime/pack_loader.go` | 신규 | 설치된 Pack에서 playbook/policy/memory/breakpoint를 각 서브시스템에 로딩 |
| `go/internal/store/pack_store.go` | 신규 | PackRegistry 영속화 (`~/.ham/packs/registry.json`) |
| `go/cmd/ham/cmd_pack.go` | 신규 | `ham pack install`, `ham pack remove`, `ham pack list`, `ham pack update`, `ham pack create` CLI 명령 |

### 7-4. Swift 변경사항

| 파일 | 변경 | 설명 |
|------|------|------|
| `Sources/HamCore/PackPayloads.swift` | 신규 | PackManifest 디코딩 모델 |
| `Sources/HamCore/DaemonIPC.swift` | 수정 | Pack 관련 DaemonCommand 추가 |
| `Sources/HamAppServices/PackViewModel.swift` | 신규 | 설치된 Pack 관리 ViewModel |
| `apps/macos/HamMenuBarApp/Sources/PackViews.swift` | 신규 | Studio 내 Pack 탭 (설치됨, 탐색, 설치/제거) |

### 7-5. IPC 변경사항

| 커맨드 | Request 필드 | Response 필드 | 설명 |
|--------|-------------|---------------|------|
| `pack.list` | - | `[]PackManifest` | 설치된 Pack 목록 |
| `pack.install` | `source` (git URL or path) | `PackManifest` | Pack 설치 |
| `pack.remove` | `pack_name` | - | Pack 제거 |
| `pack.update` | `pack_name` | `PackManifest` | Pack 업데이트 |
| `pack.create` | `output_path` | `PackManifest` | 현재 프로젝트 설정에서 Pack 생성 |

### 7-6. 선행 작업 / 의존성

| 의존성 | Phase | 설명 |
|--------|-------|------|
| Phase 2 Playbook | Phase 2 | Pack에 포함할 Playbook 포맷 |
| Org Policy Engine | Phase 3 | Pack에 포함할 Policy 포맷 |
| Persistent Memory Graph | Phase 3 | Pack에 포함할 Memory 항목 포맷 |
| AI Agent Debugger | Phase 3 | Pack에 포함할 Breakpoint 규칙 포맷 |
| Phase 2 ham Studio | Phase 2 | Pack 탐색/관리 UI |

### 7-7. 구현 불가능한 부분과 대안

| 불가능한 것 | 원인 | 대안 |
|-------------|------|------|
| **중앙 Pack 레지스트리 (npm-style)** | 서버 인프라 없음. ham은 로컬 도구 | **git 기반 배포**: Pack은 git 리포로 배포. `ham pack install https://github.com/team/my-pack`으로 설치. 향후 GitHub Releases + 인덱스 파일로 탐색 기능 추가. Claude Code의 Agent Skills(agentskills.io) 표준과 호환 검토 |
| **Pack 내 실행 코드 (플러그인)** | 보안 위험. 임의 코드 실행은 거버넌스 목적에 반함 | Pack은 **선언적 파일만** 포함: YAML(playbook, policy), JSON(memory, breakpoint), Markdown(문서). 실행 로직은 ham 코어 + Claude Code에 위임 |
| **Pack 간 의존성 해결** | 복잡한 의존성 그래프 관리는 스코프 초과 | 단순 의존성만 지원: `requires: [pack-name >= 1.0]`. 순환 의존성은 설치 시 에러. 복잡한 의존성 트리는 지원하지 않음 |
| **대시보드 위젯 커스텀 UI** | Swift UI 코드를 동적으로 로딩하는 것은 macOS 앱 서명 정책에 반함 | **데이터 기반 대시보드**: Pack은 "어떤 데이터를 어떤 차트로 보여줄지"를 JSON 스키마로 정의. Studio가 빌트인 차트 컴포넌트로 렌더링 |

#### Realism Check

| 필요 데이터 / 기능 | 현재 확보 경로 | 실현성 |
|-------------------|---------------|--------|
| 로컬 Pack 설치/제거 | git clone + 파일 복사로 구현 가능 | ✓ |
| Playbook/Policy/Memory 로딩 | 각 서브시스템이 안정된 후 Pack 포맷 정의 가능 | ⚠ Phase 3 후반부 의존 |
| 중앙 레지스트리 (npm registry 유사) | ham 로컬 데몬, 서버 인프라 없음 | ✗ git URL 기반 배포만 가능 |
| 대시보드 위젯 커스텀 UI | macOS 앱 서명 정책 — 동적 Swift 코드 로딩 불가 | ✗ 데이터 기반 빌트인 차트만 가능 |
| Pack 서명 / 보안 검증 | 서명 인프라 없음 | ✗ 사용자 책임 (git URL 신뢰) |

**결론**: Pack Marketplace 는 "로컬 Pack 설치/로딩" 은 구현 가능하다. 그러나 "Marketplace" 라는 이름이 암시하는 중앙 레지스트리, 탐색, 검증 기능은 Phase 3 범위 밖이다. MVP 는 **"Pack Loader (git URL 기반)"** 수준으로 시작하며, 중앙 레지스트리는 ham cloud 서비스 계획이 확정된 후 별도 계획한다. 기능명 뒤에 **(Best-effort, git URL 기반 — 중앙 레지스트리 없음)** 태그를 부착한다.

### 7-9. Realistic MVP 스코프

**MVP (빌드 가능):**
- `pack.yaml` 매니페스트 포맷 정의
- 로컬 디렉토리에서 Pack 설치/제거 (`ham pack install ./my-pack`)
- 설치된 Pack의 playbook/policy 파일을 해당 서브시스템에 자동 로딩
- `ham pack list` / `ham pack create`

**MVP 이후:**
- git URL에서 Pack 설치
- Pack 업데이트 (git pull)
- Studio UI Pack 관리
- 대시보드 위젯 시스템
- 커뮤니티 Pack 인덱스

---

## 8. 저장소 전략

### 8-1. 파일 기반 저장소의 한계

Phase 3 기능들은 현재 파일 기반 저장소로는 지원이 어렵다:

| 기능 | 필요한 쿼리 | 파일 기반 한계 |
|------|------------|---------------|
| AI Agent Debugger | 세션별 이벤트 필터링, 시간 범위 검색, tool-call 체인 재구성 | JSONL 전체 스캔 필요. 10K 상한으로 장기 분석 불가 |
| Org Policy Engine | 정책 위반 이력 검색, 감사 로그 시간 범위 쿼리 | JSONL 추가 파일 필요. 인덱스 없음 |
| Persistent Memory Graph | 키워드 검색, 관계 그래프 탐색, 스코프별 필터링 | JSON 파일로 그래프 표현 비효율. 전문 검색 불가 |
| Autonomous Maintenance | 실행 이력 조회, 작업별 통계 | JSONL 또는 JSON 파일 추가 |

### 8-2. Embedded DB 전환 계획

**Phase 3 진입 시점에 embedded DB 도입을 권장한다.**

| 후보 | 장점 | 단점 |
|------|------|------|
| **SQLite (via go-sqlite3 또는 modernc.org/sqlite)** | 검증된 안정성, 풍부한 쿼리, FTS5 전문검색, JSON1 확장 | CGO 의존(go-sqlite3) 또는 순수 Go 포트 성능 이슈(modernc) |
| **Badger** | 순수 Go, 고성능 KV, LSM-tree | SQL 쿼리 없음. 복잡한 쿼리는 직접 구현 |
| **BBolt** | 순수 Go, 단순 KV, B+tree | 범위 쿼리 가능하지만 인덱싱 직접 구현 |

**권장: SQLite (modernc.org/sqlite)**
- CGO 불필요 (순수 Go 포트)
- FTS5로 메모리 검색 지원
- 복잡한 쿼리 (JOIN, GROUP BY, 시간 범위) 자연스러움
- `~/Library/Application Support/ham-agents/ham.db` 단일 파일

**마이그레이션 전략:**
1. 기존 JSONL/JSON 파일은 유지 (하위 호환)
2. SQLite를 추가 read model로 도입
3. 기존 파일 데이터를 SQLite로 마이그레이션하는 `ham migrate` 명령 제공
4. Phase 3 기능은 SQLite만 사용
5. 장기적으로 JSONL write-ahead log + SQLite read model 이중 구조

### 8-3. 용량 추정

| 데이터 | 예상 증가율 | 1년 후 예상 용량 |
|--------|------------|-----------------|
| Event Log | ~1,000건/일 (활발한 사용) | ~365K건, ~150MB |
| SessionTrace | ~10건/일 | ~3.6K건, ~50MB |
| MemoryNode | ~5건/일 | ~1.8K건, ~5MB |
| PolicyViolation | ~2건/일 | ~730건, ~1MB |
| MaintenanceRun | ~3건/일 | ~1K건, ~2MB |
| PackRegistry | 소수 | <1MB |
| **합계** | - | **~210MB** |

SQLite 단일 파일로 충분히 관리 가능한 규모.

---

## 9. Claude Code 통합 포인트

### 9-1. Skills 연동

| ham 기능 | Skills 활용 | 설명 |
|----------|------------|------|
| Memory Graph | `SKILL.md` + YAML frontmatter | 메모리 항목을 skill로 노출. Claude Code가 세션 시작 시 관련 메모리를 자동 로드 |
| Playbook (Phase 2) | Agent Skills 표준 (agentskills.io) | Playbook을 Agent Skills 포맷으로 컴파일하여 Claude Code에서 직접 실행 가능 |
| Pack | Skills 번들 | Pack에 포함된 skill 파일을 Claude Code의 skill 디렉토리에 심링크 |

### 9-2. Scheduled Tasks 연동

| ham 기능 | Scheduled Tasks 활용 | 설명 |
|----------|---------------------|------|
| Autonomous Maintenance | CronCreate/CronList/CronDelete | ham이 유지보수 작업을 Claude Code scheduled task로 등록 |
| PR Babysitting | 주기적 체크 | PR 상태를 주기적으로 확인하는 scheduled task |
| Docs Drift | 일일 1회 | 코드-문서 불일치를 일일 검사하는 scheduled task |

**제약: 세션당 최대 50개.** ham은 유사 작업을 배치로 묶어 효율적으로 슬롯을 사용해야 한다.

### 9-3. Remote Control (Server Mode) 연동

| ham 기능 | Remote Control 활용 | 설명 |
|----------|---------------------|------|
| Autonomous Maintenance | `--spawn` worktree | 유지보수 세션을 격리된 worktree에서 실행 |
| Debugger | capacity 모니터링 | `--capacity 32` 환경에서 동시 세션 상태 추적 |

**제약: OAuth 인증 필요.** ham이 server mode에 접근하려면 OAuth 토큰 관리가 필요. 초기 MVP에서는 수동 토큰 설정.

### 9-4. Channels 연동

| ham 기능 | Channels 활용 | 설명 |
|----------|--------------|------|
| Autonomous Maintenance | Telegram/Discord/iMessage | 유지보수 결과를 채널로 보고 |
| Policy Engine | 위반 알림 | 정책 위반을 팀 채널로 전파 |
| Debugger | 실패 리포트 | 세션 실패 시 채널로 디버그 요약 전송 |

**현재 상태: Research Preview.** 양방향 통신 가능. ham의 Notification Inbox와 연동하여 알림을 외부 채널로 라우팅.

---

## 10. 경쟁 제품 참조

### 10-1. 디버거 / Observability

| 제품 | 관련 기능 | 참조 포인트 | URL |
|------|----------|------------|-----|
| **AgentOps.ai** | 세션 리플레이, 비용 추적 | 세션 타임라인 UI, tool-call 시각화, LLM cost dashboard | https://www.agentops.ai/ |
| **LangSmith** (LangChain) | 트레이싱, 평가, 모니터링 | trace 계층 구조 (run tree), 입출력 캡처, latency 분석 | https://smith.langchain.com/ |
| **Langfuse** | 오픈소스 LLM observability | trace/span 모델, 프롬프트 관리, 비용 추적, 평가 파이프라인 | https://langfuse.com/ |
| **Braintrust** | LLM 평가 + 로깅 | A/B 비교, scoring, dataset 관리 | https://www.braintrust.dev/ |
| **Helicone** | LLM 프록시 + 분석 | 요청 레벨 로깅, 비용/latency 대시보드, 캐싱 | https://www.helicone.ai/ |

**ham 차별점:** 위 도구들은 대부분 API 호출 수준의 로깅이다. ham은 Claude Code의 hook/이벤트 시스템에 네이티브로 통합되어 **세션 수준의 행동 관찰**이 가능하다. 특히 permission 요청/거부, context compaction, sub-agent 스포닝 같은 Claude Code 고유 이벤트를 추적할 수 있다.

### 10-2. 유지보수 자동화

| 제품 | 관련 기능 | 참조 포인트 | URL |
|------|----------|------------|-----|
| **Cursor Automations** | 자동 이슈 해결 | GitHub Issue -> 자동 코드 수정 -> PR 생성 파이프라인 | https://www.cursor.com/ |
| **Cursor Bugbot** | 버그 탐지 | PR의 잠재적 버그를 자동 탐지하여 코멘트 | https://www.cursor.com/ |
| **Dependabot** (GitHub) | 의존성 업데이트 | 자동 dependency PR, 보안 취약점 알림 | https://github.com/dependabot |
| **Renovate** | 의존성 업데이트 | 다중 언어 지원, 세밀한 스케줄링, automerge 정책 | https://www.mend.io/renovate/ |
| **Socket** | 공급망 보안 | 패키지 위험도 분석, 알려지지 않은 취약점 탐지 | https://socket.dev/ |

**ham 차별점:** Cursor Automations는 cloud agent 기반이고 Cursor 에디터에 종속된다. Dependabot/Renovate는 의존성에 특화된 단일 목적 도구다. ham은 Claude Code를 실행 엔진으로 활용하여 **임의의 유지보수 작업**을 Playbook으로 정의하고 실행할 수 있다.

### 10-3. 오케스트레이션 / 에이전트 운영

| 제품 | 관련 기능 | 참조 포인트 | URL |
|------|----------|------------|-----|
| **Warp Oz** | 클라우드 에이전트 오케스트레이션 | Agent Management Panel, 병렬 에이전트, 터미널 통합 | https://www.warp.dev/ |
| **CrewAI** | 멀티 에이전트 프레임워크 | 역할 기반 에이전트, task delegation, 협업 패턴 | https://www.crewai.com/ |
| **AutoGen** (Microsoft) | 멀티 에이전트 대화 | 에이전트 간 대화 프로토콜, 그룹 채팅, 코드 실행 | https://microsoft.github.io/autogen/ |
| **Composio** | 도구/통합 플랫폼 | 150+ 도구 통합, 인증 관리, 실행 환경 | https://composio.dev/ |

**ham 차별점:** Warp는 terminal replacement를 지향하고, CrewAI/AutoGen은 프레임워크를 지향한다. ham은 기존 터미널(iTerm/tmux/SSH) 위에 얹히는 **terminal-agnostic control plane**이며, Claude Code의 네이티브 기능(hooks, teams, skills)을 직접 활용하므로 별도 프레임워크 레이어가 필요 없다.

### 10-4. 정책 / 거버넌스

| 제품 | 관련 기능 | 참조 포인트 | URL |
|------|----------|------------|-----|
| **Guardrails AI** | LLM 출력 검증 | 입출력 가드레일, 구조화 출력 강제, 유효성 검사 | https://www.guardrailsai.com/ |
| **Lakera Guard** | AI 보안 | 프롬프트 인젝션 탐지, 민감 데이터 감지 | https://www.lakera.ai/ |
| **OPA (Open Policy Agent)** | 범용 정책 엔진 | Rego 언어, 선언적 정책, 다양한 통합 | https://www.openpolicyagent.org/ |

**ham 차별점:** 기존 정책 도구들은 API 호출 수준이거나 인프라 수준이다. ham의 Policy Engine은 **Claude Code의 tool-call 수준**에서 동작하며, hook 이벤트 스트림에 기반한 실시간 감지가 가능하다. OPA의 Rego처럼 범용 언어를 만들기보다, YAML 기반의 단순한 규칙 언어로 시작하되 Claude Code의 도구 체계에 맞춘 도메인 특화 정책을 제공한다.

### 10-5. 커뮤니티 도구 참조

| 도구 | 설명 | URL |
|------|------|-----|
| **Claude Code Hooks** | Claude Code 공식 hook 시스템 문서 | https://docs.anthropic.com/en/docs/claude-code/hooks |
| **Agent Skills** | 에이전트 스킬 공유 오픈 표준 | https://agentskills.io/ |
| **Model Context Protocol** | 도구/리소스 표준 프로토콜 | https://modelcontextprotocol.io/ |
| **AGENTS.md** | 에이전트 지시 파일 표준 (비공식) | https://github.com/anthropics/claude-code/blob/main/AGENTS.md |
| **llm-cost** | LLM API 비용 계산 라이브러리 | https://github.com/BerriAI/litellm |

---

## 부록: Phase 3 구현 순서 제안

```
Phase 3 구현 순서 (선행 의존성 기반):

1. 저장소 전환 (SQLite 도입)
   └── 모든 Phase 3 기능의 전제 조건

2. AI Agent Debugger (MVP)
   ├── SessionTrace 구성 (이벤트 로그 -> 세션 트레이스)
   ├── CLI 리플레이 + step-through
   └── Phase 1 EventBus + 이벤트 스키마에 의존

3. Org Policy Engine (MVP)
   ├── YAML 정책 파일 로딩
   ├── tool-call 패턴 매칭 + 경고
   └── Phase 1 EventBus에 의존

4. Persistent Memory Graph (MVP)
   ├── 수동 메모리 CRUD
   ├── CLAUDE.md 연동
   └── SQLite FTS5 검색에 의존

5. Autonomous Maintenance (MVP)
   ├── 수동 유지보수 작업 실행
   ├── 결과 보고
   └── Phase 2 Playbook + Phase 3 Policy Engine에 의존

6. Pack Marketplace (MVP)
   ├── 로컬 Pack 설치/제거
   ├── Playbook/Policy/Memory 로딩
   └── 모든 Phase 3 서브시스템이 안정된 후
```

---

> **이 문서는 Phase 3의 설계 방향을 정의하는 기획서이며, 구현 과정에서 Phase 1/2의 실제 결과에 따라 조정될 수 있다. 특히 Claude Code 생태계의 변화(새 API, 새 기능)에 따라 "구현 불가능한 부분"이 해소될 수 있으므로 정기적으로 재검토해야 한다.**
