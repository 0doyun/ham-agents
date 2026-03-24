# ham-agents 제품 스펙

문서 상태: Draft  
제품 코드명: **ham-agents**

---

## 1. 제품 한 줄 정의

**ham-agents는 터미널에서 돌아가는 여러 AI 세션을 “햄스터 팀”처럼 관리하는 메뉴바 기반 로컬 오케스트레이터다.**

사용자는 메뉴바에서 계속 움직이는 햄스터를 보다가, 클릭하면 작은 픽셀 오피스 안에서 각 에이전트가 일하는 상태를 한눈에 확인하고, 필요할 때 특정 세션을 열고, 메시지를 보내고, 완료·실패·질문 알림을 받을 수 있다.

---

## 2. 문제 정의

현재 터미널 기반 AI 작업은 보통 다음 문제가 있다.

- iTerm2 탭과 패널 여러 개에 Claude, 셸, 로그, 보조 세션이 섞여 있다.
- 어떤 세션이 살아 있는지, 멈췄는지, 입력을 기다리는지 한눈에 보기 어렵다.
- 끝났는지 확인하려고 직접 열어봐야 한다.
- 병렬 세션이 늘수록 “누가 지금 뭘 하는지” 인지부하가 커진다.
- IDE 기반 agent UI는 보통 플러그인이 만든 세션만 잘 보이고, 외부 터미널 전체를 포괄하지 못한다.

ham-agents는 이 문제를 **귀엽지만 실용적인 관찰·정리·제어 레이어**로 푼다.

---

## 3. 제품 목표

핵심 목표는 다음과 같다.

1. 메뉴바에서 **에이전트 팀 상태를 항상 보이게** 한다.  
2. 여러 터미널 세션을 **작업 주체(agent)** 로 추상화한다.  
3. 사용자는 세션을 일일이 뒤지지 않고도 **누가 일하고 있는지 / 막혔는지 / 끝났는지** 안다.  
4. 클릭 한 번으로 해당 세션을 열고, 필요한 메시지를 보내고, 다시 돌아올 수 있다.  
5. 기능적 도구이면서도 **캐릭터성과 애착**이 있는 경험을 만든다.

비목표:
- 처음부터 모든 터미널 앱과 모든 AI 도구를 완벽 지원하지 않는다.
- 원격 대시보드나 멀티디바이스 동기화는 우선 제외한다.
- agent의 사고 내용을 고정밀 해석하는 것보다 **상태 관찰과 운영 UX**를 우선한다.

---

## 4. 타깃 사용자

1차 타깃:
- iTerm2에서 Claude류 세션을 2개 이상 동시에 돌리는 개발자
- 터미널 중심 워크플로를 선호하는 사용자
- AI 작업을 “작은 팀 운영”처럼 느끼고 싶은 사용자

2차 타깃:
- tmux/여러 프로젝트/여러 브랜치를 동시에 관리하는 파워유저
- 여러 agent를 병렬로 돌리는 사용자

---

## 5. 핵심 가치 제안

ham-agents의 핵심 가치는 햄스터 캐릭터 자체가 아니라 다음 세 가지다.

- **관찰 비용 절감**
- **완료/실패/질문 시그널의 즉시성**
- **세션을 팀처럼 느끼게 하는 감정적 인터페이스**

픽셀 햄스터는 장식이 아니라 **상황판을 자주 보게 만드는 UX 엔진**이다.

---

## 6. 핵심 개념 모델

### Agent
하나의 터미널 세션 또는 transcript source를 대표하는 논리 객체.

예상 필드:
- `agent_id`
- `display_name`
- `provider`
- `host`
- `mode`
- `project_path`
- `role`
- `status`
- `status_confidence`
- `last_event_at`
- `last_user_visible_summary`
- `notification_policy`
- `session_ref`
- `avatar_variant`

### Team
여러 agent를 묶는 단위.  
예: `frontend-squad`, `release-war-room`

### Workspace
프로젝트 경로 중심 묶음. 메뉴바 팝오버 기본 필터 단위.

---

## 7. 세션 모드 정의

### Managed
`ham run ...` 으로 시작한 세션.

특징:
- 생성 시점부터 ham-agents가 추적
- 가장 높은 정확도의 상태 추론
- focus / reopen / stop / message 전송 지원
- 초기 구현 기준 모드

### Attached
이미 존재하는 iTerm2 세션을 선택해서 붙는 모드.

특징:
- 세션 탐지 가능
- 일부 제어 가능
- 초기 context 부족으로 정확도는 Managed보다 낮음
- 사용자가 명시적으로 attach 해야 함

### Observed
transcript/log/file watch만으로 추적하는 모드.

특징:
- 가장 넓은 호환성
- 가장 낮은 제어력
- 상태는 휴리스틱 + confidence 기반
- 클릭 시 원 세션을 직접 열지 못할 수도 있음

원칙:
**모든 agent를 같은 확신도로 보이게 하면 안 된다.**  
mode와 confidence를 UI에 드러내야 한다.

---

## 8. UX 개요

### 메뉴바 상시 경험
메뉴바에는 작은 햄스터 1마리가 항상 노출된다.

상태 예시:
- idle이면 느린 걷기 또는 졸기
- running agent가 늘면 속도 증가
- waiting_input이 있으면 말풍선/느낌표
- error가 있으면 경고 배지 또는 순간 멈춤
- 모든 작업 완료 시 만족 모션

원칙:
**보여주되 시끄럽지 않다.**

### 클릭 시 팝오버
메뉴바 햄스터를 클릭하면 `Ham Office` 팝오버가 열린다.

구성:
1. 상단 헤더: workspace, 총 agent 수, running/waiting/done 수
2. 중앙 픽셀 오피스 캔버스
3. 우측/하단 상세 패널
4. activity feed

### 세부 보기
agent를 클릭하면 다음 액션을 제공할 수 있다.
- Focus in iTerm2
- Open project folder
- Send quick message
- Pause notifications
- Rename role
- Stop tracking / kill session

---

## 9. 오피스 UI 스펙

초기에는 자유 배치형보다 **고정 룸 레이아웃**이 적합하다.

권장 구역:
- Desk zone: active coding
- Library zone: reading / waiting
- Kitchen zone: idle / cooldown
- Alert corner: blocked / error / input needed

필수 애니메이션 세트:
- idle
- walk
- run
- type
- read
- think
- sleep
- celebrate
- alert
- error/stunned

원칙:
**귀여워도 정보는 숨기지 않는다.**

---

## 10. 상태 머신

권장 상태 집합:
- `booting`
- `idle`
- `thinking`
- `reading`
- `running_tool`
- `waiting_input`
- `done`
- `error`
- `disconnected`
- `sleeping`

판정 원칙:
1. 구조화된 신호가 있으면 최우선 사용
2. 없으면 휴리스틱 사용
3. 휴리스틱 사용 시 confidence 계산
4. confidence가 낮으면 과한 표현을 피함

confidence 레벨:
- High: managed + structured events
- Medium: attached + session text + timing
- Low: observed + transcript/log only

---

## 11. 알림 스펙

알림 종류:
- 완료 알림
- 입력 필요 알림
- 실패 알림
- 장시간 침묵 알림
- 팀 요약 알림

기본 정책:
- done: long-running task에만 즉시 알림
- waiting_input: 즉시 알림
- error: 즉시 알림
- noisy progress: 기본 비활성

방해 금지:
- 집중 시간대 설정
- 특정 team/agent mute
- 연속 유사 알림 dedupe

---

## 12. CLI 스펙

핵심 명령 후보:
```bash
ham run <provider>
ham attach
ham list
ham open <agent>
ham ask <agent-or-team> "..."
ham stop <agent>
ham team create <name>
ham team add <name> <agent>
ham status
ham logs <agent>
ham doctor
ham ui
```

원칙:
- UI보다 CLI와 상태 레이어가 먼저 안정적이어야 함
- 자동화 친화적으로 `--json` 지원 고려

---

## 13. 시스템 아키텍처

구성 요소:
1. `ham` CLI
2. local runtime / daemon
3. macOS menu bar app
4. adapter layer
5. local persistence

---

## 14. iTerm2 연동 스펙

우선순위 높은 통합 대상은 iTerm2다.

필수 기능 후보:
- 세션 목록 가져오기
- 현재 활성 세션 식별
- 세션 종료 감지
- 레이아웃 변경 감지
- 세션 포커스 이동
- 선택적 텍스트 전송

---

## 15. 상태 추론 엔진

입력 신호 후보:
- structured launch events
- transcript file changes
- session output tail
- silence duration
- known tool markers
- process exit
- user keystroke / message send events

출력 예시:
```json
{
  "status": "waiting_input",
  "confidence": 0.72,
  "reason": "no output for 18s after question-like prompt"
}
```

---

## 16. 저장 데이터와 프라이버시

저장 대상:
- agent registry
- session mapping
- workspace/team mapping
- event log
- notification history
- user settings
- sprite/asset preferences

원칙:
- transcript 전체 저장은 opt-in
- 기본은 event summary 위주
- 민감 경로/환경변수는 마스킹

---

## 17. 성공 지표

- 하루 평균 활성 agent 수
- 알림 클릭률
- 완료 후 open-session 없이 확인하는 비율
- attach 대비 managed 사용 비율
- 세션 추적 이탈률

---

## 18. 리스크

- 상태 오판
- 외부 세션 일반화 실패
- 귀여움이 기능을 가리는 문제
- 메뉴바 피로감

---

## 19. 최종 요약

ham-agents는 “귀여운 메뉴바 펫”이 아니라, **터미널 AI 세션을 팀처럼 관리하는 로컬 운영 레이어**다.
