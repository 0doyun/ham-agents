# Technical Migration Specification

> 2026.04 | ham-agents v2.0 기술 마이그레이션 명세 | roadmap-0405.md 섹션 6 구현 계획

---

## 목차

1. [마이그레이션 의존성 그래프](#마이그레이션-의존성-그래프)
2. [페이즈 매핑](#페이즈-매핑)
3. [6-1. hamd를 레지스트리에서 이벤트 브로커로 승격](#6-1-hamd를-레지스트리에서-이벤트-브로커로-승격)
4. [6-2. 정규화된 이벤트 스키마 도입](#6-2-정규화된-이벤트-스키마-도입)
5. [6-3. 저장소를 event log + read model로 분리](#6-3-저장소를-event-log--read-model로-분리)
6. [6-4. IPC를 명령 채널과 스트림 채널로 분리](#6-4-ipc를-명령-채널과-스트림-채널로-분리)
7. [6-5. Claude Code 어댑터를 계층화](#6-5-claude-code-어댑터를-계층화)
8. [6-6. UI를 2계층으로 확장](#6-6-ui를-2계층으로-확장)
9. [6-7. 확장성은 Claude 생태계에 맞춘다](#6-7-확장성은-claude-생태계에-맞춘다)
10. [6-8. 품질 전략을 제품 기능으로](#6-8-품질-전략을-제품-기능으로)
11. [성능 고려사항](#성능-고려사항)
12. [저장소 진화 계획](#저장소-진화-계획)
13. [커뮤니티 접근법 비교](#커뮤니티-접근법-비교)

---

## 마이그레이션 의존성 그래프

```
6-2 (이벤트 스키마)
 ├──→ 6-1 (이벤트 브로커)  ──→ 6-4 (IPC 분리)
 └──→ 6-3 (저장소 분리)    ──→ 6-4 (IPC 분리)

6-5 (어댑터 계층화) ── 독립, 단 6-2 스키마 확정 후 진행 권장

6-4 (IPC 분리) ──→ 6-6 (UI 2계층)

6-7 (확장성)   ── 6-2 스키마 + 6-6 UI 이후

6-8 (품질 전략) ── 모든 단계에 병렬 진행, 각 항목의 테스트가 해당 항목과 동시 구현
```

**의존성 요약:**

| 항목 | 선행 의존 | 후행 의존 |
|------|-----------|-----------|
| 6-1 | 6-2 | 6-4 |
| 6-2 | 없음 (최우선) | 6-1, 6-3, 6-5 |
| 6-3 | 6-2 | 6-4 |
| 6-4 | 6-1, 6-3 | 6-6 |
| 6-5 | 6-2 (권장) | 없음 |
| 6-6 | 6-4 | 6-7 |
| 6-7 | 6-2, 6-6 | 없음 |
| 6-8 | 없음 (상시) | 없음 |

---

## 페이즈 매핑

### Phase 1: 이벤트 기반 전환 (Mission Control MVP와 동시)

| 항목 | 설명 | 예상 규모 |
|------|------|-----------|
| 6-2 | 이벤트 스키마 정규화 | `core/agent.go` Event 구조체 확장, additive 변경 |
| 6-8a | hook contract test + synthetic replayer | 테스트 인프라 |
| 6-5a | hooks 어댑터 계층 정리 | 기존 코드 리팩터링 |

### Phase 2: 브로커 + 저장소 분리

| 항목 | 설명 | 예상 규모 |
|------|------|-----------|
| 6-1 | EventBus 도입, Registry 역할 축소 | `runtime/` 패키지 대규모 변경 |
| 6-3 | write-ahead log + read model 분리 | `store/` 패키지 신규 + 기존 변경 |
| 6-4a | 스트림 채널 프로토타입 (NDJSON over UDS) | ipc/ 패키지 — 기존 server.go 에 PTY dispatch 추가 (ADR-2 참조) |
| 6-8b | attach/detach chaos test, golden traces | 테스트 인프라 |

### Phase 3: UI 확장 + 생태계

| 항목 | 설명 | 예상 규모 |
|------|------|-----------|
| 6-4b | 스트림 채널 완성, 폴링 제거 | Swift + Go 양쪽 변경 |
| 6-6 | ham Studio 도입, 메뉴바 경량화 | Swift 대규모 신규 |
| 6-5b | PTY/attach + observe 어댑터 | Go 어댑터 확장 |
| 6-7 | Playbook 포맷, 확장 팩 | 신규 패키지 |
| 6-8c | cost/approval/status reducer 회귀 테스트 | 테스트 인프라 |

---

## 6-1. hamd를 레지스트리에서 이벤트 브로커로 승격

### 현재 상태

hamd의 핵심 객체는 `Registry` (`go/internal/runtime/registry.go`)이며, 이것은 agent 상태 저장소다.

**데이터 흐름:**
```
hook/CLI → IPC Server (go/internal/ipc/server.go) → Registry.mutateAgent()
  → FileAgentStore.SaveAgents() (managed-agents.json 전체 스냅샷)
  → Registry.appendEvent() → FileEventStore.Append() (events.jsonl 1행 추가)
```

**핵심 함수:**
- `Registry.mutateAgent()` (`registry.go:239-246`): Lock → LoadAgents(전체) → Find → Mutate → SaveAgents(전체) + Append event → Unlock
- `Registry.mutateAgentLocked()` (`registry.go:251-286`): 실제 load-mutate-save 사이클
- `Registry.saveAgentsAndEvents()` (`registry.go:321-329`): store.SaveAgents + eventStore.Append
- `Registry.appendEvent()` (`registry.go:331-355`): 이벤트 ID/시간 자동 채움, 프레젠테이션 힌트 생성

**문제점:**
- 모든 mutation이 전체 agent 목록을 load/save한다 (O(N) per operation)
- Registry가 상태 저장, 이벤트 기록, 비즈니스 로직을 모두 담당하는 God Object
- 이벤트는 부산물(side effect)이지 핵심 데이터가 아님
- fan-out 없음: 이벤트 소비자(Swift UI)는 폴링으로만 확인 가능

### 목표 상태

hamd의 핵심 객체가 `Agent`에서 `SessionEvent`로 전환된다.

```
hook/CLI → IPC Server → EventBus.Publish(SessionEvent)
  → subscriber: AgentProjector (read model 갱신)
  → subscriber: EventStore (JSONL 영속)
  → subscriber: StreamChannel (실시간 UI 푸시)
  → subscriber: PolicyEngine (승인/비용 정책 평가)
```

**EventBus 설계:**
- `Publish(event SessionEvent) error`: 이벤트를 모든 subscriber에 fan-out
- `Subscribe(filter EventFilter) <-chan SessionEvent`: 필터 기반 구독
- 동기 subscriber (저장소, projector)와 비동기 subscriber (UI 스트림) 분리
- back-pressure: 비동기 subscriber의 채널이 가득 차면 drop + 로그

### 마이그레이션 전략

**단계 1: EventBus 인터페이스 도입 (Registry 내부)**

Registry 내부에 EventBus를 주입하되, 기존 `appendEvent()` 호출부를 EventBus.Publish()로 교체한다. 초기 subscriber는 기존 FileEventStore 하나뿐이므로 외부 동작 변화 없음.

**단계 2: AgentProjector 분리**

`mutateAgent()`의 load-mutate-save 패턴을 분해한다:
1. hook 이벤트를 EventBus에 publish
2. AgentProjector가 이벤트를 받아 in-memory agent 상태를 갱신
3. AgentProjector가 주기적으로 또는 변경 시 FileAgentStore에 스냅샷 저장

**단계 3: Registry 경량화**

Registry를 읽기 전용 facade로 축소한다. List/Snapshot은 AgentProjector에서 직접 읽음.

### Go 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `go/internal/runtime/eventbus.go` (신규) | `EventBus` 인터페이스, `InMemoryEventBus` 구현체 (sync.Mutex + []chan) |
| `go/internal/runtime/projector.go` (신규) | `AgentProjector`: 이벤트 스트림 → in-memory agent map 갱신 |
| `go/internal/runtime/registry.go` | `Registry` 구조체에 `eventBus EventBus` 필드 추가, `appendEvent()` → `eventBus.Publish()` 교체 |
| `go/internal/runtime/registry.go` | `mutateAgent()`를 점진적으로 이벤트 기반으로 전환 (단계 2) |
| `go/cmd/hamd/main.go` | `NewRegistry()` 호출 시 EventBus 주입, subscriber 등록 |

### Swift 변경사항

Phase 1에서는 Swift 변경 없음. EventBus 도입은 Go 내부 리팩터링이며, IPC 프로토콜은 유지된다. Swift 쪽 변경은 6-4 (스트림 채널)에서 발생.

### IPC 변경사항

Phase 1에서는 IPC 프로토콜 변경 없음. 기존 52개 Command 상수와 Request/Response 구조체를 그대로 유지한다. EventBus는 서버 내부에서만 동작하며, 외부 클라이언트는 기존 폴링 방식으로 계속 접근 가능.

### 하위 호환성

- `managed-agents.json` 형식 유지 (AgentProjector가 동일 형식으로 스냅샷 저장)
- `events.jsonl` 형식 유지 (FileEventStore subscriber가 기존 방식으로 append)
- IPC Command/Request/Response 구조체 변경 없음
- `ham` CLI의 모든 기존 명령이 동일하게 동작

### 위험 요소

1. **EventBus 순서 보장**: 동기 subscriber (projector, store)는 publish 순서대로 처리되어야 한다. 비동기 subscriber에서 순서가 뒤집히면 UI에 일시적 불일치 발생 가능.
2. **AgentProjector 장애 시 복구**: in-memory 상태가 손실되면 FileAgentStore 스냅샷에서 재구축해야 한다. events.jsonl로부터의 replay 메커니즘 필요.
3. **mutateAgent 분해 중 이중 쓰기**: 단계 2 전환 중 Registry가 직접 저장하면서 동시에 EventBus를 통해 Projector도 저장하는 이중 쓰기 구간이 생긴다. 이 구간을 최소화해야 한다.

### 선행 작업

- 6-2 (이벤트 스키마 정규화)가 먼저 완료되어야 EventBus가 전달하는 이벤트 타입이 확정됨

---

## 6-2. 정규화된 이벤트 스키마 도입

### 현재 상태

이벤트는 `core.Event` 구조체 (`go/internal/core/agent.go:155-168`)에 정의되어 있다:

```go
type Event struct {
    ID                   string    `json:"id"`
    AgentID              string    `json:"agent_id"`
    Type                 EventType `json:"type"`
    Summary              string    `json:"summary"`
    OccurredAt           time.Time `json:"occurred_at"`
    PresentationLabel    string    `json:"presentation_label,omitempty"`
    PresentationEmphasis string    `json:"presentation_emphasis,omitempty"`
    PresentationSummary  string    `json:"presentation_summary,omitempty"`
    LifecycleStatus      string    `json:"lifecycle_status,omitempty"`
    LifecycleMode        string    `json:"lifecycle_mode,omitempty"`
    LifecycleReason      string    `json:"lifecycle_reason,omitempty"`
    LifecycleConfidence  float64   `json:"lifecycle_confidence,omitempty"`
}
```

**EventType 상수** (`agent.go:138-153`): 13개 타입 (agent.registered, agent.role_updated, agent.status_updated, agent.removed, team.teammate_idle 등)

**문제점:**
- `session_id`, `parent_id`, `task_id` 없음 -- 이벤트를 세션/태스크 단위로 그룹핑 불가
- `source` 없음 -- hook에서 온 이벤트인지, 폴링에서 온 이벤트인지, 수동 조작인지 구분 불가
- `confidence` 필드가 Event가 아닌 Agent에만 존재 (`Agent.StatusConfidence`)
- `artifact_ref`, `cost`, `approval_state` 없음 -- 디버거, 비용 추적, 거버넌스에 필요한 데이터
- Managed/Attached/Observed가 별도 `AgentMode`이지만, 실제로는 같은 이벤트의 다른 소스일 뿐
- `Summary` 필드가 자유 형식 문자열이라 구조화된 쿼리 불가

### 목표 상태

> **참고**: SessionEvent 는 별도 타입이 아니라 `core.Event` 를 확장한 것이다. 단일 스키마 정의는 **mission-control.md ADR-1** 에 있다. 이 문서는 ADR-1 을 참조만 한다.

**Mode 통합**: Managed/Attached/Observed는 ADR-1의 `Source` 필드로 구분한다.
- `source: "hook"` = 현재의 Managed (hooks로부터 직접 수신)
- `source: "pty"` = 현재의 Attached (PTY 관찰)
- `source: "log"` = 현재의 Observed (로그 파싱)
- `source: "poll"` = iTerm2/tmux 폴링으로 추론

### 마이그레이션 전략

**Additive 확장 원칙**: 기존 `Event` 구조체에 새 필드를 추가하되, 기존 필드를 제거하지 않는다. JSONL 파일의 기존 이벤트는 새 필드가 zero value로 디코딩된다.

**단계 1: Event 구조체 확장**

`core.Event`에 신규 필드를 `omitempty`로 추가한다. 기존 코드는 영향 없음.

**단계 2: 이벤트 생성부에서 신규 필드 채우기**

`Registry`의 각 `RecordHook*` 함수에서 `Source`, `SessionID`, `Confidence`, `ConfidenceModel`을 채운다. 이 작업은 각 hook handler에 1-2줄 추가로 완료.

**단계 3: CostInfo 수집 경로 구축**

Claude Code hooks에서 비용 정보를 추출하는 경로를 만든다. 현재 hooks에는 cost 데이터가 없으므로, `hook.stop` 또는 `hook.agent-finished`의 마지막 메시지에서 파싱하거나, 별도 cost hook을 기다린다.

### Go 변경사항

> **참고**: 추가할 필드의 권위 있는 정의는 **mission-control.md ADR-1** 을 따른다. 아래 표는 구현 위치 안내이며, 스키마 불일치 시 ADR-1 이 우선한다.

| 파일 | 변경 내용 |
|------|-----------|
| `go/internal/core/agent.go` | `Event` 구조체에 ADR-1 필드 추가 (SessionID, ParentAgentID, TaskID, Source, Confidence, ConfidenceModel, ArtifactRef, ApprovalState, Payload, Cost 등 — 전체 목록은 ADR-1 참조) |
| `go/internal/core/agent.go` | `EventType` 상수 확장: `event.cost_recorded`, `event.approval_requested`, `event.approval_resolved` |
| `go/internal/runtime/registry.go` | `appendEvent()`에서 `Source` 필드 기본값 설정 로직 |
| `go/internal/runtime/managed_state.go` | 각 `RecordHook*` 함수에서 `Source: "hook"`, `SessionID` 채우기 |
| `go/internal/runtime/registry_attached.go` | Attached 이벤트에 `Source: "pty"` 설정 |
| `go/internal/runtime/registry_observed.go` | Observed 이벤트에 `Source: "log"` 설정 |

### Swift 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `Sources/HamCore/DaemonPayloads.swift` | `AgentEventPayload`에 `sessionID`, `source`, `confidence`, `cost` 등 옵셔널 필드 추가 |
| `Sources/HamAppServices/EventPresentation.swift` | `AgentEventPresenter`에서 `source`, `confidence` 기반 표시 로직 |

### IPC 변경사항

IPC 프로토콜 변경 없음. `Response.Events` 배열의 각 `Event` JSON에 새 필드가 추가되지만, `omitempty`이므로 기존 클라이언트는 무시한다. Swift 측 디코딩도 unknown key를 무시하는 기본 동작.

### 하위 호환성

- 기존 `events.jsonl`의 이벤트는 새 필드가 zero value로 디코딩됨 (정상)
- 기존 `Event` 필드 (`LifecycleStatus`, `LifecycleMode` 등) 유지
- 기존 IPC Response 형식 완전 호환
- Swift `AgentEventPayload`의 새 필드는 모두 옵셔널

### 위험 요소

1. **JSONL 크기 증가**: 새 필드 추가로 이벤트당 바이트 수 증가. `Payload` 필드가 `map[string]any`이므로 의도치 않게 큰 데이터가 들어갈 수 있음. 크기 제한 필요.
2. **Cost 데이터 신뢰성**: Claude Code hooks에서 비용 정보가 직접 제공되지 않으므로, 추정치를 사용해야 할 수 있음. `ConfidenceModel`로 이를 구분.
3. **스키마 버전 관리**: 향후 필드 변경 시 migration path가 필요. 이벤트에 `schema_version` 필드를 두는 것을 고려.

### 선행 작업

- 없음. 이 항목이 전체 마이그레이션의 시작점.

---

## ADR-2: PTY Transport for Embedded Studio Terminal

> **Note (라운드 3 재번호)**: 라운드 2 ADR 목록의 "ADR-2 (Approval 경로 — 외부 API 가용성)" 은 **ADR-7** 로 재번호됐다. 본 ADR-2 는 라운드 3 PTY Transport 결정이다. mission-control.md ADR 상태표 참조.

**Status**: Proposed (2026-04-06, 라운드 3)
**Scope**: docs/spec/ham-studio.md, docs/spec/tech-migration.md, docs/spec/implementation-plan.md
**Depends on**: ADR-1 (SessionEvent 스키마)

### Context
ham-agents Phase 2 방향이 "관찰만 하는 control plane" 에서 "내장 PTY 탭을 가진 Claude Code agent command center" 로 전환됐다. ham Studio 가 primary UX 이고 Claude Code 세션이 Studio 탭 안에서 직접 돌아가야 한다.

현재 상태 (US-001 조사):
- `ham run <provider>` CLI 경로는 이미 `/dev/ptmx` 기반 PTY 를 가지고 있다 (`go/cmd/ham/pty.go:69, 129-153`). 단 ptmx fd 는 ham CLI 프로세스 내부에만 존재하고 hamd 로 넘어가지 않는다
- `hamd.ManagedService.Start` (managed.go:61-70) 경로는 plain `StdoutPipe/StderrPipe` 만 사용한다. 즉 IPC 로 스폰 요청이 들어오면 PTY 가 할당되지 않는다
- `Agent.SessionTTY` (agent.go:62) 필드가 존재하지만 managed 경로에서 설정되지 않는다
- IPC 는 주로 request-response 지만 `CommandFollowEvents` (server.go:279) 는 이미 long-poll 스트리밍 패턴을 사용하고 있다
- Swift `DaemonIPC.swift` 의 16 개 `DaemonCommand` 중 어느 것도 tty/fd/stream 을 노출하지 않는다

### Decision options

세 가지 운송 옵션을 비교한다.

#### Option 1: NDJSON stream upgrade (main IPC 확장)
기존 Unix socket IPC 에 `CommandFollowPTY` / `CommandWritePTY` 등을 추가하고, `CommandFollowEvents` 장기 연결 패턴을 재사용해 base64 인코딩된 PTY 프레임을 NDJSON 으로 스트리밍한다.

- **hamd 구현**: `ManagedService.Start` 에서 `openPTY()` 호출 (pty.go:129-153 코드를 hamd 쪽으로 공유). ptmx 를 `managedProcess` 구조체에 저장. IPC 서버에 PTY subscriber 목록 + ptmx read fan-out 고루틴 추가
- **Swift 구현**: `DaemonCommand.followPTY(agent_id)` 호출 후 NDJSON 프레임을 `AsyncStream<Data>` 로 받아 SwiftTerm feed 에 주입. 사용자 입력은 `writePTY(agent_id, base64)` 로 송신
- **IPC 영향**: 기존 request-response 변형. 새 command 2-3 개 추가. `CommandFollowEvents` 와 동일한 dispatch 패턴
- **장점**: (a) 기존 IPC 경로/권한/소켓 재사용 (b) Swift 쪽 구현이 `CommandFollowEvents` 와 대칭 (c) 디버깅 용이 (`ham events --follow` 처럼 `ham pty --follow` 가능)
- **단점**: (a) base64 인코딩 오버헤드 (평균 33% 데이터 증가) (b) NDJSON 파싱 지연 (작은 프레임도 개행 구분 필요) (c) 대용량 출력 시 소켓 backpressure 처리 필요
- **구현 난이도**: 중

#### Option 2: Per-session pty.sock (별도 소켓)
세션당 전용 Unix 도메인 소켓 (`~/.ham/run/pty-{agent_id}.sock`) 을 열고 Swift 가 직접 read/write. 프레이밍 없이 raw bytes.

- **hamd 구현**: `ManagedService.Start` 에서 ptmx 할당 + 전용 socket listener 생성 + ptmx ↔ socket 양방향 io.Copy. Agent 에 `SessionPtySocket string` 필드 추가
- **Swift 구현**: `CFSocket` 또는 `NWConnection` 으로 `~/.ham/run/pty-{agent_id}.sock` 직접 연결. raw bytes 를 SwiftTerm 에 feed
- **IPC 영향**: 메인 IPC 는 완전히 건드리지 않음. 새 command 0 개 (agent 조회 시 `SessionPtySocket` 경로만 리턴)
- **장점**: (a) 메인 IPC dispatch 에 영향 0 (b) raw bytes 로 인코딩 오버헤드 없음 (c) 세션 격리: 한 PTY 소켓 문제가 다른 세션에 파급되지 않음
- **단점**: (a) 세션당 소켓 파일 관리 (생성/삭제/퍼미션/stale cleanup) (b) 각 세션 listener 수락 루프 추가 (c) Swift 가 소켓 경로를 별도로 받아 관리 (IPC 와 분리된 상태)
- **구현 난이도**: 중상

#### Option 3: SCM_RIGHTS fd passing (fd 전달)
ptmx master fd 를 `sendmsg(SCM_RIGHTS)` 로 Swift 프로세스에 직접 전달. Swift 는 전달받은 fd 를 `FileHandle` 로 감싸 SwiftTerm 에 연결.

- **hamd 구현**: `ManagedService.Start` 에서 ptmx 할당. 기존 IPC 연결에서 fd 를 `unix.Sendmsg` / `SCM_RIGHTS` 로 넘김. Go 에서는 `golang.org/x/sys/unix.Sendmsg` + `unix.UnixRights` 사용
- **Swift 구현**: `recvmsg` + `SCM_RIGHTS` 수신. Swift Foundation 이 `recvmsg` 를 직접 노출하지 않아 C bridging 필요 (`cmsghdr` 매크로). 받은 raw fd 를 `FileHandle(fileDescriptor:)` 로 래핑
- **IPC 영향**: 메인 IPC 에 새 "fd-transfer" 단계 도입. `sendmsg`/`recvmsg` 는 일반 read/write 와 다른 경로라 현재 JSON framing 과 공존 설계 필요
- **장점**: (a) 커널 fd 를 직접 전달하므로 hamd 는 이후 데이터 중계에 관여하지 않음 (b) 최대 성능 (zero-copy) (c) 단일 소유권으로 racy 상태 최소화
- **단점**: (a) Swift 에서 recvmsg/SCM_RIGHTS 사용하려면 C bridging 필요 — 이식성 낮음 (b) fd 전달 실패 시 fallback 경로가 없음 (c) 디버깅 난이도 상 (fd 가 어디로 갔는지 추적 어려움) (d) 권한 모델 복잡 (Sandbox entitlements)
- **구현 난이도**: 상

### Recommended option: Option 1 (NDJSON stream upgrade)

**근거**:
1. `CommandFollowEvents` (server.go:279) 가 이미 long-poll 스트리밍 패턴을 사용하고 있어 구조 재사용 가능. PTY 스트리밍은 이 패턴의 자연스러운 확장
2. Swift 쪽이 C bridging 없이 순수 `AsyncStream` 으로 구현 가능 (`DaemonIPC.swift` 의 기존 JSON decoder 재사용)
3. 디버깅/관측 용이: `ham pty --follow <agent>` CLI 명령으로 같은 스트림을 터미널에서 재생 가능
4. base64 오버헤드는 Phase 2 규모 (개인 개발자 멀티세션, 초당 수 KB~수십 KB) 에서 허용 가능
5. 실패 모드가 명확: 소켓 끊김 → Studio 재연결 → `CommandFollowPTY(resume_from_seq)` 로 resume 가능

**폐기 사유**:
- Option 2: 구조 격리 장점은 크지만 세션당 소켓 관리가 long-term 운영 부담. Phase 3 Policy Engine 이 IPC 경유로 PTY 흐름을 관찰하려면 결국 메인 IPC 와 동등한 레이어를 재구현해야 함
- Option 3: 성능은 최고지만 Swift C bridging 은 이식성/디버깅/entitlements 비용이 구현 시간 내내 누적된다. Phase 2 에서 차분히 굴러가는 것이 최고 성능보다 중요

> **Note on PRD deviation**: PRD 라운드 3 는 Option 2 (per-session pty.sock) 를 expected 로 가리켰다. 본 ADR 은 Option 1 로 편향했는데, 이유는 기존 `CommandFollowEvents` long-poll 패턴이 Option 2 의 "bounded blast radius" 이점의 80% 를 제공하면서 구현 비용은 20% 수준이기 때문이다. 또한 Phase 3 Policy Engine 이 PTY 트래픽을 관찰하려면 결국 메인 IPC 와 동등한 관찰 레이어가 필요하므로 Option 2 의 격리 장점이 실질적으로 희석된다.

### Go-side sketch

```go
// go/internal/runtime/managed.go (conceptual)
type managedProcess struct {
    cmd     *exec.Cmd
    ptmx    *os.File    // ← 신규: PTY master
    subs    []chan []byte // ← 신규: PTY subscribers
    subsMu  sync.Mutex
}

// go/internal/ipc/ipc.go (conceptual)
const (
    CommandFollowPTY Command = "pty.follow"
    CommandWritePTY  Command = "pty.write"
    CommandResizePTY Command = "pty.resize"
)

// go/internal/ipc/server.go dispatch (conceptual)
case CommandFollowPTY:
    return s.handleFollowPTY(ctx, conn, request) // 장기 연결, NDJSON 스트림
case CommandWritePTY:
    return s.handleWritePTY(ctx, request)         // 일회성 request-response
case CommandResizePTY:
    return s.handleResizePTY(ctx, request)        // 일회성 (rows/cols 갱신)
```

NDJSON 프레임 포맷 (PTY → Studio 방향):
```json
{"type":"pty_data","agent_id":"...","seq":123,"data":"<base64>"}
{"type":"pty_exit","agent_id":"...","exit_code":0,"reason":"normal"}
```

Studio → PTY 방향은 별도 `CommandWritePTY` 일회성 호출.

### Swift-side sketch

```swift
// Sources/HamCore/DaemonIPC.swift (conceptual)
enum DaemonCommand: String {
    // ... 기존 16 개 ...
    case ptyFollow = "pty.follow"
    case ptyWrite  = "pty.write"
    case ptyResize = "pty.resize"
}

// Sources/HamApp/PtyClient.swift (신규)
final class PtyClient {
    func follow(agentID: String) -> AsyncStream<PtyFrame> { ... }
    func write(agentID: String, data: Data) async throws { ... }
    func resize(agentID: String, cols: Int, rows: Int) async throws { ... }
}

// Studio tab view integrates SwiftTerm with PtyClient
```

### ADR-1 cross-reference
PTY 출력은 raw bytes 로 흐르지만, 동시에 hamd 는 line-tee 해서 `SessionEvent` (ADR-1) 로도 기록한다. 즉:
- Studio Tab ← NDJSON PTY stream (raw 사용자 보기 용)
- Session Graph / Inbox / Timeline ← SessionEvent (구조화 된 메타데이터)

동일 데이터의 두 표현을 병행 유지하며, Phase 3 Policy Engine 은 SessionEvent 경로로 구독한다.

### Backward compatibility

1. **기존 request-response IPC 무변경**: 새 3 개 command 추가는 기존 dispatch 테이블에 append 만 한다. 기존 16 개 Swift command / 52 개 Go Command 는 그대로
2. **attached / observed 모드**: PTY 내장은 managed 모드에 한정. 기존에 iTerm/tmux 에 붙어있는 attached 모드와 transcript 감시하는 observed 모드는 legacy fallback 으로 유지. Studio 에서는 "외부 터미널에 이미 세션이 있어요" 탭 타입으로 표시
3. **`ham run` CLI 경로**: 기존 ham CLI 의 로컬 PTY 코드 (pty.go) 는 유지. 사용자가 `ham run` 을 직접 쓰면 기존처럼 본인 터미널이 PTY. Studio 가 열면 hamd 가 PTY 를 소유하고 NDJSON 으로 스트림
4. **PTY 미지원 플랫폼**: Windows 는 원래 scope 밖 (ham-agents 는 macOS 전용). Linux 는 `/dev/ptmx` 공통 지원
5. **마이그레이션 순서**: 기존 사용자는 기존 CLI 경로 그대로 사용 가능. Studio 탭 내장은 opt-in. Phase 2 초기에는 두 경로가 공존

### Failure modes
- **hamd 크래시**: PTY ptmx 파일이 OS 레벨로 닫힘. Claude Code 프로세스는 SIGHUP 받고 종료. Studio 탭은 `pty_exit` 프레임을 못 받고 연결 끊김 감지로 에러 표시 후 재연결 시도 (새 세션으로)
- **Studio 앱 크래시**: ptmx 는 hamd 가 계속 소유. Claude Code 프로세스는 계속 돌아감. 사용자는 Studio 재시작 후 같은 agent_id 로 `CommandFollowPTY` 재구독 (세션 resume)
- **소켓 끊김**: Studio 는 `seq` 를 저장하고 있다가 재연결 시 `resume_from_seq` 로 buffered 프레임을 재생
- **PTY write 실패 (예: Claude Code 종료 중)**: `CommandWritePTY` 가 에러 리턴, Studio 는 탭을 read-only 로 전환

### Spike Results (2026-04-08)

> Ralph Phase 2 spike run. Investigation only — no implementation code. See commits `fc9f62e` (Step 0a) and `1855cb4` (Step 0b) on dev/phase2-spike.

#### Step 0a — SwiftTerm SPM Integration: **PASS**

**방법**: Package.swift 에 SwiftTerm 1.13.0 dependency 추가 + HamAppServices 타겟에 product 링크 후 Swift/Go 빌드 회귀 확인.

**증거**:
- 사용 버전: SwiftTerm 1.13.0 (revision `8e7a1e154f470e19c709a00a8768df348ba5fc43`)
- Transitive: swift-argument-parser 1.7.1
- `swift build --disable-sandbox`: Build complete, 94 SwiftTerm + HamAppServices + HamMenuBarApp 모듈 컴파일 (10.72s)
- `swift test --disable-sandbox --filter HamCoreTests`: 16 tests, 0 failures
- `go build ./go/cmd/ham ./go/cmd/hamd`: clean
- `go test ./... -count=1 -short`: 8 packages PASS
- Platform: 기존 `.macOS(.v13)` 이 SwiftTerm 11+ 요구를 이미 초과 → 변경 없음

**결론**: SwiftTerm SPM 통합에 블로커 없음. P2-1 Embedded PTY Runtime 의 Swift 쪽 터미널 에뮬레이터 의존성을 SwiftTerm 으로 확정 가능.

#### Step 0b — Permission Interception: 2 layers separated (P2-3 진행 가능)

**Verdict — 2 layers separated:**
- **Transport layer: BLOCKS** — hamd IPC request/response cycle is synchronous. Python proxy experiment confirmed server-side delay propagates 1:1 to client (0/500/1000/5000ms matrix, <10ms overhead).
- **Application layer: Fire-and-forget today** — hamd currently returns empty `Response{}` without a PermissionDecision field; no wait primitive in CommandHookPermissionReq handler; runHook emits empty stdout. P2-3 implementation must add the decision payload and wait primitive on top of the existing sync transport.

**P2-3 verdict**: proceed with design as written in ham-studio.md. Required implementation: 4 additions (see below).

**방법**: 
1. 정적 분석으로 `ham hook` CLI → IPC → hamd 체인이 동기인지 확인
2. Claude Code hook 계약 (`docs/external/claude-code-hooks-2026-04-08.md`; fetched 2026-04-08 from https://code.claude.com/docs/en/hooks) 에서 PermissionRequest 응답 semantics 확인
3. 격리 환경 (`HAM_AGENTS_HOME=/tmp/ham-spike-*`) 에서 hamd 띄우고 Python 프록시로 서버 응답 지연 주입하여 IPC client latency 측정
4. 실제 `ham hook permission-request` 서브프로세스를 `time` wrapper 로 직접 계측하여 end-to-end wall-clock 측정

**Claude Code hook 계약**:
- `PermissionRequest` 는 동기 이벤트
- 핸들러가 stdin 으로 JSON 읽고 stdout 으로 `hookSpecificOutput.decision.{behavior, message}` JSON 쓰고 exit 할 때까지 Claude 가 블록
- 기본 timeout: command hooks **600s**, agent hooks 60s
- exit 2 또는 `behavior=deny` → 거절

**hamd 체인 증거**:
| 레이어 | 파일:라인 | 현재 상태 |
|--------|----------|----------|
| CLI 진입 | `go/cmd/ham/commands.go:386-388` `runHook` | 동기 `client.HookPermissionRequest` 호출, exit 0 + 빈 stdout |
| IPC client | `go/internal/ipc/ipc.go:513-516` | 동기 request/response 소켓 |
| IPC server dispatch | `go/internal/ipc/server.go:478-488` `CommandHookPermissionReq` | `RecordHookPermissionRequest` 호출 후 빈 `Response{}` 반환 |
| Registry 상태 | `go/internal/runtime/managed_state.go:632-655` | Status → WaitingInput, 이벤트 기록, 상태 쓰기만 |

→ 전송은 동기이지만 응용 로직은 현재 fire-and-forget (Response 에 decision 필드 없음, hamd 쪽 wait primitive 없음, runHook 이 decision 을 stdout 으로 emit 하지 않음).

**동적 실험 매트릭스 — IPC layer (Python proxy)**:
| 서버 응답 지연 | 클라이언트 관찰 latency | 응답 body |
|---------------|----------------------|----------|
| 0 ms | 0.6 ms | `{}` |
| 500 ms | 502.9 ms | `{}` |
| 1000 ms | 1002.0 ms | `{}` |
| 5000 ms | 5009.9 ms | `{}` |

오버헤드 < 10ms, 서버 지연이 1:1 로 클라이언트까지 전파됨 확인.

**서브프로세스 wall-clock 측정 — ham hook subprocess (실제 생산 경로)**:

방법: `HAM_AGENT_ID=<id> HAM_AGENTS_HOME=/tmp/ham-spike-run3 /tmp/ham-spike hook permission-request Bash < /dev/null` 을 `time` builtin 으로 3회 계측. hamd 정상 기동 상태, 실제 managed agent 등록 후 측정.

| Run | Wall-clock | Exit | Stdout |
|-----|-----------|------|--------|
| 1   | 11 ms     | 0    | (empty) |
| 2   | 9 ms      | 0    | (empty) |
| 3   | 11 ms     | 0    | (empty) |

Mean ~10ms. Go binary startup ~7ms + IPC roundtrip <1ms. 서브프로세스 경로가 완전히 동기임을 실증 — `c.request()` 는 unix socket 위 blocking `json.NewEncoder/Decoder` 교환이며 goroutine detachment 없음. 자세한 분석: `.omc/phase2-step0b-results.md` "Subprocess Wall-Clock Measurement" 섹션 참조.

**결론**: **BLOCKS (transport layer)** — 전송 계층은 이미 동기 블로킹. P2-3 Approval Interception 은 기존 설계 방향대로 진행 가능. 단 아래 **4개** 추가 작업이 필요:

1. **IPC Response 확장**: `go/internal/ipc/ipc.go:118` `Response` 구조체에 `PermissionDecision *PermissionDecision` 필드 추가.
2. **새 IPC 커맨드 `decision.permission`** (Studio → hamd): `{agent_id, request_id, behavior, message}` payload. ham-studio UI 가 사용자 결정을 대기 중인 핸들러에 주입.
3. **hamd wait primitive**: `go/internal/ipc/server.go:478` `CommandHookPermissionReq` 핸들러에서 decision channel 과 context deadline 을 `select` 로 기다림. timeout 에는 no-decision (Claude native dialog fallback) 반환.
4. **runHook decision emit**: `go/cmd/ham/commands.go:386-388` `runHook` 이 permission-request 브랜치에서 hamd 응답의 `PermissionDecision` 이 non-nil 이면 `hookSpecificOutput` JSON 형식으로 stdout 에 쓰고 exit.

**리스크 / 불확실성**:
- 동일 agent 에 동시 permission 요청 발생 시 request_id 로 키잉 필요 (agent_id 만으로는 race)
- Claude Code 가 60s+ hook latency 를 UI fallback 없이 견디는지는 spike 로 실증 불가 — 실제 `claude` 세션에서 수동 smoke test 필요
- Claude Code hook 실행 환경이 PTY 가 아니고 stdin/stdout pipe 기반이므로, P2-1 PTY 런타임 과 P2-3 Permission Interception 은 독립 경로 (PTY 는 렌더용, hook 은 개입용)

#### Phase 2 설계 영향 요약

- P2-1 Embedded PTY Runtime → SwiftTerm 1.13.0 확정 사용, Package.swift 는 이 spike 커밋에서 이미 dependency 선언됨
- P2-3 Approval Interception → 설계 재설계 불필요. 기존 `docs/spec/ham-studio.md` P2-3 섹션 그대로 진행하되, 위 3개 IPC 확장을 태스크로 분해
- ADR-2 Option 1 (NDJSON 스트림) 권장안 유지 — 이 spike 는 ADR-2 결정을 뒤집지 않음

---

## 6-3. 저장소를 event log + read model로 분리

### 현재 상태

**Agent 저장소** (`go/internal/store/store.go`):
- `FileAgentStore`: `managed-agents.json`에 전체 에이전트 목록을 JSON으로 스냅샷 저장
- `LoadAgents()` (`store.go:47-52`): mutex lock → `os.ReadFile` → `json.Unmarshal`
- `SaveAgents()` (`store.go:54-91`): mutex lock → sort → `json.MarshalIndent` → tmpfile write → `os.Rename` (atomic)
- 매 mutation마다 전체 목록을 직렬화/역직렬화

**이벤트 저장소** (`go/internal/store/events.go`):
- `FileEventStore`: `events.jsonl`에 append-only로 이벤트 기록
- `Append()` (`events.go:47-82`): mutex lock → `json.Marshal` → `os.OpenFile(O_APPEND)` → write
- `Load()` (`events.go:115-152`): mutex lock → `os.ReadFile` → line split → `json.Unmarshal` 전체
- 1000 append마다 `truncateLocked()` 호출 → 최근 10,000개만 유지
- `FollowEvents()` (`runtime/events.go:29-64`): 200ms 간격 폴링, 최대 60초 대기

**기타 저장소:**
- `FileSettingsStore` (`store/settings.go`): `settings.json`, atomic write
- `FileTeamStore` (`store/team.go`): `teams.json`, atomic write
- 모든 파일 경로: `~/Library/Application Support/ham-agents/`

### 목표 상태

```
Write Path:
  SessionEvent → EventBus → FileEventStore (JSONL, write-ahead log)
                          → AgentProjector → in-memory state

Read Path:
  Query → ReadModel (in-memory / snapshot file / embedded DB)
```

- **Write-ahead log (WAL)**: events.jsonl가 유일한 source of truth
- **Read model**: AgentProjector가 이벤트 스트림을 소비하여 구축한 뷰
  - Phase 2 초기: in-memory map + 주기적 파일 스냅샷 (현재 managed-agents.json과 동일 형식)
  - Phase 3: embedded DB (BBolt 또는 SQLite) -- 에이전트 수 100+ 또는 이벤트 100K+ 시점에 전환

### 마이그레이션 전략

**단계 1: Read model 인터페이스 정의**

```go
type ReadModel interface {
    Agent(ctx context.Context, id string) (core.Agent, error)
    Agents(ctx context.Context) ([]core.Agent, error)
    AgentsBySession(ctx context.Context, sessionID string) ([]core.Agent, error)
    Events(ctx context.Context, filter EventFilter) ([]core.SessionEvent, error)
    Snapshot(ctx context.Context) (core.RuntimeSnapshot, error)
}
```

**단계 2: InMemoryReadModel 구현**

AgentProjector가 이벤트를 받아 `map[string]*core.Agent`를 갱신한다. 이 map이 ReadModel의 backing store. 주기적으로 (또는 N개 이벤트마다) managed-agents.json에 스냅샷 flush.

**단계 3: Registry를 ReadModel 소비자로 전환**

`Registry.List()`, `Registry.Snapshot()`이 `store.LoadAgents()` 대신 `ReadModel.Agents()`를 호출하도록 변경. `mutateAgent()`는 EventBus.Publish()로 대체 (6-1과 동시 진행).

**단계 4 (Phase 3): Embedded DB 도입**

에이전트 수가 임계치를 넘거나, 이벤트 쿼리 (시간 범위, 세션별 필터링)가 필요해지면 SQLite 또는 BBolt로 전환. ReadModel 인터페이스는 동일하므로 상위 코드 변경 없음.

### Go 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `go/internal/store/readmodel.go` (신규) | `ReadModel` 인터페이스 정의 |
| `go/internal/store/memory_readmodel.go` (신규) | `InMemoryReadModel` 구현 (map + RWMutex) |
| `go/internal/store/events.go` | `FileEventStore`에 WAL 모드 옵션: fsync 주기 설정, 세그먼트 파일 분할 (향후) |
| `go/internal/runtime/registry.go` | `Registry` 구조체에 `readModel ReadModel` 필드 추가 |
| `go/internal/runtime/registry.go` | `List()`, `Snapshot()`, `FindAgentBySessionID()`가 readModel 사용 |
| `go/internal/runtime/projector.go` (신규, 6-1과 공유) | 이벤트 → Agent 상태 매핑 로직 |

### Swift 변경사항

Phase 2에서는 Swift 변경 없음. Read model은 Go 서버 내부 구조이며, IPC 응답 형식은 동일.

### IPC 변경사항

없음. `agents.list`, `agents.status`, `events.list`, `events.follow` 커맨드의 응답 형식이 동일하게 유지된다.

### 하위 호환성

- `managed-agents.json` 파일을 계속 생성한다 (InMemoryReadModel이 주기적으로 flush)
- 기존 형식의 스냅샷 파일이므로, 이전 버전의 hamd로 롤백해도 읽기 가능
- events.jsonl 형식도 유지 (6-2의 additive 필드 추가만 있음)

### 위험 요소

1. **시작 시 상태 복구**: hamd 재시작 시 events.jsonl에서 replay하여 in-memory 상태를 재구축해야 한다. 10,000개 이벤트 replay 시간을 측정하고, 느리면 스냅샷 + 이후 이벤트만 replay하는 전략 필요.
2. **메모리 사용량**: in-memory agent map은 현재 규모 (수십 개 에이전트)에서는 무시할 수준이나, 이벤트 히스토리 전체를 메모리에 올리면 문제. ReadModel은 최근 이벤트만 캐시.
3. **이중 쓰기 위험**: 전환 기간 중 FileAgentStore와 ReadModel이 동시에 존재. 둘의 불일치가 발생하면 어느 쪽이 정본인지 혼란. 전환 완료 전까지 FileAgentStore를 정본으로 유지.
4. **Embedded DB 도입 시점 판단**: 너무 일찍 도입하면 복잡성 증가, 너무 늦으면 성능 문제. "외부 의존성 제로는 초기의 이점이지 장기 원칙이 아니다"라는 원칙에 따라 측정 기반으로 결정.

### 선행 작업

- 6-2 (이벤트 스키마 정규화): SessionEvent 구조체가 확정되어야 projector의 이벤트 → 상태 매핑을 구현할 수 있음

---

## 6-4. IPC를 명령 채널과 스트림 채널로 분리

### 현재 상태

**IPC 프로토콜** (`go/internal/ipc/ipc.go`):
- Unix domain socket (`~/Library/Application Support/ham-agents/hamd.sock`)
- 연결당 하나의 JSON request-response (connect → send JSON → recv JSON → close)
- 52개 `Command` 상수 (`ipc.go:22-74`)
- 평탄한 `Request` 구조체 (`ipc.go:76-116`) -- 모든 커맨드의 필드가 하나의 struct에 합쳐짐
- 평탄한 `Response` 구조체 (`ipc.go:118-129`) -- 모든 응답 타입이 하나의 struct에 합쳐짐

**서버** (`go/internal/ipc/server.go`):
- `handleConnection()` (`server.go:104-126`): goroutine per connection, JSON decode → dispatch → JSON encode → closeWrite
- `dispatch()` (`server.go:136-634`): 거대한 switch 문 (52개 case)
- read deadline: 10초 (`server.go:108`)
- connection 재사용 없음 (매 요청마다 새 연결)

**클라이언트 (Go)** (`go/internal/ipc/ipc.go`):
- `Client.request()` (`ipc.go:578-605`): dial → set deadline → JSON encode → JSON decode
- timeout: 3초 (`ipc.go:163`)

**클라이언트 (Swift)** (`Sources/HamAppServices/DaemonClient.swift`):
- `UnixSocketDaemonTransport.send()` (`DaemonClient.swift:265-341`): BSD socket API 직접 사용
- socket timeout: 90초 (FollowEvents의 60초 long-poll을 수용)
- 연결당 1회 request-response

**폴링 패턴:**
- hamd → iTerm2/tmux: 2초 간격 (`go/cmd/hamd/main.go:109`, `pollRuntimeState` interval)
- Swift VM → hamd: 5초 간격 refresh (`MenuBarViewModel`, `pollIntervalNanoseconds: 5_000_000_000`)
- Swift VM → hamd: 15초 대기 event follow (`eventFollowWaitMilliseconds: 15_000`)
- events.follow: 200ms 폴링 간격, 60초 최대 대기 (`runtime/events.go:27,33`)

### 목표 상태

**명령 채널 (Control Plane)**:
- 기존 JSON request-response 유지 (또는 JSON-RPC 2.0으로 업그레이드)
- 상태 변경 명령: register, remove, rename, settings 등
- 요청-응답 패턴 유지

**스트림 채널 (Data Plane)**:
- 별도 UDS 또는 같은 UDS의 다른 프로토콜
- 서버 → 클라이언트 단방향 이벤트 스트림
- 형식: NDJSON over UDS (가장 단순) 또는 subscribe RPC + streaming response
- 클라이언트가 구독하면 서버가 실시간 이벤트를 push
- 목표: 5초/15초 폴링 제거, UI가 이벤트 드리븐으로 전환

### 마이그레이션 전략

**단계 1: 스트림 프로토콜 선정**

NDJSON over UDS를 1차 구현으로 선정한다:
- 가장 단순 (JSON 라인 write, newline delimiter)
- 기존 Unix socket 인프라 재활용
- SSE는 HTTP 서버가 필요하므로 과잉

프로토콜:
```
클라이언트 → 서버: {"subscribe": "events", "filter": {"agent_id": "..."}}  (1회)
서버 → 클라이언트: {"event": {...}}\n                                      (반복)
서버 → 클라이언트: {"event": {...}}\n
서버 → 클라이언트: {"heartbeat": true}\n                                   (30초 간격 keepalive)
```

**단계 2: 서버 측 StreamServer 구현**

별도 소켓 (`hamd-stream.sock`) 또는 같은 소켓에서 첫 메시지의 `subscribe` 필드로 구분한다. EventBus (6-1)의 subscriber로 등록되어 이벤트를 받아 연결된 클라이언트에 fan-out.

**단계 3: Swift 측 StreamClient 구현**

`UnixSocketDaemonTransport` 옆에 `StreamTransport`를 추가한다. `MenuBarViewModel`의 `eventFollowTask`를 스트림 구독으로 교체. 폴링 fallback은 유지 (스트림 연결 실패 시).

**단계 4: 폴링 제거**

스트림이 안정화되면 `pollIntervalNanoseconds`를 크게 늘리거나 제거하고, 스트림 이벤트만으로 UI를 갱신한다. `events.follow` 커맨드는 레거시 호환용으로 유지.

### Go 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `go/internal/ipc/server.go` (기존 확장) | 기존 server.go 에 PTY case 3 개 추가 (ADR-2 Option 1 방식). 신규 stream.go 파일은 생성하지 않음. `StreamServer` / `StreamSubscription` 로직은 server.go 내 `handleFollowPTY` / `handleWritePTY` / `handleResizePTY` 로 통합 |
| `go/internal/ipc/server.go` | `dispatch()`에서 `subscribe` 커맨드 추가, 또는 별도 소켓으로 분리 |
| `go/cmd/hamd/main.go` | StreamServer 초기화, EventBus subscriber 등록, 소켓 경로 설정 |
| `go/internal/ipc/ipc.go` | (선택) JSON-RPC 2.0 래퍼 -- 기존 Command 상수를 method 이름으로 매핑 |

### Swift 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `Sources/HamAppServices/StreamTransport.swift` (신규) | UDS 연결, NDJSON 읽기 루프, reconnect 로직 |
| `Sources/HamAppServices/DaemonClient.swift` | `HamDaemonClientProtocol`에 `subscribeEvents() -> AsyncStream<AgentEventPayload>` 추가 |
| `Sources/HamAppServices/MenuBarViewModel.swift` | `eventFollowTask`를 `StreamTransport` 기반으로 교체 |
| `Sources/HamAppServices/MenuBarViewModel.swift` | `refreshTask`의 폴링 간격을 30초로 늘림 (스트림이 주 갱신 경로) |

### IPC 변경사항

| 변경 | 상세 |
|------|------|
| 신규 소켓 | `hamd-stream.sock` (또는 기존 소켓에서 프로토콜 분기) |
| 신규 메시지 | `{"subscribe": "events", "filter": {...}}` -- 스트림 구독 |
| 신규 메시지 | `{"event": {...}}` -- 서버 → 클라이언트 이벤트 push |
| 신규 메시지 | `{"heartbeat": true}` -- keepalive |
| 기존 유지 | 52개 Command + Request/Response 구조체 그대로 |

### 하위 호환성

- 기존 명령 채널 완전 유지. 스트림 채널은 순수 추가.
- 기존 `events.follow` 커맨드 유지 (스트림 미지원 클라이언트용)
- Swift 앱은 스트림 연결 실패 시 자동으로 기존 폴링 모드로 fallback
- `ham` CLI는 기존 명령 채널만 사용 (스트림 불필요)

### 위험 요소

1. **연결 수 관리**: 스트림은 장기 연결이므로, 좀비 연결 정리가 필요. heartbeat 응답이 없으면 30초 후 연결 종료.
2. **재연결 시 상태 동기화**: 스트림 연결이 끊겼다 재연결되면 빠진 이벤트가 있을 수 있음. 재연결 시 `last_event_id`를 보내고 서버가 이후 이벤트를 replay.
3. **소켓 경로 증가**: 별도 소켓을 쓰면 경로가 2개로 늘어남. 같은 소켓에서 프로토콜 분기가 더 깔끔하지만 구현이 복잡.
4. **fan-out 병목**: 연결된 클라이언트가 많을 때 (실제로는 1-3개) 각 클라이언트에 write하는 시간이 이벤트 처리를 지연시킬 수 있음. 비동기 write 채널로 해결.

### 선행 작업

- 6-1 (EventBus): 스트림 채널은 EventBus의 subscriber로 구현됨
- 6-3 (저장소 분리): 재연결 시 이벤트 replay를 위해 ReadModel의 이벤트 쿼리 기능 필요

---

## 6-5. Claude Code 어댑터를 계층화

### 현재 상태

Claude Code와의 연동 경로:

**1. Hooks (현재 주 경로)**
- IPC server의 `CommandHook*` 핸들러 26개 (`server.go:324-619`)
- `prepareHookRequest()` (`server.go:637-665`): sessionID로 agent 찾기, 없으면 auto-register
- hook → hamd IPC → `Registry.RecordHook*()` → agent 상태 갱신 + 이벤트 기록
- 신뢰도 높음 (직접 이벤트), 지연 낮음

**2. iTerm2 폴링 (Attached)**
- `adapters.NewIterm2Adapter()` → AppleScript로 iTerm2 세션 목록 조회
- `Registry.RefreshAttachedByScheme()` (`runtime/registry_attached.go`): 세션 정보 → agent 상태 추론
- 2초 간격 폴링 (`hamd/main.go:109`)
- 신뢰도 중간 (프로세스 이름, 활동 상태로 추론)

**3. tmux 폴링 (Attached)**
- `adapters.NewTmuxAdapter()` → `tmux list-panes` 등으로 세션 조회
- 같은 `RefreshAttachedByScheme()` 경로
- 2초 간격 폴링

**4. Transcript 관찰 (Observed)**
- `adapters.NewTranscriptAdapter()` → 파일 시스템에서 transcript 파일 discover
- `Registry.RefreshObserved()` → 파일 내용 파싱 → 상태 추론
- `inference.ObservedInference` (`inference/observed.go`): 텍스트 패턴 매칭으로 상태 추론
- 신뢰도 낮음 (로그 파싱 기반)

### 목표 상태

어댑터를 우선순위 계층으로 정리한다:

```
Layer 1: Hooks (최고 신뢰도, 최저 지연)
  - Claude Code hooks → IPC → SessionEvent(source: "hook", confidence: 0.95+)
  - 현재 구현 유지, 스키마만 6-2에 맞춰 확장

Layer 2: PTY/Attach (높은 신뢰도, 실시간)
  - iTerm2 AppleScript / tmux command → SessionEvent(source: "pty", confidence: 0.7-0.9)
  - PTY output 직접 읽기 (향후): 더 풍부한 상태 추론

Layer 3: Observe/Log (중간 신뢰도, 비동기)
  - Transcript 파일 파싱 → SessionEvent(source: "log", confidence: 0.3-0.6)
  - Claude Code JSONL 세션 로그 파싱 (향후)

Layer 4 (선택): Channels/Scheduled/Remote
  - Claude Code channels API (향후, "nice-to-have")
  - Scheduled tasks webhook
  - Remote session 관찰
```

**Confidence 통합 규칙**: 같은 agent에 대해 여러 Layer의 이벤트가 도착하면:
- 높은 Layer의 이벤트가 우선 (hook > pty > log)
- 같은 Layer 내에서는 최신 타임스탬프 우선
- Confidence 값이 임계치 미만이면 UI에 "추정" 표시

### 마이그레이션 전략

**단계 1 (Phase 1): 기존 어댑터에 Source/Confidence 태깅**

각 어댑터의 이벤트 생성 코드에 `Source`와 `Confidence` 필드를 추가한다. 코드 변경 최소.

**단계 2 (Phase 2): 어댑터 인터페이스 통합**

```go
type Adapter interface {
    Name() string
    Source() string  // "hook", "pty", "log"
    Start(ctx context.Context, bus EventBus) error
    Stop() error
}
```

모든 어댑터가 EventBus에 직접 publish하도록 통합. Registry가 어댑터별 분기 로직을 갖는 대신, EventBus + Projector가 Source 필드 기반으로 우선순위 처리.

**단계 3 (Phase 3): PTY 어댑터 강화**

iTerm2/tmux의 PTY output을 직접 읽어 더 정확한 상태 추론. 현재 프로세스 이름/활동 상태만 보는 것에서, 실제 출력 텍스트를 분석하는 것으로 강화.

### Go 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `go/internal/adapters/adapter.go` (신규) | `Adapter` 인터페이스 정의 |
| `go/internal/adapters/hook_adapter.go` (신규) | 기존 IPC hook 핸들링을 Adapter 인터페이스로 래핑 |
| `go/internal/adapters/iterm2.go` | `Source()` 반환, EventBus publish 추가 |
| `go/internal/adapters/tmux.go` | `Source()` 반환, EventBus publish 추가 |
| `go/internal/adapters/transcript.go` | `Source()` 반환, EventBus publish 추가 |
| `go/internal/runtime/registry_attached.go` | `RefreshAttachedByScheme()`에서 Source/Confidence 태깅 |
| `go/internal/runtime/registry_observed.go` | `RefreshObserved()`에서 Source/Confidence 태깅 |
| `go/internal/inference/observed.go` | 추론 결과에 Confidence 값 포함 |

### Swift 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `Sources/HamCore/Agent.swift` | (이미 `statusConfidence` 존재) `source` 필드 추가 |
| `Sources/HamAppServices/MenuBarViewModel.swift` | `confidenceLevelText()` 로직에 source 기반 표시 추가 |

### IPC 변경사항

없음. 어댑터 계층화는 서버 내부 구조 변경.

### 하위 호환성

- 기존 `ham attach`, `ham observe` CLI 명령 동일 동작
- 기존 iTerm2/tmux 폴링 경로 유지
- Agent의 `Mode` 필드 (managed/attached/observed) 유지 -- `Source` 필드는 추가이지 대체가 아님

### 위험 요소

1. **Confidence 충돌**: hook 이벤트와 PTY 이벤트가 동시에 도착하면 어느 쪽이 맞는지 판단 로직이 복잡해짐. 단순 규칙 (hook always wins) 부터 시작.
2. **PTY 읽기 권한**: macOS에서 다른 프로세스의 PTY를 직접 읽으려면 특수 권한이 필요할 수 있음. iTerm2 AppleScript 경유는 계속 작동하지만 직접 PTY 읽기는 제한적.
3. **어댑터 인터페이스 통합 비용**: 기존 어댑터 코드가 Registry에 직접 의존하므로, EventBus로의 전환이 큰 리팩터링이 될 수 있음.

### 선행 작업

- 6-2 (이벤트 스키마): Source, Confidence 필드가 정의되어야 어댑터가 태깅 가능

---

## 6-6. UI를 2계층으로 확장

### 현재 상태

**메뉴바 앱** (`apps/macos/HamMenuBarApp/`):
- `HamMenuBarApp.swift`: SwiftUI App, menu bar extra
- `MenuBarViews.swift`: 메뉴바 팝오버 뷰
- `PixelOfficeView.swift`: 픽셀 오피스 (햄스터 애니메이션)
- `MenuBarPlatform.swift`: 플랫폼별 설정

**ViewModel** (`Sources/HamAppServices/MenuBarViewModel.swift`):
- 934줄의 대형 ViewModel
- `refresh()` (`MenuBarViewModel.swift:491-526`): 5초 폴링으로 snapshot + agents + settings + sessions 동시 조회
- `followLatestEvents()` (`MenuBarViewModel.swift:528-557`): 15초 대기 long-poll
- `statusBarTint` (`MenuBarViewModel.swift:28-34`): 상태 색상 (red/yellow/blue/green/gray)
- `statusLine` (`MenuBarViewModel.swift:101-107`): "ham 2▶ 1? 3✓" 형식
- 알림 엔진 (`StatusChangeNotificationEngine`), 필터링, 기록 관리
- 팀 필터링, 워크스페이스 필터링
- 픽셀 오피스 매핑 (`PixelOfficeMapper`)

**DaemonClient** (`Sources/HamAppServices/DaemonClient.swift`):
- `UnixSocketDaemonTransport`: BSD socket API로 UDS 통신
- `HamDaemonClientProtocol`: 11개 메서드 (fetchSnapshot, fetchAgents, followEvents 등)
- `MenuBarSummaryService`: snapshot + events 조합

### 목표 상태

**메뉴바 (Layer 1)**: 빠른 상태 확인
- 배지, 색상 코딩, 긴급 승인 알림
- 현재 기능의 경량화 버전 유지
- 픽셀 오피스(ambient UI)는 메뉴바에 잔류

**ham Studio (Layer 2)**: 전체 운영 화면
- 타임라인 뷰: 세션별 이벤트 타임라인
- Diff 뷰: 에이전트가 변경한 파일 목록 + diff
- 승인 큐: pending 승인 목록 + approve/deny
- 비용 대시보드: 세션별/프로젝트별 비용
- 세션 리플레이: 과거 세션의 이벤트 재생
- **에디터를 만들지 않는다**: Claude Code는 이미 VS Code/Desktop 표면을 가짐

### 마이그레이션 전략

**단계 1: MenuBarViewModel 분할**

현재 934줄 ViewModel을 역할별로 분리:
- `AgentStateStore`: agent 목록 관리, 필터링
- `EventStreamStore`: 이벤트 수신/캐싱
- `NotificationStore`: 알림 엔진, 필터, 히스토리
- `SettingsStore`: 설정 CRUD
- `MenuBarViewModel`: 위 store들을 조합하는 경량 facade

**단계 2: 스트림 기반 갱신**

6-4의 StreamTransport가 완성되면 `EventStreamStore`가 스트림을 구독. 폴링은 fallback 전용.

**단계 3: ham Studio 쉘**

별도 macOS 윈도우 앱 (또는 메뉴바 앱의 별도 윈도우)으로 ham Studio를 시작한다. 초기에는 타임라인 뷰 + 에이전트 상세 패널.

**단계 4: Studio 기능 확장**

Diff 뷰, 승인 큐, 비용 대시보드를 순차 추가. 각 기능은 독립된 SwiftUI View + 전용 Store.

### Go 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `go/internal/ipc/ipc.go` | Studio 전용 커맨드 추가: `studio.timeline`, `studio.diff`, `studio.approvals`, `studio.cost` |
| `go/internal/ipc/server.go` | 새 커맨드에 대한 dispatch 핸들러 |
| `go/internal/runtime/registry.go` | (선택) Timeline 쿼리를 위한 이벤트 필터링 API |

### Swift 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `Sources/HamAppServices/MenuBarViewModel.swift` | ViewModel을 AgentStateStore, EventStreamStore 등으로 분할 |
| `Sources/HamAppServices/AgentStateStore.swift` (신규) | Agent 목록/필터링 전용 ObservableObject |
| `Sources/HamAppServices/EventStreamStore.swift` (신규) | 이벤트 수신/캐싱, StreamTransport 연동 |
| `Sources/HamAppServices/NotificationStore.swift` (신규) | 알림 엔진 분리 |
| `Sources/HamStudio/` (신규 패키지) | ham Studio 뷰: TimelineView, DiffView, ApprovalQueueView, CostDashboardView |
| `apps/macos/HamMenuBarApp/Sources/StudioWindow.swift` (신규) | Studio 윈도우 열기/관리 |

### IPC 변경사항

| 변경 | 상세 |
|------|------|
| 신규 커맨드 | `studio.timeline` -- 세션별 이벤트 타임라인 조회 |
| 신규 커맨드 | `studio.cost` -- 비용 집계 조회 |
| 스트림 활용 | Studio는 스트림 채널 (6-4)을 적극 활용하여 실시간 갱신 |

### 하위 호환성

- 메뉴바 앱의 기존 기능 100% 유지
- ham Studio는 순수 추가 기능
- ViewModel 분할은 내부 리팩터링이며 외부 동작 변화 없음

### 위험 요소

1. **ViewModel 분할 회귀**: 934줄 ViewModel을 분할하면서 기존 동작이 깨질 수 있음. 기존 테스트 (`MenuBarViewModelTests.swift`) 유지가 필수.
2. **Studio 범위 팽창**: "에디터를 만들지 않는다"는 원칙을 지키기 어려울 수 있음. Diff 뷰가 점차 에디터로 진화하는 것을 경계.
3. **2개 윈도우 간 상태 동기화**: 메뉴바와 Studio가 같은 데이터를 보여주되, 갱신 타이밍이 다르면 혼란. 공유 Store 계층으로 해결.

### 선행 작업

- 6-4 (스트림 채널): Studio의 실시간 갱신에 스트림이 필수

---

## 6-7. 확장성은 Claude 생태계에 맞춘다

### 현재 상태

ham-agents는 자체 확장 메커니즘이 없다. 모든 기능이 Go/Swift 코드에 하드코딩되어 있다.

**현재 설정 구조** (`go/internal/core/settings.go`, `store/settings.go`):
- `Settings` 구조체: Notifications, Appearance, General, Integrations, Privacy 섹션
- `settings.json`에 JSON으로 저장
- iTerm 연동 on/off, transcript 디렉터리, provider adapter 토글 정도

**현재 Claude Code 연동:**
- hooks만 사용 (26개 hook 타입)
- MCP, skills, plugins, channels 미활용
- Claude Code의 `settings.json`에 hook 설정을 주입하는 `ham setup` 명령 (`go/cmd/ham/setup.go`)

### 목표 상태

**Playbook 포맷**: Claude Code의 skills/plugins와 호환되는 확장 단위

```yaml
# example: code-review-playbook.yaml
name: code-review
version: 1.0
triggers:
  - hook: tool-done
    filter: {tool_name: "Edit"}
actions:
  - type: ham.ui_pane
    config: {view: "diff", auto_open: true}
  - type: ham.analytics
    config: {track: "edit_count", group_by: "session"}
  - type: ham.policy
    config: {require_approval: false, cost_limit_usd: 5.0}
```

**확장 유형 (ham 고유):**
1. **UI Pane**: ham Studio에 커스텀 패널 추가
2. **Analytics Pack**: 이벤트 스트림에서 메트릭 추출/집계
3. **Policy Pack**: 승인 규칙, 비용 제한, 접근 제어

**Claude 생태계 호환:**
- Playbook 파일이 Claude Code skills 디렉터리에도 로드 가능
- ham-specific 액션 (`ham.ui_pane`, `ham.analytics`, `ham.policy`)은 Claude Code에서 무시됨
- Claude Code의 standard 액션은 ham에서도 실행 가능

### 마이그레이션 전략

**단계 1: Playbook 스키마 정의**

YAML/JSON 스키마를 정의한다. Claude Code skills 포맷과의 교집합을 최대화.

**단계 2: Playbook 로더**

hamd에 Playbook 로더를 추가. 설정된 디렉터리에서 Playbook 파일을 읽고, trigger 조건에 따라 EventBus subscriber로 등록.

**단계 3: ham 고유 액션 엔진**

`ham.ui_pane`, `ham.analytics`, `ham.policy` 액션의 실행 엔진. 각 액션 타입은 인터페이스로 정의되어 확장 가능.

### Go 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `go/internal/playbook/schema.go` (신규) | Playbook YAML/JSON 스키마 정의, 파싱 |
| `go/internal/playbook/loader.go` (신규) | 디렉터리 스캔, 파일 워치, Playbook 로드 |
| `go/internal/playbook/engine.go` (신규) | trigger 매칭, action 실행 |
| `go/internal/playbook/actions/` (신규 디렉터리) | 각 ham 고유 액션 타입 구현 |
| `go/internal/core/settings.go` | `Settings.Playbooks` 필드 추가 (Playbook 디렉터리 경로 등) |

### Swift 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `Sources/HamStudio/PlaybookPaneView.swift` (신규) | Playbook이 정의한 UI 패널 렌더링 |
| `Sources/HamAppServices/SettingsStore.swift` | Playbook 설정 UI |

### IPC 변경사항

| 변경 | 상세 |
|------|------|
| 신규 커맨드 | `playbooks.list` -- 로드된 Playbook 목록 |
| 신규 커맨드 | `playbooks.reload` -- Playbook 디렉터리 재스캔 |
| 신규 커맨드 | `playbooks.execute` -- 수동 Playbook 실행 |

### 하위 호환성

- Playbook은 순수 추가 기능. Playbook이 없으면 기존 동작과 동일.
- 기존 `settings.json`에 `playbooks` 섹션이 추가되지만, 없어도 기본값 사용.

### 위험 요소

1. **스키마 안정성**: Playbook 포맷이 확정되기 전에 사용자가 작성한 Playbook이 있으면, 스키마 변경 시 호환성 문제. 1.0까지는 "unstable" 표시.
2. **Claude Code 생태계 변동**: Claude Code의 skills/plugins 포맷이 변경되면 호환성 유지 비용 발생. 교집합을 최소화하고, ham 고유 부분을 명확히 분리.
3. **보안**: Playbook에서 임의 명령 실행을 허용하면 보안 위험. action 타입을 화이트리스트로 제한.

### 선행 작업

- 6-2 (이벤트 스키마): Playbook trigger가 SessionEvent 필드를 참조
- 6-6 (UI 2계층): UI Pane 액션이 ham Studio 프레임워크에 의존

---

## 6-8. 품질 전략을 제품 기능으로

### 현재 상태

**테스트 현황:**
- `go/internal/runtime/registry_test.go`: Registry 단위 테스트
- `go/internal/store/store_test.go`: FileAgentStore 단위 테스트
- `go/internal/store/events_test.go`: (존재 여부 확인 필요)
- `go/internal/store/team_test.go`: Team store 테스트
- `go/internal/ipc/ipc_test.go`: IPC 클라이언트/서버 테스트
- `go/internal/runtime/managed_test.go`: Managed service 테스트
- `go/internal/adapters/iterm2_test.go`, `tmux_test.go`, `transcript_test.go`: 어댑터 테스트
- `go/internal/inference/observed_test.go`: 관찰 추론 테스트
- Swift 측: `Tests/HamAppServicesTests/`, `Tests/HamCoreTests/`, `Tests/HamNotificationsTests/`

**부족한 영역:**
- hook schema contract test 없음 (Go/Swift 간 직렬화 호환성 미검증)
- 통합 테스트 없음 (hamd → IPC → Registry → Store 전체 경로)
- 부하/성능 테스트 없음
- chaos test 없음 (연결 끊김, 파일 잠금 충돌 등)
- golden trace 없음 (이벤트 시퀀스 회귀 테스트)

### 목표 상태

| 테스트 카테고리 | 설명 | 자동화 수준 |
|----------------|------|-------------|
| Hook Schema Contract | Go IPC Request ↔ Swift DaemonRequest 직렬화 왕복 | CI 필수 |
| Synthetic Session Replayer | 녹화된 hook 이벤트 시퀀스를 재생하여 상태 전이 검증 | CI 필수 |
| Attach/Detach Chaos | 무작위 연결/해제/재연결 시나리오 | CI 선택 |
| Golden Traces | 알려진 정상 이벤트 시퀀스에 대한 스냅샷 테스트 | CI 필수 |
| Cost/Approval/Status Reducer 회귀 | 집계/정책 로직의 입출력 고정 | CI 필수 |
| 성능 벤치마크 | mutateAgent 처리량, 이벤트 처리 지연 | 주기적 |

### 마이그레이션 전략

**Phase 1 (즉시):**
- Hook Schema Contract Test: Go에서 모든 Command에 대해 Request를 직렬화하고, Swift에서 같은 JSON을 디코딩하는 테스트. 공유 JSON 파일로 검증.
- Synthetic Session Replayer: `testdata/sessions/` 디렉터리에 JSONL 형식의 hook 시퀀스를 저장. 테스트가 이를 재생하며 Registry 상태를 검증.

**Phase 2 (6-1, 6-3과 동시):**
- Attach/Detach Chaos Test: goroutine 100개에서 동시 attach/detach/reconnect, 최종 상태 일관성 검증.
- Golden Traces: 특정 시나리오 (단일 세션 시작→작업→종료, 팀 작업, 에러 복구)의 이벤트 시퀀스를 golden file로 저장.

**Phase 3 (6-6과 동시):**
- Cost Reducer Test: 이벤트 스트림 → 비용 집계의 입출력 고정 테스트.
- Approval Reducer Test: 승인 상태 전이의 입출력 고정 테스트.
- Status Reducer Test: 이벤트 → agent 상태 전이의 입출력 고정 테스트.

### Go 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `go/internal/ipc/contract_test.go` (신규) | 모든 Command에 대한 Request/Response 직렬화 왕복 테스트 |
| `go/internal/runtime/replay_test.go` (신규) | Synthetic Session Replayer |
| `go/testdata/sessions/` (신규 디렉터리) | 녹화된 hook 시퀀스 JSONL 파일 |
| `go/internal/runtime/chaos_test.go` (신규) | Attach/Detach chaos test |
| `go/testdata/golden/` (신규 디렉터리) | Golden trace 파일 |
| `go/internal/runtime/golden_test.go` (신규) | Golden trace 스냅샷 비교 |
| `go/internal/runtime/benchmarks_test.go` | (기존) mutateAgent 벤치마크 확장 |

### Swift 변경사항

| 파일 | 변경 내용 |
|------|-----------|
| `Tests/HamCoreTests/ContractTests.swift` (신규) | Go 측 contract test와 같은 JSON 파일을 사용한 디코딩 검증 |
| `Tests/HamAppServicesTests/ReplayTests.swift` (신규) | Synthetic session 이벤트 재생 → ViewModel 상태 검증 |

### IPC 변경사항

없음. 테스트 인프라는 기존 IPC를 검증하는 것이지 변경하는 것이 아님.

### 하위 호환성

해당 없음. 테스트만 추가되며 프로덕션 코드 변경 없음 (테스트가 발견한 버그 수정은 별도).

### 위험 요소

1. **Contract Test 유지 비용**: Go와 Swift 양쪽에서 공유 JSON fixture를 관리해야 함. 한쪽만 업데이트하면 다른 쪽 테스트 실패.
2. **Golden Trace 깨짐**: 이벤트 스키마 변경 (6-2) 시 golden file이 대량 업데이트 필요. 자동 업데이트 스크립트 필요.
3. **Chaos Test 비결정성**: 동시성 테스트가 환경에 따라 flaky할 수 있음. 재현 가능한 시드 기반 무작위화.

### 선행 작업

- 없음. 품질 전략은 모든 단계에 병렬로 진행. 각 항목의 테스트가 해당 항목 구현과 동시에 작성.

---

## 성능 고려사항

### 현재 병목

**1. mutateAgent의 전체 로드/세이브** (`registry.go:251-286`)
```
Lock → os.ReadFile(managed-agents.json)       // 전체 파일 읽기
     → json.Unmarshal(전체 에이전트 목록)       // O(N) 역직렬화
     → Find agent by ID                        // O(N) 선형 탐색
     → Mutate                                  // O(1)
     → json.MarshalIndent(전체 에이전트 목록)   // O(N) 직렬화
     → os.WriteFile(tmpfile) + os.Rename       // 전체 파일 쓰기
     → eventStore.Append                       // O(1) append
     → Unlock
```

에이전트 10개 기준: 매 hook 이벤트마다 ~1ms (디스크 캐시 hit 시), ~10ms (cold).
에이전트 100개 기준: ~5-10ms (hot), JSON 크기 증가로 I/O 비용 상승.

**2. FileEventStore.Load()의 전체 읽기** (`events.go:115-152`)
```
Lock → os.ReadFile(events.jsonl)   // 10,000줄 전체 읽기
     → line split                  // O(N) 바이트 스캔
     → json.Unmarshal per line     // O(N) 역직렬화
     → Unlock
```

10,000개 이벤트 기준: ~50-100ms. `events.follow`에서 200ms 간격 폴링이므로 이벤트 수가 많아지면 실질적 지연.

**3. 폴링 오버헤드**
- iTerm2 AppleScript: 세션당 ~100ms (AppleScript 실행 비용)
- tmux: ~10ms (subprocess 호출)
- Swift VM 5초 폴링: 매 5초마다 snapshot + agents + settings + sessions 4개 요청 (4 UDS round-trip)

### 개선 계획

| 단계 | 개선 | 예상 효과 |
|------|------|-----------|
| Phase 2 | In-memory agent map (6-1 AgentProjector) | mutateAgent에서 LoadAgents 제거, O(1) lookup |
| Phase 2 | EventBus fan-out (6-1) | events.follow 폴링 제거 |
| Phase 2 | 스트림 채널 (6-4) | Swift 5초 폴링 제거 |
| Phase 3 | Embedded DB (6-3) | 이벤트 범위 쿼리 O(log N), 전체 로드 불필요 |
| Phase 3 | 이벤트 세그먼트 파일 | events.jsonl 분할로 Load 범위 축소 |

---

## 저장소 진화 계획

### 현재: 파일 기반

```
~/Library/Application Support/ham-agents/
├── managed-agents.json    (FileAgentStore, 전체 스냅샷)
├── events.jsonl           (FileEventStore, append-only, max 10K)
├── settings.json          (FileSettingsStore, atomic write)
├── teams.json             (FileTeamStore, atomic write)
└── hamd.sock              (IPC 소켓)
```

장점: 외부 의존성 제로, 디버깅 용이 (JSON 직접 읽기), 설치 단순.

### Phase 2: 파일 + 스냅샷

```
~/Library/Application Support/ham-agents/
├── events.jsonl           (WAL, source of truth)
├── snapshot.json          (AgentProjector가 주기적으로 flush)
├── managed-agents.json    (레거시 호환, snapshot.json의 복사본)
├── settings.json
├── teams.json
└── hamd.sock
```

변화: Agent 상태의 source of truth가 managed-agents.json에서 events.jsonl + in-memory projector로 이동. managed-agents.json은 호환용으로 유지.

### Phase 3: Embedded DB 전환 조건

다음 중 하나라도 해당되면 embedded DB (SQLite 또는 BBolt) 도입:

| 조건 | 임계치 | 이유 |
|------|--------|------|
| 동시 에이전트 수 | 100+ | JSON 전체 직렬화 비용 |
| 이벤트 누적량 | 100K+ | JSONL 전체 로드 비용 |
| 이벤트 쿼리 복잡도 | 시간 범위 + 필터 | 선형 스캔 불가 |
| 비용 집계 | 세션별/프로젝트별 | 인덱스 필요 |

**DB 선택 기준:**
- SQLite: 복잡한 쿼리 (JOIN, GROUP BY) 필요 시. 비용 집계, 타임라인 분석에 적합.
- BBolt: 단순 key-value 접근이면 충분할 때. 낮은 오버헤드.

"외부 의존성 제로는 초기의 이점이지 장기 원칙이 아니다." -- 측정 기반으로 전환 시점을 결정한다.

#### P3-0 Data Loss Risk Matrix

| 위험 | 영향 | 완화 |
|------|------|------|
| 마이그레이션 중 정전 | SQLite 반쯤 채워진 상태, JSONL 원본 소실 | 원본 JSONL 을 `.bak` 로 복사 후 마이그레이션, 완료 확인 전까지 원본 유지 |
| 스키마 mismatch | 특정 event type 누락 | dry-run 모드: 첫 pass 에서 개수/스키마만 검증, 실제 쓰기는 2nd pass |
| 트랜잭션 경계 오류 | 부분 커밋으로 데이터 일관성 깨짐 | 파일 단위 BEGIN...COMMIT, 각 JSONL 파일마다 독립 트랜잭션 |
| rollback 경로 | SQLite 포기 시 JSONL 로 복귀 불가 | 완료 7일간 원본 JSONL 보관, 이후 사용자 확인 받고 삭제 |

**Dry-run 절차**:
1. `ham migrate --dry-run` — 읽기 전용으로 스키마/개수 검증, SQLite 쓰지 않음
2. 검증 통과 시 `ham migrate --commit` — 백업 후 2nd pass 쓰기
3. 완료 후 `ham migrate --verify` — SQLite 과 원본 JSONL 의 count match 확인

#### Multi-process settings.json Lock (H-11 대응)

여러 프로세스(hamd, ham CLI, Swift UI의 settings refresh)가 `~/.ham/settings.json` 에 동시 접근할 때 쓰기 충돌 가능.

**현재 상태**: Go 측은 단일 프로세스 내 mutex 만 존재. 프로세스 간 락 없음.

**제안**:
- 파일 단위 advisory lock (`flock(LOCK_EX)`) 도입. Go 는 `golang.org/x/sys/unix.Flock`, Swift 는 `flock(fd, LOCK_EX)` 직접 호출.
- 락 파일 경로: `~/.ham/settings.json.lock`
- 락 획득 실패 시 최대 500ms 대기 후 에러 리턴, 호출자는 retry 1회
- 읽기는 LOCK_SH, 쓰기는 LOCK_EX

**대안**: 단일 writer (hamd 전용) 로 제한하고, ham CLI / Swift UI 는 IPC 경유로만 settings 수정. 이 경우 락 불필요.

**결정**: Phase 1 에서 "단일 writer" 방식 채택 권장. ham CLI 의 `ham settings set` 도 IPC 경유로 변경.

---

## 커뮤니티 접근법 비교

### claude-squad

- **아키텍처**: tmux 세션 + git worktree per agent
- **상태 관리**: tmux 세션 자체가 상태 (프로세스 존재 = 실행 중)
- **UI**: TUI (터미널 UI)
- **장점**: 단순, 프로세스 격리, git 충돌 회피
- **한계**: 이벤트 히스토리 없음, 비용 추적 없음, 거버넌스 없음
- **ham-agents와의 차이**: claude-squad는 "동시 실행" 문제를 풀고, ham-agents는 "관측+조율+디버깅" 문제를 품

### AMUX (Agent Multiplexer)

- **아키텍처**: tmux + 웹 대시보드
- **상태 관리**: tmux 세션 모니터링 + 웹소켓 실시간 갱신
- **UI**: 웹 브라우저 기반 대시보드
- **장점**: 크로스 플랫폼, 웹소켓 실시간
- **한계**: 브라우저 의존, 네이티브 통합 없음 (메뉴바, 알림 등)
- **ham-agents와의 차이**: AMUX는 웹 UI, ham-agents는 네이티브 macOS. AMUX의 웹소켓 접근은 ham-agents의 스트림 채널 (6-4)과 유사한 목표.

### ham-agents의 차별화

| 축 | claude-squad | AMUX | ham-agents (목표) |
|-----|-------------|------|-------------------|
| 실행 관리 | tmux worktree | tmux | hooks + PTY + log (계층화) |
| 상태 추적 | 프로세스 존재 여부 | 세션 모니터링 | 이벤트 기반 (confidence 포함) |
| UI | TUI | 웹 대시보드 | 네이티브 macOS (메뉴바 + Studio) |
| 이벤트 히스토리 | 없음 | 제한적 | JSONL WAL + ReadModel |
| 비용 추적 | 없음 | 없음 | SessionEvent.CostInfo |
| 거버넌스 | 없음 | 없음 | ApprovalState + PolicyEngine |
| 확장성 | 없음 | 없음 | Playbook 포맷 |
| Claude Code 통합 깊이 | 낮음 (tmux) | 중간 (세션 모니터링) | 높음 (hooks 26종 + 스키마 정규화) |

---

## 부록: 파일 참조 인덱스

### Go 핵심 파일

| 파일 | 역할 | 주요 타입/함수 |
|------|------|---------------|
| `go/internal/core/agent.go` | 도메인 모델 | `Agent`, `Event`, `AgentMode`, `AgentStatus`, `EventType`, `RuntimeSnapshot` |
| `go/internal/runtime/registry.go` | 상태 관리 핵심 | `Registry`, `mutateAgent()`, `appendEvent()`, `saveAgentsAndEvents()` |
| `go/internal/runtime/registration.go` | 에이전트 등록 | `RegisterManaged()`, `RegisterAttached()`, `RegisterObserved()` |
| `go/internal/runtime/events.go` | 이벤트 조회 | `Events()`, `FollowEvents()`, `eventPresentationHint()` |
| `go/internal/runtime/managed_state.go` | hook 이벤트 처리 | `RecordHookToolStart()`, `RecordHookStop()`, ... (26개 RecordHook* 함수) |
| `go/internal/runtime/registry_attached.go` | Attached 리프레시 | `RefreshAttachedByScheme()` |
| `go/internal/runtime/registry_observed.go` | Observed 리프레시 | `RefreshObserved()` |
| `go/internal/store/store.go` | Agent 영속 | `FileAgentStore`, `LoadAgents()`, `SaveAgents()` |
| `go/internal/store/events.go` | Event 영속 | `FileEventStore`, `Append()`, `Load()`, `truncateLocked()` |
| `go/internal/ipc/ipc.go` | IPC 프로토콜 | `Command` (52개), `Request`, `Response`, `Client` |
| `go/internal/ipc/server.go` | IPC 서버 | `Server`, `dispatch()` (52-case switch), `prepareHookRequest()` |
| `go/cmd/hamd/main.go` | 데몬 진입점 | `run()`, `pollRuntimeState()` |
| `go/internal/adapters/iterm2.go` | iTerm2 어댑터 | `Iterm2Adapter`, `ListSessions()` |
| `go/internal/adapters/tmux.go` | tmux 어댑터 | `TmuxAdapter`, `ListSessions()` |
| `go/internal/adapters/transcript.go` | Transcript 어댑터 | `TranscriptAdapter`, `Discover()` |
| `go/internal/inference/observed.go` | 관찰 추론 | `ObservedInference` |

### Swift 핵심 파일

| 파일 | 역할 | 주요 타입 |
|------|------|-----------|
| `Sources/HamCore/Agent.swift` | Agent 모델 | `Agent`, `AgentStatus`, `AgentMode` |
| `Sources/HamCore/DaemonPayloads.swift` | IPC 페이로드 | `DaemonRequest`, `DaemonResponse`, `AgentEventPayload` |
| `Sources/HamCore/DaemonIPC.swift` | IPC 프로토콜 정의 | `DaemonCommand` (16개) |
| `Sources/HamAppServices/DaemonClient.swift` | 데몬 클라이언트 | `HamDaemonClient`, `UnixSocketDaemonTransport`, `MenuBarSummaryService` |
| `Sources/HamAppServices/MenuBarViewModel.swift` | 메뉴바 VM | `MenuBarViewModel` (934줄), `refresh()`, `followLatestEvents()` |
| `Sources/HamAppServices/EventPresentation.swift` | 이벤트 표시 | `AgentEventPresenter` |
| `Sources/HamAppServices/PixelOfficeModel.swift` | 픽셀 오피스 | `PixelOfficeMapper` |
| `Sources/HamNotifications/HamNotificationService.swift` | 알림 | `StatusChangeNotificationEngine` |
| `apps/macos/HamMenuBarApp/Sources/HamMenuBarApp.swift` | 앱 진입점 | SwiftUI App |
| `apps/macos/HamMenuBarApp/Sources/MenuBarViews.swift` | 메뉴바 뷰 | 팝오버 UI |
| `apps/macos/HamMenuBarApp/Sources/PixelOfficeView.swift` | 픽셀 오피스 뷰 | 햄스터 애니메이션 |
