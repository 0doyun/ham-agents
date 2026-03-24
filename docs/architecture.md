# architecture.md

## Purpose
이 문서는 **현재 구현 기준의 실제 아키텍처 초안**을 기록한다.

주의:
- 이 문서는 최종 설계 문서가 아니라 현재 작업 범위 기준의 살아있는 문서다.
- 미래 확장 방향은 적을 수 있지만, 현재 구현과 분리해서 적는다.

---

## Long-Term Product Architecture Direction

현재 ham-agents의 장기 구조 방향은 다음과 같다.

1. `ham` CLI
2. local runtime / daemon
3. macOS menu bar app
4. adapter layer
5. local persistence

---

## Active Implementation Architecture

현재 구현은 **전체 스펙으로 가기 위한 최소 실행 기반**을 먼저 고정한다.

### Module layout

1. `HamCore`
   - 공통 도메인 모델
   - `Agent`, `AgentStatus`, `AgentMode`, notification policy 등
2. `HamPersistence`
   - local registry / settings / event storage 추상화
   - 초기에는 메모리 기반 또는 단순 파일 기반으로 시작
3. `HamRuntime`
   - agent lifecycle 관리
   - snapshot 생성
   - runtime event coordination
4. `HamInference`
   - structured signal + heuristic 기반 상태 판정
   - confidence / reason 계산
5. `HamNotifications`
   - 시스템 알림 정책과 dedupe/mute 처리
6. `HamAdapters`
   - iTerm2 등 외부 시스템 연동
7. `HamCLI`
   - `ham` 명령 진입점
   - runtime/persistence 위에 thin interface 제공
8. `Apps/HamMenuBarApp`
   - 메뉴바 UI와 pixel office 경험
   - 초기에는 패키지 외부 앱 타깃으로 진화 예정

### Current primary data flow

1. CLI 또는 앱이 명령/이벤트를 생성
2. Runtime이 agent registry를 갱신
3. Persistence가 상태/이벤트를 저장
4. Inference가 가용 신호를 바탕으로 상태를 판정
5. Menu bar app과 notifications가 snapshot을 소비

### Current technical constraints

- 첫 단계에서는 새로운 외부 의존성을 추가하지 않는다.
- 저장소는 항상 빌드 가능한 상태를 유지한다.
- 메뉴바 앱은 macOS 전용이지만 core/runtime 계층은 UI와 분리한다.
- iTerm2 연동은 초기에 adapter boundary만 고정하고 실제 제어는 점진적으로 구현한다.
- 스펙 전체 구현이 목표지만, 구현 순서는 managed-mode 중심의 vertical slice로 진행한다.

## Deferred Architecture

- 실제 background daemon 분리 여부
- transcript storage와 privacy masking의 구체 설계
- attached / observed mode용 adapter 계층 확장
- pixel office 렌더링 엔진과 sprite asset pipeline
- multi-workspace/team synchronization 방식

## Build Surfaces

- Current green surface: SwiftPM (`swift build`, `swift test`)
- Future additional surface: Xcode/macOS app target for menu bar app
