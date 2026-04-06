# docs/spec/ 재검증 리뷰 노트

**작성일**: 2026-04-06 (전면 재작성)
**상태**: 이전 버전은 카운트 오류(Go Command 51 주장)로 무효. 이 문서가 유효한 최신 리뷰다.

---

## 0. 재작성 사유

이전 review-notes.md(이하 "구 리뷰")는 2026-04-06 동일 날짜에 작성되었으나 grep 검증 없이 추정에 기반한 수치를 기록했다. 특히 H-01에서 "Go Command 상수가 51개"라고 주장했고, 이 주장이 커밋 5eb1199를 유발해 docs/spec/current-state.md 외 다수 문서에서 정답(52)을 오답(51)으로 바꿨다. H-04에서는 `RecordHookSessionSeen`이 "존재하지 않는 함수"라고 주장했으나 실제로는 server.go:642, 655에서 두 번 호출된다.

이번 재검증은 grep/Read 기반 증거주의 원칙을 따른다. 모든 수치는 직접 실행한 명령의 출력으로 뒷받침된다. 추정에 기반한 항목은 명시적으로 "미검증"으로 표시한다.

---

## 1. Verification Log

이번 재검증에서 실행한 모든 grep/Read 명령과 출력을 기록한다.

| # | 명령 | 결과 | 검증 대상 |
|---|------|------|----------|
| 1 | `grep -c '^\tCommand' go/internal/ipc/ipc.go` | **52** | Go Command 총 개수 |
| 2 | `awk '/^public enum DaemonCommand/,/^}/' Sources/HamCore/DaemonIPC.swift \| grep -c '    case '` | **16** | Swift DaemonCommand 총 개수 |
| 3 | `grep -n 'RecordHookSessionSeen' go/internal/ipc/server.go` | lines **642, 655** | 함수 존재 여부 |
| 4 | Read go/internal/core/agent.go:167 | `LifecycleConfidence  float64` | 타입 확인 |
| 5 | `ls /Users/User/projects/ham-agents/go.mod` + head | exists, `module github.com/ham-agents/ham-agents` | go.mod 위치 및 모듈명 |
| 6 | `grep -n 'hookPayload' go/cmd/ham/commands.go` | line **433**: `type hookPayload struct` | hookPayload 정의 파일 |
| 7 | `grep -n 'go test\|go build' docs/spec/implementation-plan.md` | `go test ./...` 및 `go build ./go/cmd/ham ./go/cmd/hamd` | 빌드 명령 현황 |
| 8 | `grep -n 'LifecycleConfidence' docs/spec/mission-control.md` | line 235: `LifecycleConfidence    string` | mission-control의 타입 불일치 |

---

## 2. Top Issues (재검증 결과)

### ISSUE-01 (CRITICAL) — 이전 H-01 정정: Go Command 개수는 52

**증거**: `grep -c '^\tCommand' go/internal/ipc/ipc.go` = **52**

**영향**: 구 리뷰 H-01은 "실제 51개"라고 주장했고, 이 주장이 커밋 5eb1199를 유발해 docs/spec/current-state.md, mission-control.md, tech-migration.md 등 여러 문서에서 올바른 수치 52를 51로 교체하는 퇴행을 일으켰다.

**수정 위치**: US-001에서 처리됨 (5eb1199 revert 또는 각 문서 복원)

**정정**: 52가 정답. 구 리뷰의 51 주장은 grep 미실행 추정이었다.

---

### ISSUE-02 (CRITICAL) — 이전 H-04 정정: RecordHookSessionSeen은 실재함

**증거**: `grep -n 'RecordHookSessionSeen' go/internal/ipc/server.go` = lines **642, 655**

**영향**: 구 리뷰 H-04는 "managed_state.go에 `RecordHookSessionSeen`이 존재하지 않음, `RecordHookSessionStart`만 존재"라고 주장했다. 실제로 함수는 go/internal/ipc/server.go에 두 번 호출된다. 검색 대상 파일이 달랐던 것이 원인이다.

**수정 위치**: 기존 H-04 기반의 문서 수정 사항이 있다면 전부 재검토 필요.

**정정**: 이슈 내용을 "존재하지 않는 함수"에서 "server.go에서 두 번 호출되는 것이 의도된 중복 호출인지 확인 필요"로 교체. prepareHookRequest(line 637) 컨텍스트에서 642, 655 두 줄 모두 같은 인자로 호출되는 이유가 로직상 타당한지 코드 리뷰 필요.

---

### ISSUE-03 (CRITICAL) — 빌드/테스트 명령의 패키지 경로 오류

**증거**: `ls /Users/User/projects/ham-agents/go.mod` = exists, module `github.com/ham-agents/ham-agents` (레포 루트). `grep 'go test' docs/spec/implementation-plan.md` = 일부 위치에서 `go test ./go/internal/ipc/`, `go test ./go/internal/store/` 등 하위 경로 명시.

**영향**: go.mod가 레포 루트에 있으므로 레포 루트에서 실행하는 올바른 패턴은 `go test ./...` 또는 `go test ./go/...`이다. `go test ./go/internal/ipc/`처럼 특정 하위 패키지만 지정하는 명령은 CI 파이프라인 전체 검증 용도로는 부족하다. 특히 구 리뷰 M-08에서 "`go test ./go/...`가 올바른 형태일 수 있음"이라고 추정만 했으나 이미 implementation-plan.md에는 `go test ./...`와 `go build ./go/cmd/ham ./go/cmd/hamd`가 기준으로 정해져 있다.

**수정 위치**: US-002에서 처리됨. 실행 프롬프트 내 하위 패키지 한정 명령들을 통합 기준으로 교체.

**정정**: 표준 빌드 명령 = `go test ./... -race -count=1` + `go build ./go/cmd/ham ./go/cmd/hamd`. 일회성 검증용 하위 패키지 명령은 사용 가능하나 CI 기준이 되어서는 안 됨.

---

### ISSUE-04 (CRITICAL) — Event vs SessionEvent 스키마 충돌

**증거**:
- mission-control.md: `core.Event` 구조체를 10개 필드 추가 확장하는 방식 제안
- tech-migration.md: `SessionEvent`라는 신규 타입 제안 (SessionID, ParentAgentID 등 포함)
- agentops-platform.md Phase 3: 확장된 이벤트 필드를 전제로 하는 분석 기능

**영향**: Phase 1에서 `core.Event`를 확장하는 방식으로 구현하면 Phase 3에서 `SessionEvent`가 필요한 코드와 타입 불일치 발생. Phase 1 코드를 Phase 3가 필요로 하는 데이터 구조 없이 만들게 된다.

**수정 위치**: US-003에서 ADR-1로 통합 결정.

**정정**: 단일 `core.Event` 확장 방식(mission-control.md 기준)으로 통일. tech-migration.md의 `SessionEvent` 언급은 ADR-1 결정에 따라 삭제 또는 주석 처리.

---

### ISSUE-05 (CRITICAL) — LifecycleConfidence 타입 불일치 (문서 간)

**증거**:
- `Read go/internal/core/agent.go:167` = `LifecycleConfidence  float64`
- `grep -n 'LifecycleConfidence' docs/spec/mission-control.md` line 235 = `LifecycleConfidence    string`

**영향**: 실제 Go 코드는 `float64`이지만 mission-control.md 확장 스키마에서 `string`으로 기술. 이 스키마를 기준으로 구현하면 타입 불일치로 컴파일 오류 발생.

**수정 위치**: US-003에서 ADR-1 작성 시 함께 수정. mission-control.md의 확장 Event 스키마 표에서 `float64`로 교체.

**정정**: `LifecycleConfidence float64` — 코드가 정답, 문서가 오류.

---

### ISSUE-06 (HIGH) — hookPayload 구조체 파일 위치 오류 (구 H-03 유효 확인)

**증거**: `grep -n 'hookPayload' go/cmd/ham/commands.go` = line **433**: `type hookPayload struct`. main.go에는 hookPayload 없음.

**영향**: 구 리뷰 H-03은 유효한 지적이었다. current-state.md에서 `go/cmd/ham/main.go:433-460`으로 기술한 위치가 틀렸다. 개발자가 main.go를 수정하면 hookPayload에 접근 불가.

**수정 위치**: current-state.md의 hookPayload 참조를 `go/cmd/ham/commands.go:433-460`으로 수정. (구 리뷰 H-03은 올바른 지적이므로 그대로 유효.)

**정정**: `commands.go:433`이 정답. main.go 참조는 모두 교체.

---

### ISSUE-07 (HIGH) — Swift DaemonCommand 개수 문서 간 불일치 (구 H-02 유효, 확인)

**증거**: `awk '/^public enum DaemonCommand/,/^}/' Sources/HamCore/DaemonIPC.swift | grep -c '    case '` = **16**

**영향**: current-state.md "19개", tech-migration.md 부록 "17개", 실제 16개로 세 값이 모두 다르다. mission-control.md P1-0의 "6개 추가하여 동기화" 수치는 기준값에 따라 달라지므로 구현 착수 전 기준값 합의 필요.

**수정 위치**: current-state.md, tech-migration.md의 DaemonCommand 개수를 16으로 통일. P1-0의 추가 개수 재계산.

**정정**: 현재 Swift DaemonCommand = 16개. 19/17 주장은 모두 오류.

---

## 3. 이전 리뷰와의 차이

| 구 리뷰 항목 | 판정 | 사유 |
|-------------|------|------|
| H-01: Go Command 51개 주장 | **무효 — 정정 필요** | 실제 52개. grep 실행으로 확인 |
| H-02: Swift DaemonCommand 16개 | **유효** | 실제 16개 확인됨 |
| H-03: hookPayload가 commands.go에 있음 | **유효** | commands.go:433 확인됨 |
| H-04: RecordHookSessionSeen 존재하지 않음 | **무효 — 정정 필요** | server.go:642, 655에 실재 |
| M-01: LifecycleConfidence string vs float64 불일치 | **유효 (재확인)** | agent.go:167은 float64, mission-control.md는 string |
| M-02: MenuBarViewModel 줄 수 933 vs 934 | **유효 (미검증 상태 유지)** | 이번 재검증 범위 외 |
| M-03: hook 하위명령 27개 | **보류** | Go Command 51 전제 기반. 52 기준으로 재계산 필요 |
| M-04: listTmuxSessions Swift에 없음 | **유효** | 16개 목록에 없음 확인 |
| M-05: Go 비테스트 파일 수 | **미검증** | 이번 재검증 범위 외 |
| M-06~M-09, L-01~L-07 | **보류** | 구 리뷰에서 추정 기반. 개별 검증 필요 |
| ISSUE-03 (빌드 명령) | **신규** | go.mod 위치 직접 확인으로 발견 |
| ISSUE-04 (Event vs SessionEvent) | **신규 (구 M-10 격)** | 문서 간 교차 비교로 발견 |

---

## 4. 후속 조치 매핑

| 이슈 | 심각도 | 처리 스토리 | 처리 방법 |
|------|--------|------------|----------|
| ISSUE-01 (Go Command 52) | CRITICAL | US-001 | 5eb1199 revert 또는 영향 문서 52 복원 |
| ISSUE-02 (RecordHookSessionSeen 실재) | CRITICAL | US-001 | H-04 기반 수정 사항 재검토, 중복 호출 의도 확인 주석 추가 |
| ISSUE-03 (빌드 명령 통일) | CRITICAL | US-002 | `go test ./...` + `go build ./go/cmd/ham ./go/cmd/hamd` 로 통일 |
| ISSUE-04 (Event vs SessionEvent) | CRITICAL | US-003 | ADR-1으로 단일 `core.Event` 확장 방식 결정 |
| ISSUE-05 (LifecycleConfidence 타입) | CRITICAL | US-003 | mission-control.md 스키마 표에서 string → float64 수정 |
| ISSUE-06 (hookPayload 파일 위치) | HIGH | US-001 | current-state.md hookPayload 참조 commands.go:433으로 수정 |
| ISSUE-07 (Swift DaemonCommand 16개) | HIGH | US-001 | current-state.md, tech-migration.md 수치 16으로 통일 |
