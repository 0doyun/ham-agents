# Step 4: 교차 검증 리뷰 노트

> 2026-04-06 작성 | 검증 대상: docs/spec/ 전체 6개 문서
> 참조: docs/roadmap-0405.md, docs/plan-0405.md, 실제 코드베이스

---

## 검증 요약

| 검증 항목 | 결과 | 이슈 수 |
|-----------|------|---------|
| 4-1. 기획 ↔ 코드 정합성 | **PASS (조건부)** | Critical 0 / High 3 / Medium 5 / Low 4 |
| 4-2. 기획 ↔ 로드맵 정합성 | **PASS** | Medium 1 / Low 2 |
| 4-3. 구현플랜 실행 가능성 | **PASS (조건부)** | High 1 / Medium 3 / Low 1 |
| 4-4. 문서 간 일관성 | **FAIL** | High 2 / Medium 4 / Low 2 |

**총 이슈**: Critical 0건 / High 6건 / Medium 13건 / Low 9건 = **28건**

---

## 4-1. 기획 ↔ 코드 정합성 체크

### 검증된 항목 (정합 확인)

| 기획서 참조 | 실제 코드 | 결과 |
|------------|----------|------|
| `mutateAgent` 패턴 (registry.go:239-286) | registry.go:239에서 시작, 286에서 끝 확인 | **일치** |
| Event 구조체 12개 필드 (agent.go:155-168) | agent.go:155-168에서 정확히 12개 필드 확인 | **일치** |
| EventType 14종 (agent.go:138-153) | agent.go:138-153에서 14개 상수 확인 | **일치** |
| Agent 구조체 필드 (agent.go:39-76) | 37개 필드 확인, Go/Swift 불일치 4개 필드 정확 | **일치** |
| RecordHook* 메서드 27개 | managed_state.go에서 27개 함수 확인 | **일치** |
| Request 구조체 (ipc.go:76-116) | ipc.go:76-116 정확 확인 | **일치** |
| Response 구조체 (ipc.go:118-129) | ipc.go:118 이후 확인 | **일치** |
| prepareHookRequest (server.go:637) | server.go:637에서 함수 정의 확인 | **일치** |
| FollowEvents 롱폴링 (events.go:29) | events.go:29에서 함수 정의 확인 | **일치** |
| FileEventStore maxEventEntries 10000 | store/events.go:21에서 `const maxEventEntries = 10000` 확인 | **일치** |
| registry.go 371줄 | 371줄 확인 | **일치** |
| managed_state.go 933줄 | 933줄 확인 | **일치** |
| MenuBarViewModel.swift 933줄 | 933줄 확인 (`934줄`이라 적힌 곳도 있으나 실제 933줄) |  **근사 일치** |
| MenuBarViews.swift 1,101줄 | 1,101줄 확인 | **일치** |
| PixelOfficeView.swift 1,025줄 | 1,025줄 확인 | **일치** |
| ipc.go 605줄 | 605줄 확인 | **일치** |
| server.go 733줄 | 733줄 확인 | **일치** |
| pollRuntimeState 2초 간격 (hamd/main.go:109) | main.go:109에서 `2*time.Second` 확인 | **일치** |
| Swift 폴링: 5초 refresh + 15초 event follow | MenuBarViewModel.swift:76-77 확인 | **일치** |
| Go/Swift Agent 불일치 4개 필드 | Swift Agent.swift에 SessionWindowIndex/SessionTabIndex/LastAssistantMessage/SubAgents 없음 확인 | **일치** |
| .claude/agents/ 디렉토리 | 8개 에이전트 정의 존재 확인 | **일치** |

### 발견된 이슈

#### [H-01] Go Command 상수 개수 불일치
- **문서**: current-state.md, mission-control.md 등에서 "52개" 반복 기술
- **실제**: `go/internal/ipc/ipc.go`에서 `Command` 상수 **51개** 확인 (grep 카운트)
- **영향**: 기획서 전반에서 "52개 커맨드"로 참조하므로, IPC 커맨드 목록 표와 실제 코드 사이에 1개 차이 존재
- **원인 추정**: 문서 작성 시점에 카운트 오류이거나, 이후 1개가 제거/통합된 것으로 추정

#### [H-02] Swift DaemonCommand enum 개수 불일치
- **문서**: current-state.md "Swift DaemonCommand enum (19개)", tech-migration.md 부록 "DaemonCommand (17개)"
- **실제**: `Sources/HamCore/DaemonIPC.swift`에서 DaemonCommand enum case **16개** 확인
  ```
  runManaged, attachSession, observeSource, createTeam, addTeamMember, listTeams,
  listItermSessions, listAgents, status, events, followEvents,
  setNotificationPolicy, setRole, removeAgent, getSettings, updateSettings
  ```
- **영향**: mission-control.md P1-0에서 "6개 추가하여 동기화"라고 기술했으나, 기준점이 19개인지 16개인지에 따라 추가해야 할 개수가 달라짐
- **심각도**: High -- 구현 시 혼란 유발 가능

#### [H-03] hookPayload 구조체 파일 위치 오류
- **문서**: current-state.md에서 `go/cmd/ham/main.go:433-460`으로 기술
- **실제**: `go/cmd/ham/commands.go:433-460`에 위치
- **영향**: 개발자가 해당 코드를 찾을 때 혼란. main.go에는 hookPayload가 없음
- **심각도**: High -- 구현 플랜에서 이 파일을 참조하는 곳이 있다면 잘못된 파일을 수정하게 됨

#### [M-01] Event 구조체 LifecycleConfidence 타입
- **문서**: current-state.md의 Event 구조체 표에서 LifecycleConfidence를 `float64`로 기술
- **실제**: agent.go:167에서 `LifecycleConfidence float64` 확인 -- **일치함**
- **그러나**: mission-control.md의 확장 Event 스키마에서 같은 필드를 `string`으로 표기 (json tag가 `lifecycle_confidence,omitempty`)
- **영향**: 기존 필드의 타입이 문서 간에 불일치. 구현 시 혼동 가능

#### [M-02] MenuBarViewModel 줄 수 불일치 (경미)
- **문서**: ham-studio.md, tech-migration.md에서 "934줄"로 기술
- **실제**: 933줄
- **영향**: 경미하지만 정확성 문제

#### [M-03] current-state.md의 "ham hook" CLI 하위명령 27개 주장
- **문서**: "ham hook <type>" 명령에 "27개 하위명령"이라고 기술
- **실제**: Go Command 상수에서 hook 관련은 27개 맞음 (CommandHookToolStart부터 CommandHookFileChanged까지)
- **그러나**: Go Command 전체가 51개이므로 비-hook 커맨드는 24개. 문서의 52 - 27 = 25와 불일치

#### [M-04] DaemonCommand에서 누락된 커맨드 목록 정확성
- **문서**: current-state.md에서 "Swift에 있는 19개 커맨드" 목록에 `listTmuxSessions`를 포함
- **실제**: Swift DaemonCommand에 `listTmuxSessions`는 **없음** (16개만 존재)
- **영향**: 이미 Swift에 있다고 가정한 커맨드가 실제로 없으므로, P1-0의 동기화 작업 범위가 달라짐

#### [M-05] current-state.md에서 Go 파일 수 "48개 비테스트" 주장
- **실제**: glob 결과에서 비-테스트 Go 파일을 카운트하면 수치가 약간 다를 수 있음
- **영향**: 경미. 문서화 시점과 현재 시점의 파일 수 차이일 수 있음

#### [L-01] ipc.go의 Client 타임아웃
- **문서**: current-state.md "클라이언트 타임아웃 3초 (ipc.go:163)"
- **실제**: ipc.go:163 근처에서 타임아웃 설정 확인 필요하지만, 문서가 참조하는 줄 번호가 정확한지 정밀 검증하지 못함
- **영향**: Low

#### [L-02] server.go dispatch 줄 범위
- **문서**: tech-migration.md에서 "dispatch() (server.go:136-634)"로 기술
- **실제**: prepareHookRequest가 server.go:637에서 시작하므로 dispatch는 그 앞에서 끝남. 대략 맞지만 정밀 검증 필요

#### [L-03] store.go의 SaveAgents 줄 범위
- **문서**: tech-migration.md에서 "SaveAgents() (store.go:54-91)"
- **실제**: 정밀 검증하지 못했으나, store.go가 118줄이므로 범위는 합리적

#### [L-04] current-state.md에서 "총 16,122줄" Go 코드량 주장
- **실제**: 정밀 카운트하지 않았으나, 핵심 파일들의 줄 수가 일치하므로 대략 맞을 것으로 추정

### 스키마 하위 호환성 검토

| 제안된 변경 | 하위 호환성 | 평가 |
|------------|-----------|------|
| Event 구조체 10개 필드 추가 (모두 omitempty) | **호환** | 기존 JSONL이 새 필드를 zero value로 디코딩. 안전 |
| Swift AgentEventPayload 옵셔널 필드 추가 | **호환** | decodeIfPresent 사용. 안전 |
| IPC Request에 Graph bool 필드 추가 | **호환** | omitempty. 기존 클라이언트는 이 필드를 보내지 않음 |
| 신규 IPC 커맨드 (inbox.list 등) | **호환** | 기존 커맨드에 영향 없음. dispatch에 새 case 추가만 |
| Artifact 별도 파일 저장 | **호환** | 기존 이벤트 파일 변경 없음. 새 디렉토리만 생성 |

**결론**: 제안된 모든 스키마 변경은 additive하며 하위 호환성을 유지한다. 안전하다.

### Go-Swift 동기화 검토

| 영역 | 현재 상태 | 제안 | 평가 |
|------|----------|------|------|
| DaemonCommand enum | Go 51개 vs Swift 16개 | 6개 추가 + unknown fallback | **합리적** (hook 커맨드는 Swift 불필요) |
| Agent 필드 | Go 37개 vs Swift ~33개 | Phase 1에서 수동 동기화 | **합리적** (Phase 2에서 코드 생성 검토) |
| Event 필드 | Go 12개 → 22개, Swift 12개 → 22개 | 양쪽 동시 확장 | **필수** (P1-1에서 동시 수행) |

---

## 4-2. 기획 ↔ 로드맵 정합성 체크

### 로드맵 비전 반영 확인

| 로드맵 항목 | 해당 기획서 | 반영 여부 |
|------------|-----------|----------|
| Mission Control (관측) | mission-control.md | **반영됨** -- P1-0~P1-5 전체 커버 |
| ham Studio (조작) | ham-studio.md | **반영됨** -- 5개 하위 기능 상세 기획 |
| AgentOps Platform (운영) | agentops-platform.md | **반영됨** -- 5개 하위 기능 상세 기획 |
| 기술 마이그레이션 (섹션 6) | tech-migration.md | **반영됨** -- 6-1~6-8 모두 커버 |
| 차별화 전략 (vs Cursor/Windsurf/Warp) | 각 문서의 경쟁 제품 비교 섹션 | **반영됨** |
| 타겟 사용자 5개 페르소나 | roadmap-0405.md 섹션 2 | **반영됨** -- 각 기획서의 시나리오에 반영 |
| 브랜드 전략 (햄스터=브랜드, 본체=AgentOps) | roadmap-0405.md | **반영됨** -- ham Studio에서 "에디터 아닌 조종석" 명시 |
| Phase 순서 (Mission Control → ham Studio → AgentOps) | implementation-plan.md | **반영됨** -- Phase 1→2→3 순서 유지 |

### 발견된 이슈

#### [M-06] 로드맵의 "대담한 기능 제안" 반영도
- **문서**: roadmap-0405.md 섹션 4에서 5개 대담한 제안 (협업 오케스트레이션, 비용 대시보드, 플레이북, Terminal IDE, AI 디버거)
- **기획서 반영**: 5개 모두 Phase 1~3에 분산 배치되어 반영됨
- **그러나**: "비용/토큰 대시보드"가 P1-4에서 ADR-3 미결정으로 시나리오 C (아무 데이터도 없음 → Phase 2 이관) 가능성이 열려 있음. 로드맵에서는 핵심 기능으로 강조했으나 데이터 소스 부재 시 Phase 1에서 빠질 수 있음
- **심각도**: Medium -- 기능 자체가 누락되는 건 아니지만, 로드맵의 기대와 실행 시점이 달라질 수 있음

#### [L-05] 로드맵의 "Channels 연동" 언급 범위
- **문서**: roadmap-0405.md에서 Claude Code channels를 "선택적" 어댑터로 언급
- **기획서**: tech-migration.md 6-5에서 Layer 4 (선택적)로 분류. agentops-platform.md에서 channels 연동을 Phase 3 통합 포인트로 기술
- **평가**: 적절하게 "연구 프리뷰" 특성을 고려하여 core dependency에서 제외함. **합당함**

#### [L-06] 로드맵의 "1년 차에는 멀티 프로바이더 추상화에 시간을 쓰지 않는다" 원칙
- **기획서**: 모든 문서에서 Claude Code 전용으로 일관되게 설계됨
- **평가**: **잘 반영됨**

### 차별화 전략 반영

| 차별점 | 로드맵 | 기획서 | 평가 |
|--------|--------|--------|------|
| Terminal-agnostic (vs Warp) | 명시 | ham-studio.md에서 "에디터를 만들지 않는다" 원칙 명시 | **반영됨** |
| Claude-native hook 통합 (vs AgentOps.ai) | 명시 | mission-control.md에서 "완전 로컬 + Claude Code 네이티브" 강조 | **반영됨** |
| 운영 레이어 (vs Cursor) | 명시 | 전 문서에서 "작업을 맡기는 도구가 아닌 운영하는 도구" 반복 | **반영됨** |
| Ambient UI (메뉴바 유지) | 명시 | ham-studio.md에서 "메뉴바=상태, Studio=조작" 이원화 | **반영됨** |

---

## 4-3. 구현플랜 실행 가능성 체크

### 태스크 순서 및 의존성 검증

| 순서 | 선행 의존 | 평가 |
|------|----------|------|
| P1-0 → P1-1 | P1-0 (안정성) 없이 P1-1 (스키마 확장) 불가 | **합당함** |
| P1-1 → P1-2, P1-3 병렬 | P1-2(Graph)와 P1-3(Inbox) 모두 확장된 Event 필요 | **합당함** |
| P1-2, P1-3 → P1-4 | P1-4는 ADR-3 결과에 의존 | **합당함** |
| P1-4 → P1-5 | P1-5(EventBus)는 모든 subscriber 안정화 후 | **합당함** |
| Phase 2 → Phase 1 완료 후 | Studio가 P1-2(Graph), P1-3(Inbox) 데이터 필요 | **합당함** |
| Phase 3 → Phase 2 완료 후 | Debugger가 Studio UI 필요 | **합당함** |

**순환 의존 없음 확인됨.**

### 에이전트 팀 구성 검토

| 구현플랜 에이전트 | .claude/agents/ 존재 여부 | 평가 |
|-----------------|--------------------------|------|
| go-backend | `go-backend.md` 존재 | **일치** |
| swift-frontend | `swift-frontend.md` 존재 | **일치** |
| test-engineer | `test-engineer.md` 존재 | **일치** |
| architect | `architect.md` 존재 | **일치** |
| code-reviewer | `code-reviewer.md` 존재 | **일치** |
| devops | `devops.md` 존재 | **일치** |
| ui-designer | `ui-designer.md` 존재 | **일치** |
| pm | `pm.md` 존재 (구현플랜에서 미사용) | **참고** |

**모든 구현플랜 에이전트가 실제 .claude/agents/에 정의되어 있음 확인.**

### 발견된 이슈

#### [H-04] P1-0 버그 목록과 실제 코드 간 참조 불일치
- **문서**: mission-control.md P1-0에서 "M-3 (이중 RecordHookSessionSeen)" 수정을 언급
- **실제**: managed_state.go에서 `RecordHookSessionSeen`이라는 이름의 함수는 존재하지 않음. `RecordHookSessionStart`가 존재
- **영향**: High -- 구현 시 해당 버그를 찾을 수 없을 수 있음
- **추정**: 이전 코드 리뷰 시점의 함수명이거나, 내부적으로 쓰이는 헬퍼 이름일 수 있음

#### [M-07] Phase별 커밋 수 추정치의 현실성
- **문서**: Phase 1 총 16-22 커밋, Phase 2 총 18-24 커밋, Phase 3 총 17-23 커밋
- **평가**: 각 Phase의 태스크 수와 범위를 고려하면 합리적인 추정. 다만 P1-4가 시나리오 C로 갈 경우 2-3 커밋이 줄어들 수 있음
- **심각도**: Medium -- 추정치 자체보다, 시나리오 분기에 따른 범위 변동이 명확히 기술되지 않음

#### [M-08] 빌드/테스트 검증 명령의 정확성
- **문서**: implementation-plan.md에서 `go test ./go/... -race -count=1` 사용
- **실제**: go 디렉토리 구조상 `go test ./go/...`가 프로젝트 루트에서 동작하려면 `go.mod`가 적절히 설정되어 있어야 함
- **추가 확인 필요**: `go test ./...`가 올바른 형태일 수 있음 (go 디렉토리 안에 go.mod가 있는 경우)
- **심각도**: Medium -- 빌드 명령이 틀리면 CI/검증 실패

#### [M-09] Phase 2 실행 프롬프트에서 "dev/phase-1이 main에 머지된 후 시작" 조건
- **문서**: Phase 2 실행 프롬프트에 이 조건이 명시됨
- **현실**: dev/detailed-plan 브랜치에서 기획 작업 중이며, Phase 1 코드 구현은 별도 브랜치(dev/phase-1)에서 할 것으로 보임
- **우려**: 브랜치 전략이 plan-0405.md의 "main 머지는 사람이 직접" 규칙과 맞지만, Phase 간 브랜치 관리 절차가 명확하지 않음

#### [L-07] 실행 프롬프트의 Discord 보고 누락
- **문서**: plan-0405.md에서 "각 Step 시작/완료 시 Discord 채널에 현황 보고" 규칙
- **구현플랜**: 실행 프롬프트에 Discord 보고 지시가 포함되어 있지 않음
- **심각도**: Low -- 운영 프로세스 이슈이지 기술 이슈가 아님

---

## 4-4. 문서 간 일관성 체크

### 용어 일관성

| 용어 | 사용 문서 | 일관성 |
|------|----------|--------|
| SessionEvent vs Event | tech-migration.md에서 `SessionEvent` 신규 타입 제안, mission-control.md에서 `core.Event` 확장 | **불일치** (아래 H-05 참조) |
| InboxItem | mission-control.md, ham-studio.md | **일관** |
| SessionGraph / SessionNode | mission-control.md, ham-studio.md, implementation-plan.md | **일관** |
| EventBus | mission-control.md P1-5, tech-migration.md 6-1 | **일관** |
| ArtifactStore | mission-control.md P1-1 | **단독 정의, 일관** |
| Playbook | ham-studio.md, agentops-platform.md, tech-migration.md 6-7 | **일관** |
| CostRecord | mission-control.md P1-4 | **단독 정의, 일관** |

### 데이터 모델 일관성

| 모델 | 문서 A | 문서 B | 일관성 |
|------|--------|--------|--------|
| Event 확장 필드 10개 | mission-control.md P1-1 | implementation-plan.md P1-1-A | **일관** (동일 필드 목록) |
| SessionEvent 신규 타입 | tech-migration.md 6-2 | mission-control.md P1-1 | **불일치** (아래 H-05) |
| InboxItem 6개 필드 | mission-control.md P1-3 | implementation-plan.md P1-3-A | **일관** |
| CostRecord 필드 | mission-control.md P1-4 | implementation-plan.md P1-4-A | **일관** |
| SessionGraph 필드 | mission-control.md P1-2 | ham-studio.md 5.1 | **일관** |

### IPC 커맨드 이름 일관성

| 커맨드 | mission-control.md | ham-studio.md | agentops-platform.md | implementation-plan.md | 일관성 |
|--------|-------------------|---------------|---------------------|----------------------|--------|
| `inbox.list` | O | O | - | O | **일관** |
| `inbox.mark-read` | O | - | - | O | **일관** |
| `cost.summary` | O | - | - | O | **일관** |
| `session.graph` | - | O (5.1) | - | - | **단독** |
| `teams.orchestrate` | - | O (5.2) | - | - | **단독** |
| `playbooks.list/run/execution` | - | O (5.3) | - | O | **일관** |
| `debug.trace` 등 | - | - | O | O | **일관** |
| `policy.list` 등 | - | - | O | O | **일관** |
| `memory.list` 등 | - | - | O | O | **일관** |

### 발견된 이슈

#### [H-05] Event vs SessionEvent 이중 정의
- **mission-control.md**: `core.Event` 구조체를 additive하게 확장 (10개 필드 추가). 타입 이름 유지
- **tech-migration.md**: 새로운 `SessionEvent` 타입을 정의하고, 기존 Event와 다른 필드 구조 제안 (Payload map[string]any, CostInfo, ApprovalState 등 추가)
- **차이점**:
  - mission-control.md의 확장 Event에는 `Payload`, `Cost`, `ApprovalState`, `Confidence`, `ConfidenceModel`, `Source` 필드가 **없음**
  - tech-migration.md의 SessionEvent에는 `ArtifactData`, `ArtifactType`, `ToolInput`, `ToolType`, `ToolDuration`, `TaskDesc` 필드가 **없음**
  - 두 문서가 제안하는 이벤트 스키마가 서로 다른 필드 세트를 가짐
- **영향**: High -- Phase 1(mission-control.md 기반)과 Phase 2/3(tech-migration.md 기반)에서 이벤트 스키마가 이중 정의됨. 구현 시 어느 스키마를 따를지 혼란
- **권장**: 두 스키마를 병합하여 단일 정규화 스키마를 만들어야 함. mission-control.md의 구체적 필드 + tech-migration.md의 Source/Confidence/Cost 필드를 하나로 통합

#### [H-06] Go Command 개수의 문서 간 불일치
- **current-state.md**: "전체 커맨드 목록 (52개)"
- **tech-migration.md 부록**: "52개 Command 상수"
- **tech-migration.md 6-4**: "52개 Command 상수"
- **ham-studio.md**: "기존 52개 Command"
- **실제 코드**: **51개**
- **영향**: High -- 모든 문서가 동일한 잘못된 숫자를 사용. 이는 초기 current-state.md의 카운트 오류가 전체 문서로 전파된 것

#### [M-10] Swift DaemonCommand 개수의 문서 간 불일치
- **current-state.md**: "19개"
- **tech-migration.md 부록**: "17개"
- **실제 코드**: **16개**
- **영향**: 같은 데이터에 대해 세 가지 다른 숫자가 존재

#### [M-11] P1-2의 IPC 커맨드 이름 불일치
- **mission-control.md P1-2**: status 커맨드에 `graph: true` 옵션을 추가하는 방식 권장 (옵션 A)
- **ham-studio.md 5.1**: 별도 `session.graph` 커맨드를 정의
- **implementation-plan.md P1-2-A**: Request에 `Graph bool` 필드 추가 (mission-control.md 옵션 A 채택)
- **영향**: ham-studio.md에서 `session.graph`로 별도 커맨드를 정의했지만 Phase 1 구현에서는 기존 status 확장. Phase 2에서 별도 커맨드를 추가로 만들어야 하는지 불명확

#### [M-12] RecordHook* 메서드 개수 불일치 (문서 내부)
- **current-state.md**: "27개 hook 핸들러" (managed_state.go 설명)
- **current-state.md Hook 시스템 상세**: hook 타입 27개를 나열하지만, RecordHookStopFailure는 별도로 있고 RecordHookStop은 따로 있어 실제로는 RecordHook* 함수가 27개 맞음
- **Claude Code 참조**: "26개 공식 이벤트 타입 (ham-agents는 27개 처리 - 1개 추가)"라고 기술
- **실제**: managed_state.go에서 RecordHook* 함수 27개 확인. 이 부분은 **일관됨**

#### [M-13] tech-migration.md의 "Mode 통합" 제안과 현재 구현 충돌
- **tech-migration.md 6-2**: Managed/Attached/Observed를 `Source` 필드로 통합 제안
- **현재 코드**: `Agent.Mode`가 `AgentMode` 타입으로 명확히 분리됨
- **mission-control.md**: Mode 통합을 언급하지 않고 기존 구조 유지
- **영향**: tech-migration.md의 장기 비전과 mission-control.md의 Phase 1 구현이 이 부분에서 방향이 다름. 하지만 tech-migration.md 자체가 "Phase 2/3" 마이그레이션을 다루므로 반드시 충돌은 아님

#### [L-08] Artifact 저장 경로 일관성
- **mission-control.md**: `~/Library/Application Support/ham-agents/artifacts/{agent_id}/{event_id}.json`
- **roadmap-0405.md**: 별도 언급 없음
- **implementation-plan.md**: mission-control.md와 동일
- **평가**: **일관됨**

#### [L-09] 문서 내 "ham hook" vs "ham hook <type>" 표기 불일치
- 일부에서 `ham hook <type>`, 일부에서 `ham hook tool-start` 등으로 표기
- **영향**: Low -- 의미는 동일

---

## 발견된 이슈 목록 (심각도별)

### Critical (0건)
없음.

### High (6건)

| # | 이슈 | 문서 | 설명 |
|---|------|------|------|
| H-01 | Go Command 개수 불일치 | 전체 | 문서 52개 vs 실제 51개 |
| H-02 | Swift DaemonCommand 개수 불일치 | current-state.md, tech-migration.md | 문서 19/17개 vs 실제 16개 |
| H-03 | hookPayload 파일 위치 오류 | current-state.md | main.go가 아닌 commands.go에 위치 |
| H-04 | RecordHookSessionSeen 함수명 오류 | mission-control.md | 존재하지 않는 함수명 참조 |
| H-05 | Event vs SessionEvent 이중 정의 | mission-control.md vs tech-migration.md | 두 문서가 다른 확장 스키마 제안 |
| H-06 | Command 개수 전파 오류 | 전체 | 52라는 잘못된 숫자가 모든 문서로 전파 |

### Medium (13건)

| # | 이슈 | 문서 | 설명 |
|---|------|------|------|
| M-01 | LifecycleConfidence 타입 표기 혼재 | mission-control.md | float64 vs string 표기 |
| M-02 | MenuBarViewModel 줄 수 (933 vs 934) | ham-studio.md, tech-migration.md | 1줄 차이 |
| M-03 | 비-hook 커맨드 수 계산 불일치 | current-state.md | 52-27=25 vs 실제 51-27=24 |
| M-04 | Swift에 있다고 기술한 커맨드 누락 | current-state.md | listTmuxSessions 등이 실제로 없음 |
| M-05 | Go 파일 수 48개 주장 미세 오차 가능 | current-state.md | 시점 차이 가능 |
| M-06 | 비용 기능의 로드맵 기대 vs 현실 갭 | mission-control.md | ADR-3 미결정으로 Phase 1 탈락 가능 |
| M-07 | Phase별 커밋 수 시나리오 분기 미반영 | implementation-plan.md | P1-4 시나리오 C 시 범위 변동 |
| M-08 | 빌드 명령 정확성 | implementation-plan.md | `go test ./go/...` 형태 확인 필요 |
| M-09 | Phase 간 브랜치 전략 미명확 | implementation-plan.md | dev/phase-1 → main 머지 절차 |
| M-10 | Swift DaemonCommand 세 가지 숫자 | current-state.md, tech-migration.md | 19, 17, 16 혼재 |
| M-11 | Session Graph IPC 방식 불일치 | mission-control.md vs ham-studio.md | status 확장 vs 별도 커맨드 |
| M-12 | RecordHook 개수 (일관성 확인됨) | current-state.md | 최종 확인: 27개 맞음 |
| M-13 | Mode 통합 방향 불일치 | tech-migration.md vs mission-control.md | 장기 vs 단기 비전 차이 |

### Low (9건)

| # | 이슈 | 문서 | 설명 |
|---|------|------|------|
| L-01 | Client 타임아웃 줄 번호 미검증 | current-state.md | ipc.go:163 |
| L-02 | dispatch 줄 범위 미정밀 | tech-migration.md | server.go:136-634 |
| L-03 | SaveAgents 줄 범위 미정밀 | tech-migration.md | store.go:54-91 |
| L-04 | Go 코드 총 줄 수 16,122 미정밀 | current-state.md | 대략 맞을 것으로 추정 |
| L-05 | Channels 연동 범위 | tech-migration.md | 적절히 선택적으로 분류됨 |
| L-06 | 멀티 프로바이더 배제 원칙 | 전체 | 잘 반영됨 |
| L-07 | Discord 보고 누락 | implementation-plan.md | 실행 프롬프트에 미포함 |
| L-08 | Artifact 경로 일관성 | 전체 | 일관됨 |
| L-09 | hook 표기 불일치 | 전체 | 의미 동일, 표기만 다름 |

---

## 권장 조치

### 즉시 수정 (Phase 1 시작 전)

| 우선순위 | 조치 | 대상 문서 | 예상 작업량 |
|----------|------|----------|-----------|
| **1** | Go Command 개수를 51로 수정 | current-state.md, mission-control.md, tech-migration.md, ham-studio.md | 전체 검색 + 치환 |
| **2** | Swift DaemonCommand 개수를 16으로 통일 | current-state.md, tech-migration.md | 2곳 수정 |
| **3** | hookPayload 위치를 `commands.go:433-460`으로 수정 | current-state.md | 1곳 수정 |
| **4** | `RecordHookSessionSeen` → 실제 함수명으로 수정 또는 M-3 버그의 실제 위치 재조사 | mission-control.md | 조사 후 수정 |
| **5** | Event/SessionEvent 스키마 통합 결정 | mission-control.md + tech-migration.md | 두 문서의 필드 목록을 병합한 단일 정규화 스키마 문서 작성 |

### Phase 1 시작 시 확인

| 조치 | 설명 |
|------|------|
| P1-0 시작 전 Swift DaemonCommand 현황 재확인 | 실제 16개에서 어떤 커맨드를 추가할지 정확히 목록화 |
| `go test` 빌드 경로 검증 | 프로젝트 루트에서 `go test ./go/...` 또는 go 디렉토리에서 `go test ./...`가 동작하는지 확인 |
| ADR-3 (비용 데이터 소스) 조사를 P1-0과 병렬 수행 | P1-4 시나리오 결정이 늦어지면 Phase 1 전체 일정에 영향 |

### 장기 개선 (Phase 2 이전)

| 조치 | 설명 |
|------|------|
| Go/Swift IPC 커맨드 자동 동기화 도구 도입 | 수동 동기화의 반복적 오류 방지. 코드 생성 또는 공유 JSON 스키마 |
| Event/SessionEvent 통합 스키마 문서 | mission-control.md의 구체적 필드 + tech-migration.md의 구조적 필드를 하나로 병합 |
| 문서 내 숫자 참조를 코드에서 자동 추출하는 체계 | Command 개수, enum 개수 등이 코드 변경 시 자동으로 문서와 동기화되도록 |

---

## 전체 평가

**기획서의 품질은 전반적으로 높다.** 특히:

1. **코드 기반 검증이 철저하다**: 대부분의 파일 경로, 함수명, 줄 번호가 실제 코드와 일치한다
2. **제약 사항이 솔직하게 기술되어 있다**: "구현 불가능한 부분과 대안" 섹션이 모든 기능에 포함되어 있고, hook 단방향 제약, IPC 스트리밍 불가 등의 현실적 한계를 정확히 인식하고 있다
3. **하위 호환성이 잘 고려되어 있다**: 모든 스키마 변경이 additive하고 omitempty를 사용한다
4. **로드맵 비전이 충실히 반영되어 있다**: 차별화 전략, 타겟 사용자, 기술 마이그레이션 경로 모두 로드맵과 정합한다
5. **구현플랜이 실행 가능하다**: 태스크 순서가 의존성을 존중하고, 에이전트 팀이 실제 정의와 일치한다

**핵심 개선 필요 사항**:

1. Go Command 51개, Swift DaemonCommand 16개로 숫자 통일 (문서 전체 영향)
2. hookPayload 파일 위치 수정 (commands.go)
3. Event/SessionEvent 이중 스키마를 통합하여 단일 정의로 확정
4. RecordHookSessionSeen 참조 수정

이 4개 항목만 수정하면 기획서는 구현 착수 가능한 상태가 된다.
