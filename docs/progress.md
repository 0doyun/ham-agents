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
