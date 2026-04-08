# hamd Polling & Resource Audit

**Date:** 2026-04-08
**Scope:** go/cmd/hamd + go/internal/runtime + go/internal/store + go/internal/adapters
**Method:** Static code analysis, cross-validated across 4 parallel investigation stages

---

## TL;DR

1. **최악 offender:** iTerm2 어댑터가 매 2초마다 세션당 `ps -ax` (전체 프로세스 테이블 덤프) + `lsof`를 fork+exec — 세션 10개면 2초마다 21+ exec
2. **즉시 조치:** RefreshObservedAgent에 mtime guard 추가 (매 2초 전체 파일 읽기 제거), `ps -ax` 단일 호출로 통합, pprof endpoint 추가
3. **장기 개선:** fsnotify 기반 이벤트 전환, adaptive polling interval, CostTracker offset 기반 incremental parse

---

## 1. Polling Inventory

### 1.1 pollRuntimeState (2초 고정)

| 순서 | 작업 | 조건 | 파일 I/O | 외부 exec |
|------|------|------|----------|-----------|
| 1 | `registry.RefreshObserved(ctx)` | 무조건 | agents.json read + 에이전트당 (stat + full read) | - |
| 2 | `settings.Get(ctx)` | 무조건 | settings.json full read + JSON parse | - |
| 3 | `ensureObservedTranscripts()` | transcript adapter 활성 시 | recursive WalkDir + agents.json read | - |
| 4 | `itermAdapter.ListSessions()` | 무조건 | - | osascript 1 + 세션당 ps 1 + 세션당 lsof 1 |
| 5 | `tmuxAdapter.ListSessions()` | 무조건 | - | list-sessions 1 + 세션당 list-windows + 윈도우당 list-panes + 페인당 lsof |
| 6 | `emitHeartbeatEvents()` | heartbeat 활성 시 | agents.json read (via List) | - |

**Source:** `go/cmd/hamd/main.go:168-205`

### 1.2 CostTracker (5초 고정)

| 작업 | 파일 I/O | 조건 |
|------|----------|------|
| `discoverTranscriptFiles()` | 2-level ReadDir (비재귀) | 무조건 |
| `ingestFile()` per .jsonl | stat 1회 | 무조건 |
| `ParseTranscriptFile()` | 파일 전체 read + line-by-line JSON parse | size 변경 시만 |
| `store.Append()` | JSONL append write | 신규 레코드만 |

**Source:** `go/internal/runtime/cost_tracker.go:70-93, 97-122`

### 1.3 기타

| 위치 | 주기 | 설명 |
|------|------|------|
| `FollowEvents` polling | 200ms | 요청 스코프 (클라이언트 long-poll), 장수 goroutine 아님 |
| `InboxManager.HandleEvent` | 이벤트 발생 시 | 매번 전체 ring JSON 직렬화 + atomic write |
| `eventCallback goroutine` | 이벤트 발생 시 | `go cb(event)` — 무제한 스폰 |

---

## 2. Goroutine Inventory

| # | 위치 | 수명 | 종료 조건 |
|---|------|------|----------|
| 1 | `pollRuntimeState` (main.go:149) | 장수 | `ctx.Done()` |
| 2 | `CostTracker.Start` (cost_tracker.go:74) | 장수 | `ctx.Done()` |
| 3 | signal handler (main.go:135) | 장수 | sigCh 수신 후 종료 |
| 4 | ctx listener (server.go:81) | 장수 | `ctx.Done()` |
| 5 | `go cb(event)` (registry.go:371) | 단발 | callback 완료 |
| 6 | `handleConnection` (server.go:95) | 연결 스코프 | 연결 종료 |
| 7 | SIGKILL timer (managed.go:106) | 2초 | Sleep 후 종료 |

**총 장수 goroutine: 4개 고정.** 이벤트/연결 비례 임시 goroutine 추가.

---

## 3. Anti-Pattern Findings

### CRITICAL

#### AP-1: Observed agent full-read without mtime guard
- **Location:** `go/internal/inference/observed.go:24-44`
- **Description:** 매 2초마다 모든 observed agent의 파일을 `os.Stat()` + `os.ReadFile()` + `strings.ToLower(전체 내용)`. mtime 비교 후 스킵하는 로직 없음.
- **Impact:** observed agent 10개 × 100KB 파일 = 2MB read + 2MB alloc/tick = 60MB/min allocation churn

#### AP-2: seenIDs sync.Map unbounded growth
- **Location:** `go/internal/runtime/cost_tracker.go:47, 181`
- **Description:** 모든 CostRecord dedup key가 영구 저장됨. TTL/eviction/size limit 없음. `seenIDs.Delete()` 호출이 전체 코드에 존재하지 않음.
- **Impact:** 1000 records/day × 200 bytes/key = 200KB/day. 100세션 시 10MB/day. 30일 = 300MB.

#### AP-3: ps -ax called N times per tick (iTerm2)
- **Location:** `go/internal/adapters/iterm2.go:187` (via `sessionActivityForTTY`)
- **Description:** iTerm2 세션마다 `ps -ax -o tty=,pid=,command=` 개별 실행. 전체 프로세스 테이블을 세션 수만큼 반복 덤프.
- **Impact:** 세션 10개 = ps 10회 × 5-50ms = 50-500ms/tick. 단일 호출로 통합하면 90% 절감.

### MAJOR

#### AP-4: settings.json full read every 2s without mtime cache
- **Location:** `go/internal/store/settings.go:42-65`
- **Description:** `FileSettingsStore.Load()`가 매번 `os.ReadFile` + `json.Unmarshal`. `FileSettingsStore` 구조체에 캐시 필드 없음 (path, mu만 존재, line 20-22).
- **Impact:** 설정 변경은 수동 조작 시에만 발생. 99.9% tick이 불필요한 파싱.

#### AP-5: agents.json loaded multiple times per tick
- **Location:** `go/internal/store/store.go:47-52` via `go/internal/runtime/registry.go:69-78`
- **Description:** 한 tick 내에서 `LoadAgents()` (ReadFile + Unmarshal)가 최소 3회 호출: RefreshObserved 1회 + RefreshAttachedByScheme("iterm2") 1회 + RefreshAttachedByScheme("tmux") 1회. heartbeat/transcript 활성 시 추가.
- **Impact:** agents.json은 보통 <10KB로 I/O 자체는 가볍지만, 동일 파일을 3번 읽고 3번 파싱하는 것은 순수 낭비.

#### AP-6: mutex held during I/O (InboxManager)
- **Location:** `go/internal/runtime/inbox.go:59-66`
- **Description:** `HandleEvent`가 `m.mu` 보유 상태에서 `persistLocked()` 호출 → `json.Marshal` + `os.WriteFile` + `os.Rename`. concurrent `List()` / `MarkRead()` 블로킹.
- **Impact:** 고빈도 이벤트 시 contention. ring 100개이므로 직렬화 자체는 빠르지만, 디스크 I/O 지연이 lock 보유 시간에 반영됨.

#### AP-7: CostTracker full-file reparse on size change
- **Location:** `go/internal/runtime/cost_tracker.go:158-191`
- **Description:** 주석 (L170): "We always reparse the whole file." size 변경 감지 시 파일 전체를 byte 0부터 파싱. offset 기반 incremental read 미구현.
- **Impact:** 활성 1MB transcript = tick당 ~200-500줄 JSON parse. dedup으로 store 중복은 방지되지만 CPU/alloc 비용은 매번 발생.

#### AP-8: FileEventStore open/close per append
- **Location:** `go/internal/store/events.go:115-119`
- **Description:** 매 `Append()`마다 `os.OpenFile` → `file.Write` → `file.Close()`. 파일 핸들 미재사용, 버퍼링 없음.
- **Impact:** append 1회 = open + write + close = 3 syscall. burst 시 초당 수십 회.

#### AP-9: No idle detection for iTerm2/tmux adapters
- **Location:** `go/cmd/hamd/main.go:190-198`
- **Description:** iTerm2/tmux가 실행 중인지 확인하지 않고 무조건 `ListSessions()` 호출. 미설치/미실행 시에도 fork+exec 발생 (실패로 끝남).
- **Impact:** iTerm2 미실행 시 osascript exec ≈ 20-50ms 낭비/tick.

### MINOR

#### AP-10: FileAgentStore uses MarshalIndent
- **Location:** `go/internal/store/store.go:76`
- **Description:** `json.MarshalIndent` 사용. compact JSON 대비 ~30-50% 파일 크기 증가 및 CPU 오버헤드.
- **Impact:** 사람이 읽기 쉬운 장점 있으나, tick당 여러 번 쓰는 파일에는 비효율.

#### AP-11: TranscriptAdapter.Discover uses recursive WalkDir
- **Location:** `go/internal/adapters/transcript.go:30`
- **Description:** `filepath.WalkDir` 무한 깊이 재귀. 대량 파일 디렉토리를 transcript dir로 지정하면 매 2초 심각한 I/O.
- **Impact:** transcript adapter 활성 시에만. 대부분 얕은 디렉토리이므로 실질 위험 낮음.

#### AP-12: appendCount resets on daemon restart
- **Location:** `go/internal/store/events.go:34, 125-128`
- **Description:** `appendCount`가 in-memory 카운터로 재시작 시 0. truncation은 1000 append마다 트리거. 짧은 수명의 데몬은 truncation에 도달하지 못함.
- **Impact:** 빈번한 재시작 시 events.jsonl이 10000줄 이상으로 성장 가능.

#### AP-13: Prune methods are dead code
- **Location:** `go/internal/store/artifacts.go:74`, `go/internal/store/cost_store.go:136`
- **Description:** `FileArtifactStore.Prune()`과 `FileCostStore.Prune()` 모두 구현되어 있으나 production 호출처 없음. 테스트만 존재.
- **Impact:** cost.jsonl과 artifact 파일이 무한정 성장. 100세션 × 30일 = cost.jsonl ~300MB.

#### AP-14: eventCallback spawns unbounded goroutines
- **Location:** `go/internal/runtime/registry.go:370-371`
- **Description:** `go cb(event)` — 이벤트마다 새 goroutine. pool/semaphore 없음.
- **Impact:** 각 goroutine은 ~4KB stack. InboxManager callback이 디스크 I/O를 수행하므로 I/O 지연 시 goroutine 누적 가능.

---

## 4. Measurement Estimates

### 4.1 Idle 5분 (에이전트 0, observed 0, iTerm2+tmux 미실행)

```
Per tick (2s):
  LoadAgents:        ~0.1ms (빈 파일)
  settings.Get:      ~0.1ms
  osascript (실패):  ~20-50ms (AppleScript IPC 왕복)
  tmux (실패):       ~5-10ms

CostTracker tick (5s):
  2x ReadDir:        ~0.2ms
  0x stat:           -

Tick당 평균: ~30-60ms
150 ticks / 5분 (pollRuntimeState) + 60 ticks / 5분 (CostTracker)

추정 idle CPU: 150 × 40ms = 6초 CPU / 300초 = ~2% single-core
```

### 4.2 활성 1세션 (1MB transcript) CostTracker tick

```
Size 변경 시:
  2x ReadDir:                      2 syscall
  1x os.Stat:                     1 syscall
  1x os.Open + bufio.Scanner:     2 syscall + N read()
  ~200-500줄 JSON unmarshal:      ~5-15ms CPU
  신규 레코드 Append:             ~1-5 write

추정 syscall/tick (변경 시): 10-20회
추정 syscall/tick (미변경 시): 3-5회 (ReadDir + Stat only)
```

### 4.3 100세션 누적 메모리 풋프린트

```
cost_store (cost.jsonl):
  500 records/session/day × 100 sessions = 50,000 records/day
  ~200 bytes/record = ~10MB/day → 30일 = ~300MB 파일
  Load() 시 전체 파싱 → 300MB 메모리 할당

seenIDs sync.Map:
  50,000 keys/day × ~200 bytes = ~10MB/day
  30일 = ~300MB in-memory (eviction 없으므로)

event_store (events.jsonl):
  maxEventEntries = 10,000 → truncation 있음
  ~500 bytes/entry × 10,000 = ~5MB (bounded)

agents.json:
  100 agents × ~1KB = ~100KB (negligible)
```

### 4.4 pprof endpoint

**존재하지 않음.** `net/http/pprof` import 없음, debug 용 HTTP listener 없음. 런타임 프로파일링 불가능.

---

## 5. Improvement Recommendations

### High Impact, Low Cost

| # | 권고 | 대상 AP | 공수 | 기대 효과 |
|---|------|---------|------|-----------|
| R1 | **RefreshObservedAgent에 mtime guard** — 이전 mtime 캐시, 미변경 시 ReadFile 스킵 | AP-1 | ~2h | idle I/O 90%+ 제거 |
| R2 | **iTerm2/tmux 프로세스 존재 확인** — ListSessions 전에 프로세스 실행 여부 체크 | AP-9 | ~1h | 미실행 시 fork+exec 제거 (~50ms/tick) |
| R3 | **ps -ax 단일 호출로 통합** — 한 번 호출 후 TTY lookup table 구축 | AP-3 | ~2h | N회 → 1회 exec (세션 10개 기준 90% 절감) |
| R4 | **tick 내 agents.json 1회 캐시** — tick 시작 시 1회 로드, 하위 작업에 전달 | AP-5 | ~3h | 중복 file read 2+회 제거 |
| R5 | **settings.json mtime guard** — stat 후 미변경 시 캐시 반환 | AP-4 | ~1h | 불필요한 JSON parse 제거 |

### Medium Impact, Medium Cost

| # | 권고 | 대상 AP | 공수 | 기대 효과 |
|---|------|---------|------|-----------|
| R6 | **Prune 호출 연결** — startup 또는 24h timer에서 cost_store/artifact_store Prune 실행 | AP-13 | ~2h | 디스크 무한 성장 방지 |
| R7 | **seenIDs TTL/LRU** — 24h 만료 또는 100K 상한 | AP-2 | ~3h | 메모리 ~20MB 이하로 bound |
| R8 | **offset 기반 transcript parsing** — lastSeenOffset에서 seek, 신규 바이트만 파싱 | AP-7 | ~4h | 파싱 비용 O(total) → O(delta) |
| R9 | **FileEventStore 핸들 재사용** — 파일 열어두고 buffered write | AP-8 | ~3h | append당 syscall 3→1 |
| R10 | **InboxManager persist를 lock 밖으로** — marshal은 lock 내, write는 lock 외 | AP-6 | ~2h | I/O 중 lock contention 제거 |

### Lower Priority / Larger Scope

| # | 권고 | 대상 AP | 공수 | 기대 효과 |
|---|------|---------|------|-----------|
| R11 | **fsnotify로 observed transcript 폴링 대체** — kqueue/inotify 기반 이벤트 | AP-11, AP-1 | ~8h | 폴링 자체 제거, macOS kqueue 지원 양호 |
| R12 | **Adaptive polling interval** — 변경 없으면 exponential backoff (2s→30s), 변경 시 리셋 | 전체 | ~6h | idle 오버헤드 80-90% 감소 |
| R13 | **pprof endpoint 추가** — `net/http/pprof` + optional `-debug-addr` flag | 측정 인프라 | ~1h | 향후 프로파일링 필수 인프라 |
| R14 | **eventCallback goroutine pool** — semaphore 또는 worker pool로 교체 | AP-14 | ~2h | burst 시 goroutine 폭주 방지 |

---

## 6. Follow-up Tickets

### P1-4.1 (carry-over, 즉시)

| 티켓 | 내용 | 공수 |
|------|------|------|
| T1 | pprof/debug endpoint 추가 | 1h |
| T2 | RefreshObservedAgent mtime guard | 2h |
| T3 | ps -ax 단일 호출 통합 (iTerm2) | 2h |
| T4 | iTerm2/tmux 프로세스 존재 확인 | 1h |
| T5 | Prune 호출 연결 (cost_store + artifact_store) | 2h |

### P1-5 (별도 phase)

| 티켓 | 내용 | 공수 |
|------|------|------|
| T6 | seenIDs TTL/LRU 도입 | 3h |
| T7 | tick 내 agents.json 1회 캐시 | 3h |
| T8 | offset 기반 CostTracker transcript parsing | 4h |
| T9 | settings.json mtime guard | 1h |
| T10 | InboxManager persist lock 분리 | 2h |

### Phase 2+ (장기)

| 티켓 | 내용 | 공수 |
|------|------|------|
| T11 | Adaptive polling interval | 6h |
| T12 | fsnotify 기반 이벤트 전환 | 8h |
| T13 | FileEventStore buffered write | 3h |
| T14 | eventCallback goroutine pool | 2h |

---

## 7. Limitations

1. **CPU 추정치는 분석적 산출.** pprof endpoint가 없어 실측 불가. 실제 값은 하드웨어, 프로세스 수, 파일 크기에 따라 다름.
2. **기본 설정 가정.** heartbeat duplicate-refresh 경로는 `HeartbeatMinutes > 0`일 때만 발생. 기본값은 코드 구조상 0으로 추정했으나 `DefaultSettings()` 반환값 미확인.
3. **100세션 시나리오는 외삽.** 실제 세션당 레코드 수, transcript 크기는 사용 패턴에 따라 크게 다름.
4. **warmup 실패 위험.** `CostTracker.warmupSeenIDsOnce()`가 `sync.Once`이므로 실패 시 재시도 불가. seenIDs가 빈 상태로 남아 중복 ingestion 발생 가능 (cost_tracker.go:208-212).
