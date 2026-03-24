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
