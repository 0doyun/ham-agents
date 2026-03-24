# ham-agents Roadmap

문서 상태: Draft  
제품 코드명: **ham-agents**

---

## 1. 이 문서의 역할

이 문서는 **ham-agents의 버전별 확장 계획**을 정리한다.

중요 원칙:
- 이 문서는 미래 계획 문서다.
- 현재 구현 범위를 직접 고정하는 문서는 아니다.
- 현재 작업 범위는 분석 후 별도 작업 문서에서 정해야 한다.
- 이 문서를 근거로 미래 기능을 미리 구현하면 안 된다.

---

## 2. 제품 단계 요약

### v0.1-alpha
가장 작은 usable prototype
- `ham run`
- managed session tracking
- 메뉴바 햄스터
- 상태 4종
- 팝오버 agent list
- 알림

### v0.2-beta
작은 팀 운영감 추가
- team/workspace 개념 시작
- agent detail 강화
- quick message
- activity feed
- confidence 표시

### v0.3
기존 세션 관리로 확장
- attached mode
- iTerm2 연동 고도화
- 상태 추론 개선
- provider adapter 확장

### v1.0
제품 정체성 완성
- pixel office
- richer animations
- polished notifications
- stronger session orchestration UX

---

## 3. 버전별 방향

### v0.1-alpha 방향
목표:
**메뉴바에서 AI 세션 상태를 볼 수 있는 최소 제품**

후보 범위:
- `ham run <provider>`
- managed agent session 생성
- local state/runtime
- `ham status`
- 메뉴바 햄스터 상시 노출
- 상태 4종
- 메뉴바 클릭 시 agent list 팝오버
- macOS notification

### v0.2-beta 방향
목표:
**agent 목록을 작은 팀 운영감이 있는 제품으로 확장**

후보 기능:
- team model
- workspace filter
- detail panel 강화
- quick message
- activity feed
- settings 일부
- sprite variation

### v0.3 방향
목표:
**이미 존재하는 세션도 agent처럼 붙여서 관리**

후보 기능:
- attached mode
- iTerm2 session picker
- metadata sync
- confidence/reason 노출 강화
- `ham attach`
- `ham open`
- `ham ask`

### v1.0 방향
목표:
**ham-agents의 대표 경험 완성**

핵심:
- 메뉴바 상주 햄스터
- 클릭 시 pixel office
- 여러 agent가 실제로 일하는 것처럼 보이는 시각화
- 상태/알림/이동/메시지 전송이 자연스럽게 이어지는 경험

---

## 4. v1 이후 아이디어

- 햄스터 스킨
- 테마별 office
- team templates
- saved agent squads
- more providers
- transcript adapters
- summary cards
- recommended attention queue
- local analytics dashboard

---

## 5. 우선순위 원칙

1. 귀여움보다 효용 우선
2. managed mode 우선
3. attach는 나중
4. pixel office는 기반 안정 후
5. 설정/프라이버시는 너무 늦추지 않음

---

## 6. 문서 우선순위 원칙

분석/구현 시 기본 원칙:
1. 현재 활성 작업 문서
2. tasks/worklog 문서
3. AGENTS.md
4. spec.md
5. roadmap.md

즉, **roadmap.md는 미래 방향 참고용**이다.
