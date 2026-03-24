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
- 2026-03-24: 첫 구현 언어/도구는 Swift + SwiftPM으로 고정한다. 이유는 macOS 메뉴바 앱과 CLI를 한 언어로 유지하기 위해서다.
- 2026-03-24: 메뉴바 앱은 추후 별도 app target으로 추가하고, 지금은 core/runtime/CLI 경계를 먼저 안정화한다.
- 2026-03-24: attached / observed mode는 최종 목표에 포함되지만, 첫 실구현 slice는 managed mode foundation으로 제한한다.
- 2026-03-24: 현재 작업 폴더는 아직 Git 워크트리가 아니므로 자동 commit/push는 로컬 Git 연결 이후에만 가능하다고 본다.
