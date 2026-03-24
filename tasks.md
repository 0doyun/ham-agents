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

현재 활성 범위는 **runtime lifecycle transition baseline** 다.

- [x] 상세 스펙 복원 및 제품 truth 강화
- [x] `Swift UI + Go CLI/runtime` 방향으로 아키텍처 정렬
- [x] 장기 backlog를 에픽 단위로 정리
- [x] Ralph용 PRD / test spec 아티팩트 생성
- [x] 핵심 모듈 골격 생성
- [x] Git 원격과 연결된 실제 작업 트리로 전환
- [x] 저장소 레이아웃을 Swift UI / Go runtime 방향으로 실제 정렬
- [x] Go workspace bootstrap 추가: `go/cmd/ham`, `go/cmd/hamd`, `go/internal/{core,runtime,store,ipc,adapters}`
- [x] 첫 hybrid implementation slice 완료: managed session registry + `ham status/list`
- [x] `ham` ↔ `hamd` 실제 IPC 연결로 direct store path 축소
- [x] runtime event log / lifecycle foundation 추가
- [x] event feed를 CLI/daemon에서 조회 가능하게 노출
- [x] Swift가 daemon snapshot/event payload를 decode 할 수 있게 정렬
- [x] Swift가 daemon socket/command surface를 통해 snapshot + events를 읽을 수 있게 연결
- [x] Swift menu bar executable target과 baseline status surface 추가
- [x] menu bar 상태 surface가 launch 이후에도 daemon 상태를 주기적으로 따라가게 만들기
- [x] status transition 기반 notification trigger foundation 추가
- [x] actual macOS notification delivery sink 연결
- [x] popover에서 선택 agent detail + recent event context 표시
- [x] popover에서 최소 agent action 연결
- [x] iTerm/workspace opening action baseline 추가
- [x] notification permission 상태를 popover에서 인지/요청 가능하게 만들기
- [x] sessionRef URL 이 있으면 이를 우선 사용하고 없으면 workspace fallback 하도록 세분화
- [x] popover에서 agent별 notification pause/resume action 추가
- [x] popover에서 quick message baseline action 추가
- [x] iTerm이 있는 경우 quick message를 실제 terminal write 로 보내는 baseline 추가
- [x] quick message 성공/실패 feedback baseline 추가
- [x] notification pause/resume 을 daemon persistence 로 이관
- [x] selected agent role rename action 추가
- [x] selected agent stop-tracking baseline 추가
- [x] mode/confidence 를 popover에서 명시적으로 표시
- [x] `ham attach` minimal flow 추가
- [x] `ham observe` minimal flow 추가
- [x] observed source contents를 읽어 status/confidence를 갱신하는 baseline 추가
- [x] daemon serve 중 observed source polling 추가
- [x] `ham open <agent>` baseline 추가
- [x] backend settings state baseline 추가
- [x] Swift menu bar에서 settings를 읽고 일부 토글을 수정할 수 있게 연결
- [x] stored notification settings가 실제 delivery behavior 에 반영되게 연결
- [x] quiet hours enabled setting이 notification suppression에 반영되게 연결
- [x] daemon-backed `ham ask <agent> "..."` baseline 추가
- [x] quiet hours 시간대 범위를 저장/적용하는 baseline 추가
- [x] richer attached/iTerm session identification baseline 추가
- [x] attach picker / iTerm session listing baseline 추가
- [x] attached session termination detection baseline 추가
- [x] broader settings sections baseline 추가
- [x] live event stream / follow baseline 추가
- [x] richer attached metadata sync baseline 추가
- [x] stronger settings sections baseline 추가
- [x] event-driven UI refresh baseline 추가
- [x] richer attached cwd/activity metadata baseline 추가
- [x] higher-fidelity event-driven UI update baseline 추가
- [x] richer attached shell-state fidelity baseline 추가
- [x] stronger event semantics baseline 추가
- [x] lower-latency UI update baseline 추가
- [x] richer event-driven UI semantics baseline 추가
- [x] lower-latency visual updates baseline 추가
- [x] stronger feed semantics baseline 추가
- [x] lower-latency visual polish baseline 추가
- [x] stronger feed semantics baseline 추가
- [x] attached shell-state heuristic refinement baseline 추가
- [x] lower-latency visual polish baseline 추가
- [x] observed lifecycle event baseline 추가
- [x] status reason baseline 추가
- [x] confidence/reason refinement baseline 추가
- [x] attention queue baseline 추가
- [x] severity-aware feed ordering baseline 추가
- [ ] runtime lifecycle transition baseline 추가

### Current Slice Checklist

- [x] settings-aware notification filtering 추가
- [x] preview-text masking behavior 추가
- [x] Swift tests로 settings-driven notification behavior 보호
- [x] daemon-backed message target resolution 재사용
- [x] CLI `ham ask` 구현
- [x] Go adapter sender/fallback 추가
- [x] Go tests/CLI smoke 로 message path 보호
- [x] quiet hours start/end schema 추가
- [x] CLI/UI 에서 quiet hours schedule 수정 가능하게 연결
- [x] current time 기반 quiet hours 판단 추가
- [x] Swift tests로 quiet hours window behavior 보호
- [x] daemon/open-target path 에 richer session identification data 추가
- [x] open/ask path 가 richer session identification 을 재사용하게 정리
- [x] Go/Swift tests로 richer session identification behavior 보호
- [x] iTerm session listing adapter baseline 추가
- [x] attach 가능한 session list surface 를 CLI/UI 쪽에 노출
- [x] Go/Swift tests로 attach picker/listing behavior 보호
- [x] attached session disconnect/termination heuristic 추가
- [x] daemon polling 또는 refresh path 에 disconnect detection 연결
- [x] Go/Swift tests로 attached disconnect behavior 보호
- [x] backend settings schema 에 non-notification section 추가
- [x] CLI/UI 에서 새 settings section 일부를 수정 가능하게 연결
- [x] Go/Swift tests로 broader settings section round-trip 보호
- [x] daemon event follow/read stream surface 추가
- [x] CLI 또는 Swift 가 polling 외의 follow path 를 사용할 수 있게 연결
- [x] Go/Swift tests로 live event follow baseline 보호
- [x] attached session metadata(cwd/title/activity) sync baseline 추가
- [x] daemon/UI 에 richer attached metadata 일부 노출
- [x] Go/Swift tests로 attached metadata sync baseline 보호
- [x] attached shell pid / command metadata 추가
- [x] daemon/UI 에 shell command / pid 일부 노출
- [x] Go/Swift tests로 shell-state fidelity baseline 보호
- [x] daemon event taxonomy 확장 또는 richer event summary 추가
- [x] UI 가 richer event semantics 를 더 직접 활용하게 연결
- [x] Go/Swift tests로 stronger event semantics baseline 보호
- [x] event-driven lane 의 refresh cadence / wakeup cost 줄이기
- [x] UI partial update 경로를 한 단계 더 넓히기
- [x] Go/Swift tests로 lower-latency UI update baseline 보호
- [x] richer event type 별 UI treatment 추가
- [x] activity feed / detail 이 richer event semantics 를 더 직접 활용
- [x] Go/Swift tests로 richer event-driven semantics baseline 보호
- [x] lower-latency visual cue/update baseline 추가
- [x] UI 가 중요 event semantics 를 더 빠르게 시각 반영
- [x] Go/Swift tests로 lower-latency visual update baseline 보호
- [x] activity feed semantics를 더 구조적으로 분류/집계
- [x] feed summary/visual grouping 을 더 직접 활용
- [x] Go/Swift tests로 stronger feed semantics baseline 보호
- [ ] lifecycle transition event coverage 확장
- [ ] runtime transition tests 강화
- [ ] lifecycle summary wording 일관화
- [x] latest-event / feed visuals 추가 polish
- [x] low-noise visual hierarchy refinement
- [x] Go/Swift tests로 visual polish baseline 보호
- [x] activity feed semantics를 더 구조적으로 분류/집계
- [x] feed summary/visual grouping 을 더 직접 활용
- [x] Go/Swift tests로 stronger feed semantics baseline 보호
- [x] settings schema 에 appearance 외 추가 section 확장
- [x] CLI/UI 에 새 settings section 일부를 더 노출
- [x] Go/Swift tests로 stronger settings section round-trip 보호
- [x] Swift view model 에 followEvents 기반 refresh lane 추가
- [x] menu bar 가 일부 event-driven update path 를 사용하게 연결
- [x] Go/Swift tests로 event-driven UI refresh baseline 보호
- [x] attached session cwd/activity metadata heuristic 추가
- [x] daemon/UI 에 cwd/activity metadata 일부 노출
- [x] Go/Swift tests로 cwd/activity metadata baseline 보호
- [x] follow event payload 로 partial UI update 범위 넓히기
- [x] polling fallback 대비 event-driven refresh cost 줄이기
- [x] Go/Swift tests로 higher-fidelity event-driven update baseline 보호
- [x] attached shell-state heuristic 정밀도 개선
- [x] daemon/UI 에 richer shell-state metadata 일부 노출
- [x] Go/Swift tests로 shell-state heuristic refinement baseline 보호
- [x] Go/Swift tests로 shell-state fidelity baseline 보호
- [x] observed status 변화 시 lifecycle event 기록 추가
- [x] activity feed 가 observed lifecycle 변화를 반영하게 연결
- [x] Go/Swift tests로 observed lifecycle event baseline 보호
- [x] daemon/agent schema 에 status reason 추가
- [x] observed/attached 상태 변화에 reason 채우기
- [x] Swift UI 에 reason 일부 노출
- [x] Go/Swift tests로 status reason baseline 보호
- [x] reason과 confidence를 함께 읽기 쉬운 형태로 UI refinement
- [x] mode별 low-confidence wording 정리
- [x] Go/Swift tests로 confidence/reason refinement baseline 보호
- [x] attention-required agent grouping/order baseline 추가
- [x] menu bar 에 attention queue/view 추가
- [x] Go/Swift tests로 attention queue baseline 보호
- [ ] feed row ordering/priority refinement 추가
- [ ] severity-aware feed scanning 개선
- [ ] Go/Swift tests로 severity-aware feed ordering baseline 보호
- [x] Swift bootstrap build/test green 유지
- [x] Go tests green 유지

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
- [x] event log 구조 정의
- [ ] lifecycle transition 정리
- [x] runtime snapshot 제공

#### Acceptance Criteria
- [ ] runtime이 agent 상태를 일관되게 관리함
- [x] 이벤트 기반으로 상태 변경 추적 가능
- [ ] 테스트로 주요 전이 보호

### Epic 4: Menu Bar Baseline
- [x] macOS menu bar app target 생성
- [x] 기본 status indicator 구현
- [x] runtime snapshot 연결
- [x] 최소 팝오버 agent list 구현

#### Acceptance Criteria
- [x] 메뉴바에서 앱이 상주함
- [x] 현재 agent 상태 요약을 볼 수 있음
- [x] CLI/runtime과 상태 소스가 분리되지 않음

### Epic 5: Notifications
- [x] done / waiting_input / error 알림 정의
- [x] dedupe / mute 정책 초안
- [x] notification trigger 연동

#### Acceptance Criteria
- [x] 핵심 상태 알림이 동작함
- [x] 과도한 noisy progress 알림은 기본 비활성

### Epic 6: iTerm2 Integration
- [ ] 세션 식별 방식 결정
- [x] focus/open 연동
- [x] 선택적 message send
- [ ] 종료 감지 초안

#### Acceptance Criteria
- [x] managed session 재오픈 또는 focus 가능
- [x] 연동 실패 시 graceful fallback 존재

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
