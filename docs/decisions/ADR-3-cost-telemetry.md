# ADR-3: Cost / Token Telemetry Data Source

| | |
|---|---|
| **Status** | accepted |
| **Date** | 2026-04-08 |
| **Phase** | Phase 1 P1-4 (비용/토큰 텔레메트리 v1) |
| **Decision** | **Scenario B — Claude Code transcript JSONL parsing** |
| **Author** | ham-agents maintainer (조사 + 결정) |

---

## Context

Phase 1 P1-4 (`docs/spec/implementation-plan.md` 라인 414~468) 는 ham CLI / 메뉴바에 에이전트별 토큰 사용량 + 추정 비용을 표시하는 기능. `docs/spec/agentops-platform.md` Realism Check 표가 비용/토큰 상관 분석을 "Phase 1 ADR-3 조사 결과에 따라 결정" 으로 명시했고, `docs/spec/implementation-plan.md` Phase 1 Scope Gate 는 ADR-3 가 `status: accepted` 로 커밋될 때 P1-4 의 Phase 1 포함 여부를 결정한다.

기존 hook 시스템은 토큰/비용 데이터를 노출하지 않는다는 것이 1차 가정이었고, 본 ADR 는 다음 3 시나리오 중 하나로 판정한다:

- **Scenario A**: hook payload 에서 토큰 데이터 직접 확보 가능
- **Scenario B**: transcript 파일 파싱으로 토큰 역산 가능
- **Scenario C**: 현재 hook/transcript 로 어떤 방식으로도 불가 → P1-4 를 Phase 2 로 이관

---

## Investigation (2026-04-08)

`~/.claude/` 디렉토리 + ham-agents 프로젝트 transcript 를 조사한 결과:

### 1. Hook payload — ❌ 토큰 정보 없음

기존 hook 27종 중 어느 것도 `input_tokens` / `output_tokens` 류 필드를 운반하지 않는다. `go/internal/ipc/ipc.go` 의 hook Command 정의를 확인했고, Claude Code 공식 hooks 문서 (`docs/external/claude-code-hooks-2026-04-08.md`) 의 hook event schema 어디에도 사용량 필드가 없다.

→ Scenario A 불가.

### 2. ham-agents 자체 stats 파일 — ❌ tool count 만

| 파일 | 내용 | 토큰 데이터? |
|---|---|---|
| `~/.claude/.session-stats.json` | 세션별 tool 호출 카운트 (Bash 32, Read 6 등) | ✗ |
| `~/.claude/stats-cache.json` | 일별 messageCount / sessionCount / toolCallCount | ✗ |
| `~/.claude/telemetry/` | 빈 디렉토리 (placeholder) | ✗ |

→ ham-agents 자체 통계는 토큰을 전혀 다루지 않음.

### 3. Claude Code transcript JSONL — ✅ **풍부한 토큰 데이터**

**위치**: `~/.claude/projects/<encoded-project-path>/<session-uuid>.jsonl`
- 프로젝트 경로는 `/` 를 `-` 로 치환 + 앞에 `-` 접두 (예: `-Users-User-projects-ham-agents`)
- 각 세션이 별개 jsonl 파일

**Record 구조** (assistant 메시지의 경우):

```json
{
  "type": "assistant",
  "uuid": "...",
  "parentUuid": "...",
  "sessionId": "...",
  "timestamp": "...",
  "requestId": "...",
  "userType": "...",
  "entrypoint": "...",
  "cwd": "/path/to/project",
  "version": "...",
  "gitBranch": "...",
  "isSidechain": false,
  "message": {
    "id": "...",
    "type": "message",
    "role": "assistant",
    "model": "claude-opus-4-6",
    "content": [...],
    "stop_reason": "...",
    "usage": {
      "input_tokens": 3,
      "cache_creation_input_tokens": 20525,
      "cache_read_input_tokens": 0,
      "output_tokens": 84,
      "service_tier": "standard",
      "cache_creation": {
        "ephemeral_1h_input_tokens": 20525,
        "ephemeral_5m_input_tokens": 0
      },
      "server_tool_use": {
        "web_search_requests": 0,
        "web_fetch_requests": 0
      }
    }
  }
}
```

**핵심 필드**:

| 필드 | 의미 | P1-4 활용 |
|---|---|---|
| `message.model` | 모델 ID (claude-opus-4-6 / claude-sonnet-4-6 / claude-haiku-4-5 등) | 가격표 lookup key |
| `message.usage.input_tokens` | 비-cache input | base price |
| `message.usage.cache_creation_input_tokens` | cache write (1.25x) | 시간대별로 ephemeral 1h vs 5m 분리 |
| `message.usage.cache_read_input_tokens` | cache hit (0.1x) | 큰 할인 |
| `message.usage.output_tokens` | assistant 응답 | output price |
| `message.usage.service_tier` | standard / priority / batch | tier 별 가격 차등 |
| `message.usage.server_tool_use.web_search_requests` | web search 횟수 | 별도 단가 |
| `sessionId` | 세션 식별 | 세션별 집계 |
| `timestamp` | ISO 8601 | 일별/시간별 집계 |
| `gitBranch` | 작업 branch | 브랜치별 비용 (선택) |
| `cwd` | 프로젝트 경로 | 프로젝트별 집계 |
| `requestId` | API 요청 ID | dedup 키 |

**측정 표본**: ham-agents 프로젝트 한 곳에서 transcript 파일 ~70개, 일부 파일은 300+ usage record. 최근 4월 5~7 일 작업 분량 충분히 잡힘.

→ **Scenario B 가능, 추정치가 아닌 정확한 토큰**.

---

## Decision

**Scenario B — Claude Code transcript JSONL parsing** 를 채택한다.

P1-4 는 Phase 1 에 포함하며, 예상 커밋 수는 `docs/spec/implementation-plan.md` 의 시나리오 B 추정 (5-6 커밋) 을 따른다.

### Rationale

- 토큰 데이터 정확도 100% (Anthropic API 응답에 박힌 값을 그대로 읽음)
- 모델 ID 가 함께 저장되어 가격표 매핑으로 USD 환산 가능
- 세션/일/모델/프로젝트별 다축 집계 가능
- 추가 외부 의존성 (Anthropic API 키, 별도 백엔드) 불필요
- ham-agents 의 다른 데이터 소스 (registry, event log) 와 동일한 file-on-disk 패턴이라 운영 일관성 확보

### Trade-offs

- **Schema 안정성 리스크**: Claude Code 가 transcript schema 를 바꾸면 파서가 깨짐. → adapter pattern 으로 격리하고 unit test 로 회귀 방지
- **가격표 유지보수**: Anthropic 가격 변경 시 수동 업데이트 필요. → `pricing.go` 에 모델별 fallback + warning 로깅
- **이벤트 모델 간접화**: 응답 완료 → transcript flush 까지 ~1초 lag 가능. 실시간성이 hook 기반보다 약함. → P1-4 v1 에서는 폴링 (5초) 으로 충분, EventBus 통합은 P2-0 이후
- **파일 위치 가정**: `~/.claude/projects/{encoded-path}/*.jsonl` 경로가 OS 별로 동일한지는 macOS 기준만 확인됨. Linux/Windows 는 후속 검증 필요. → P1-4 v1 은 macOS 전용으로 명시

---

## Implementation Footprint (P1-4 시나리오 B)

**예상 커밋 5-6**:

### Commit 1 — Core types
- `go/internal/core/cost.go` (신규): `CostRecord {AgentID, SessionID, Model, ServiceTier, InputTokens, CacheCreateTokens, CacheReadTokens, OutputTokens, WebSearchRequests, WebFetchRequests, EstimatedUSD, RecordedAt, Source}`
- `go/internal/core/pricing.go` (신규): `ModelPrice {Input, CacheCreateEphemeral5m, CacheCreateEphemeral1h, CacheRead, Output} per 1M tokens` + `LookupModelPrice(model string) (ModelPrice, bool)` + 모델별 정적 테이블 (Opus 4.6, Sonnet 4.6, Haiku 4.5, 4 시리즈, 3.7, 3.5 등)
- 신규 테스트 `cost_test.go` + `pricing_test.go`: USD 계산 + 미지 모델 fallback

### Commit 2 — Transcript parser
- `go/internal/store/transcript_parser.go` (신규): JSONL 라인 단위 read + JSON decode + assistant record 필터 + usage block → CostRecord 매핑
- adapter pattern: `TranscriptRecord` 는 internal 타입, schema 변경 시 이 파일만 수정
- 신규 테스트 `transcript_parser_test.go`: 실제 transcript 샘플 fixture (민감정보 마스킹) 기반 round-trip

### Commit 3 — Cost store
- `go/internal/store/cost_store.go` (신규): `FileCostStore` JSONL append + Load(filter) + Prune + atomic write
- 저장 경로: `${HAM_AGENTS_HOME}/cost.jsonl` (events.jsonl 옆)
- 신규 테스트 `cost_store_test.go`: append/load/filter/atomic-write

### Commit 4 — CostTracker
- `go/internal/runtime/cost_tracker.go` (신규): transcript 디렉토리 폴링 (5초 간격), 새 파일 발견 / 기존 파일 size 증가 시 delta 파싱, dedup (requestId 또는 message.id 기준), CostStore append
- agent ↔ session 매핑: `agent.SessionID` 를 transcript 의 `sessionId` 와 매칭하여 AgentID 부여
- 신규 테스트 `cost_tracker_test.go`: t.TempDir 기반 transcript 디렉토리 시뮬레이션

### Commit 5 — IPC + CLI
- `go/internal/ipc/ipc.go`: `CommandCostSummary = "cost.summary"` + Request `{AgentIDFilter, SinceDays}` + Response `{CostRecords, TotalUSD, ByModel map[string]float64}`
- `go/internal/ipc/server.go`: dispatch case
- `go/cmd/ham/parse.go`: `ham cost` 서브커맨드 (`--agent`, `--days`, `--by-model`, `--by-day`)
- `go/cmd/ham/render.go`: 표 형식 출력 (모델별 토큰 + USD)
- `go/cmd/hamd/main.go`: CostTracker 초기화 + 폴링 시작

### Commit 6 — Swift (선택)
- `Sources/HamCore/DaemonIPC.swift`: `costSummary` 케이스
- `Sources/HamCore/DaemonPayloads.swift`: `CostSummaryPayload`
- `Sources/HamAppServices/MenuBarViewModel.swift`: 5초 refresh 사이클에 today 비용 fetch
- `apps/macos/HamMenuBarApp/Sources/MenuBarViews.swift`: SummaryBadge 또는 CostSection 에 today USD 표시
- 신규 테스트 `Tests/HamCoreTests/CostSummaryPayloadTests.swift`

---

## Pricing Table (Reference, 2026-04-08 기준)

> ✅ P1-4 commit `feat(cost): core CostRecord types and Anthropic pricing table` 시점에 [Anthropic 공식 pricing 페이지](https://platform.claude.com/docs/en/docs/about-claude/pricing) 에서 fetch 완료. 실제 코드는 `go/internal/core/pricing.go` 의 `modelPriceTable` 에 박혀 있으며, 가격 변경 시 본 표 + `pricing.go` 양쪽을 업데이트해야 한다.

| 모델 ID | input ($/1M) | cache write 5m ($/1M) | cache write 1h ($/1M) | cache read ($/1M) | output ($/1M) |
|---|---|---|---|---|---|
| claude-opus-4-6 | 5.00 | 6.25 | 10.00 | 0.50 | 25.00 |
| claude-opus-4-5 | 5.00 | 6.25 | 10.00 | 0.50 | 25.00 |
| claude-opus-4-1 | 15.00 | 18.75 | 30.00 | 1.50 | 75.00 |
| claude-opus-4 | 15.00 | 18.75 | 30.00 | 1.50 | 75.00 |
| claude-sonnet-4-6 | 3.00 | 3.75 | 6.00 | 0.30 | 15.00 |
| claude-sonnet-4-5 | 3.00 | 3.75 | 6.00 | 0.30 | 15.00 |
| claude-sonnet-4 | 3.00 | 3.75 | 6.00 | 0.30 | 15.00 |
| claude-3-7-sonnet (deprecated) | 3.00 | 3.75 | 6.00 | 0.30 | 15.00 |
| claude-3-5-sonnet | 3.00 | 3.75 | 6.00 | 0.30 | 15.00 |
| claude-haiku-4-5 | 1.00 | 1.25 | 2.00 | 0.10 | 5.00 |
| claude-3-5-haiku | 0.80 | 1.00 | 1.60 | 0.08 | 4.00 |
| claude-3-haiku | 0.25 | 0.30 | 0.50 | 0.03 | 1.25 |

Sanity check (`go/internal/core/pricing_test.go`):
- `TestPricing_OpusCostlierThanSonnetCostlierThanHaiku` 에서 input/output 모두 opus > sonnet > haiku 순서 검증
- `TestLookupModelPrice_StripsDatedSuffix` 에서 `claude-opus-4-6-20260101` 같은 dated suffix 도 base 모델로 fallback
- `TestLookupModelPrice_UnknownReturnsFalse` 에서 미지 모델은 `(ModelPrice{}, false)` 반환 → CalculateUSD 호출자가 0 USD 로 처리

---

## Risks & Open Questions

1. **Linux/Windows 경로**: macOS 의 `~/.claude/projects/` 가 다른 OS 에서도 동일한지 미검증. P1-4 v1 은 macOS 전용 가드 추가, 후속에서 확장.
2. **Transcript schema drift**: Claude Code 가 `usage` 블록 필드명/구조를 바꾸면 파서가 깨짐. adapter 패턴 + schema version 감지 (`message.usage.service_tier` 같은 known field 존재 여부) 로 경고 로깅.
3. **Dedup 키**: 동일 메시지가 transcript 에 두 번 적힐 수 있는지 확인 필요. requestId 또는 message.id 를 dedup 키로 사용 권장.
4. **Sidechain 메시지**: `isSidechain: true` record 가 무엇인지 (Agent tool 호출 결과 inner conversation?) 파악하여 별도 카운트할지 결정.
5. **ham-agents 가 실행되지 않은 상태에서 발생한 비용**: 사용자가 ham 없이 `claude` 만 돌렸을 때도 transcript 는 쌓임. CostTracker 는 시작 시 historical replay 옵션 제공 여부 결정 필요.

---

## Addendum: P1-4.1 폴링 → on-demand 전환 (2026-04-08)

P1-4 v1 의 CostTracker 는 5초 간격으로 transcript 디렉토리를 폴링했다. hamd
polling audit (`docs/reviews/2026-04-08-hamd-polling-audit.md`) 에서 idle 시에도
CPU 2% + 지속 I/O 가 발생하는 것이 확인되었고, 사용자 결정에 따라 on-demand 모드로
전환:

- **Before**: `CostTracker.Start(ctx)` → 5초 ticker goroutine → 자동 scan + ingest
- **After**: goroutine 없음. `cost.summary` IPC 요청 시에만 `Tick(ctx)` 한 번 호출
- **seenIDs**: 영구 sync.Map → 매 Tick 호출마다 CostStore 에서 ephemeral rebuild (메모리 bounded)
- **Prune**: startup + 24h timer 로 cost.jsonl 30일 / artifact 100MB 자동 정리

## Next Steps

1. ✅ 본 ADR 를 `status: accepted` 로 main 에 머지 → Phase 1 Scope Gate 통과
2. ✅ P1-4 ralph 런 완료 (9 커밋, critic APPROVE-WITH-RESERVATIONS + fix)
3. ✅ P1-4.1 리소스 최적화 (on-demand + audit quick wins)
4. P1-4 v1 출시 후 Linux/Windows 경로 검증 → P1-4.2 로 확장
