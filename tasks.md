# tasks.md

## Purpose
이 문서는 **현재 활성 작업 범위와 실행 체크리스트**를 관리한다.

원칙:
- 분석 전에 이 문서를 과하게 확정하지 않는다.
- 먼저 `spec.md`와 `roadmap.md`를 읽고 현재 구현 범위를 정리한다.
- 이후 현재 버전에 필요한 작업만 작은 vertical slice로 쪼갠다.
- 미래 기능은 아이디어로 남길 수 있지만 현재 체크리스트에는 넣지 않는다.

---

## Current Status
- [x] spec / roadmap 기반 분석 완료
- [x] 전체 스펙 기준의 장기 backlog 정의 시작
- [x] 현재 활성 구현 범위 정의
- [x] architecture 초안 정리
- [x] assumptions 초안 정리
- [x] progress 로그 시작

---

## Product Goal

- [x] 최종 목표는 `spec.md`의 전체 제품 경험 구현
- [x] 구현은 작은 vertical slice 단위로 누적
- [x] 각 slice는 build/test 가능한 green 상태 유지

---

## Active Scope

현재 활성 범위는 **hybrid repository realignment + Go managed-session foundation 첫 vertical slice** 다.

- [x] 상세 스펙 복원 및 제품 truth 강화
- [x] `Swift UI + Go CLI/runtime` 방향으로 아키텍처 정렬
- [x] 장기 backlog를 에픽 단위로 정리
- [x] Ralph용 PRD / test spec 아티팩트 생성
- [x] 핵심 모듈 골격 생성
- [x] Git 원격과 연결된 실제 작업 트리로 전환
- [x] 저장소 레이아웃을 Swift UI / Go runtime 방향으로 실제 정렬
- [x] Go workspace bootstrap 추가: `go/cmd/ham`, `go/cmd/hamd`, `go/internal/{core,runtime,store,ipc,adapters}`
- [x] 첫 hybrid implementation slice 완료: managed session registry + `ham status/list`

### Current Slice Checklist

- [x] Go module/bootstrap 추가
- [x] managed agent domain model을 Go core로 이관
- [x] file-backed registry store 추가
- [x] Go runtime registry + snapshot 구현
- [x] `ham run` / `ham list` / `ham status` 구현
- [x] `hamd` bootstrap entrypoint 추가
- [x] Swift bootstrap build/test green 유지
- [x] Go tests + CLI smoke checks green

## Out of Scope For Current Slice

- [ ] pixel office 실제 렌더링 구현
- [ ] attached / observed mode의 완전 구현
- [ ] iTerm2 제어의 전체 자동화
- [ ] 고급 상태 추론 휴리스틱 완성
- [ ] production-grade notification policy 완성
- [ ] 디자인 polish / sprite asset 제작

---

## Execution Order

### Epic 1: Repository and Build Bootstrap
- [x] Swift package 생성
- [x] 모듈 경계 정의
- [x] 기본 테스트 타깃 생성
- [x] GitHub origin 연결 확인
- [x] hybrid repository layout로 재정렬 시작

#### Acceptance Criteria
- [x] 저장소 구조가 스펙 아키텍처와 대응된다
- [x] `swift build` / `swift test` 가능해야 한다
- [x] 원격 push 가능한 Git 워크트리여야 한다

### Epic 2: Managed Session Foundation
- [x] agent domain model 확정
- [x] local registry/persistence 초안 구현
- [x] `ham status`
- [x] `ham list`
- [x] `ham run` 최소 구현

#### Acceptance Criteria
- [x] managed agent 생성 및 조회 가능
- [x] CLI에서 현재 상태를 읽을 수 있음
- [x] 최소 persistence 경로가 정의됨

### Epic 3: Local Runtime and Event Flow
- [ ] runtime coordinator 구현
- [ ] event log 구조 정의
- [ ] lifecycle transition 정리
- [ ] runtime snapshot 제공

#### Acceptance Criteria
- [ ] runtime이 agent 상태를 일관되게 관리함
- [ ] 이벤트 기반으로 상태 변경 추적 가능
- [ ] 테스트로 주요 전이 보호

### Epic 4: Menu Bar Baseline
- [ ] macOS menu bar app target 생성
- [ ] 기본 status indicator 구현
- [ ] runtime snapshot 연결
- [ ] 최소 팝오버 agent list 구현

#### Acceptance Criteria
- [ ] 메뉴바에서 앱이 상주함
- [ ] 현재 agent 상태 요약을 볼 수 있음
- [ ] CLI/runtime과 상태 소스가 분리되지 않음

### Epic 5: Notifications
- [ ] done / waiting_input / error 알림 정의
- [ ] dedupe / mute 정책 초안
- [ ] notification trigger 연동

#### Acceptance Criteria
- [ ] 핵심 상태 알림이 동작함
- [ ] 과도한 noisy progress 알림은 기본 비활성

### Epic 6: iTerm2 Integration
- [ ] 세션 식별 방식 결정
- [ ] focus/open 연동
- [ ] 선택적 message send
- [ ] 종료 감지 초안

#### Acceptance Criteria
- [ ] managed session 재오픈 또는 focus 가능
- [ ] 연동 실패 시 graceful fallback 존재

### Epic 7: Attached and Observed Modes
- [ ] attach flow 정의
- [ ] confidence 표시 확장
- [ ] observed mode 최소 추적 구현

#### Acceptance Criteria
- [ ] managed 외 세션도 추적 가능
- [ ] confidence와 mode가 UI/CLI에 노출됨

### Epic 8: Inference and Attention UX
- [ ] status inference engine 심화
- [ ] reason/confidence 계산
- [ ] attention queue/feed 설계

#### Acceptance Criteria
- [ ] 구조화 신호 없는 세션도 추론 가능
- [ ] 낮은 confidence는 UI에서 절제해 표시됨

### Epic 9: Pixel Office Experience
- [ ] room layout 구현
- [ ] animation state mapping
- [ ] agent detail interactions

#### Acceptance Criteria
- [ ] 상태가 시각적으로 구분됨
- [ ] 귀여움이 정보 전달을 가리지 않음

---

## Notes
- `spec.md`가 최종 목표 문서다.
- `roadmap.md`는 참고용이며 현재 범위를 제한하지 않는다.
- UI는 Swift, CLI/runtime은 Go로 분리하는 방향을 현재 기준 아키텍처로 본다.
- Ralph/autonomous 실행은 항상 가장 높은 우선순위의 미완료 green slice부터 이어간다.
- 각 slice 완료 시 `docs/progress.md`, `docs/assumptions.md`, 테스트 결과를 함께 갱신한다.
