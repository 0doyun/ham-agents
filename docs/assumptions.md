# assumptions.md

## Purpose
이 문서는 작업 중 생기는 **가정, 모호한 부분, 임시 결정**을 기록한다.

원칙:
- 작은 모호함 때문에 작업을 멈추지 않는다.
- 대신 여기 기록하고 가장 단순한 방향으로 진행한다.
- 나중에 scope 또는 구조 변경 시 이 기록을 참고한다.

---

## Initial Entries
- 2026-03-24: 최종 목표는 `spec.md` 전체 구현으로 본다. `roadmap.md`는 현재 범위를 제한하는 문서로 사용하지 않는다.
- 2026-03-24: 제품 구현 기준 아키텍처는 `Swift UI + Go CLI/runtime` 이원 구조로 본다.
- 2026-03-24: 메뉴바 앱과 macOS 통합 UX는 Swift가 담당하고, `ham` CLI / `hamd` daemon / 상태 수집은 Go가 담당한다.
- 2026-03-24: 현재 Swift 기반 bootstrap 코드는 과도기 산출물로 보고, 최종 구조에 맞게 점진적으로 재편한다.
- 2026-03-24: attached / observed mode는 최종 목표에 포함되지만, 첫 실구현 slice는 managed mode foundation으로 제한한다.
- 2026-03-24: 현재 작업 폴더는 Git 워크트리이며 origin/main push가 가능한 상태다.
- 2026-03-24: 첫 Go persistence는 SQLite 대신 file-based JSON registry로 시작한다. 이유는 dependency 없이 managed slice를 검증 가능한 상태로 만들기 쉽기 때문이다.
- 2026-03-24: `ham` CLI는 초기 Go slice에서 daemon IPC를 생략하고 runtime/store를 직접 호출한다. `hamd`는 socket/config bootstrap을 제공하고 다음 slice에서 실제 IPC server로 확장한다.
