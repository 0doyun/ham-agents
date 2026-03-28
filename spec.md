# ham-agents 제품 기획 / 상세 스펙

문서 상태: Draft  
제품 코드명: **ham-agents**  
대표 명령어: `ham run`  
대상 플랫폼: **macOS first**  
핵심 호스트: **iTerm2 + Claude 계열 CLI 세션 우선 지원**

---

## 1. 제품 한 줄 정의

**ham-agents는 터미널에서 돌아가는 여러 AI 세션을 “햄스터 팀”처럼 관리하는 메뉴바 기반 로컬 오케스트레이터다.**

사용자는 메뉴바에서 계속 움직이는 햄스터를 보다가, 클릭하면 작은 픽셀 오피스 안에서 각 에이전트가 일하는 상태를 한눈에 확인하고, 필요할 때 특정 세션을 열고, 메시지를 보내고, 완료·실패·질문 알림을 받을 수 있다.

---

## 2. 문제 정의

현재 터미널 기반 AI 작업은 대체로 이런 식으로 흩어진다.

- iTerm2 탭과 패널 여러 개에 Claude, 셸, 로그, 보조 세션이 뒤섞여 있다.
- 어떤 세션이 살아 있는지, 멈췄는지, 입력을 기다리는지 한눈에 안 들어온다.
- 끝났는지 확인하려고 계속 직접 열어봐야 한다.
- 병렬 세션이 늘수록 “지금 누가 뭘 하는지” 인지부하가 급격히 커진다.
- IDE 기반 agent UI는 보통 그 플러그인이 만든 세션만 잘 보이고, 외부 터미널 전체를 포괄하지 못한다.

ham-agents는 이 문제를 **귀엽지만 실용적인 관찰·정리·제어 레이어**로 푼다.

---

## 3. 제품 목표

핵심 목표는 다섯 가지다.

1. 메뉴바에서 **에이전트 팀 상태를 항상 보이게** 한다.
2. 여러 터미널 세션을 **작업 주체(agent)** 로 추상화한다.
3. 사용자는 세션을 일일이 뒤지지 않고도 **누가 일하고 있는지 / 막혔는지 / 끝났는지** 안다.
4. 클릭 한 번으로 해당 세션을 열고, 필요한 메시지를 보내고, 다시 돌아올 수 있다.
5. 기능적 도구이면서도 **캐릭터성과 애착**이 있는 경험을 만든다.

비목표:

- v1에서 모든 터미널 앱과 모든 AI 도구를 완벽 지원하지 않는다.
- v1에서 원격 대시보드나 멀티디바이스 동기화는 하지 않는다.
- v1에서 agent의 사고 내용 자체를 고정밀 해석하지 않는다. 우선은 상태 관찰과 운영 UX가 중심이다.

---

## 4. 타깃 사용자

1차 타깃은 iTerm2에서 Claude류 세션을 2개 이상 동시에 돌리는 개발자다.  
터미널 중심 워크플로를 선호하고, 에이전트를 병렬로 돌리지만 결국 본인이 오케스트레이션해야 하는 사람이다.

2차 타깃은 tmux/여러 프로젝트/여러 브랜치를 동시에 관리하는 파워유저다.  
AI 작업을 단순한 툴 호출이 아니라 “작은 팀 운영”처럼 느끼고 싶은 사람이다.

---

## 5. 핵심 가치 제안

ham-agents의 핵심 가치는 햄스터 캐릭터 자체가 아니라 다음 세 가지다.

- **관찰 비용 절감**
- **완료/실패/질문 시그널의 즉시성**
- **세션을 팀처럼 느끼게 하는 감정적 인터페이스**

즉, 픽셀 햄스터는 장식이 아니라 **상황판을 자주 보게 만드는 UX 엔진**이다.

---

## 6. 핵심 개념 모델

### Agent

Agent는 하나의 터미널 세션 또는 transcript source를 대표하는 논리 객체다.

필수 필드:

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

Team은 여러 agent를 묶는 단위다.  
예: `frontend-squad`, `release-war-room`, `night-shift`

### Workspace

Workspace는 프로젝트 경로 중심 묶음이다. 메뉴바 팝오버의 기본 필터 단위다.

---

## 7. 세션 모드 정의

이건 제품의 핵심이다.

### Managed (Hook 기반 — Claude Code 우선)

`ham run ...` 으로 시작한 세션.

특징:

- 생성 시점부터 ham-agents가 추적
- **Claude Code hooks 연동으로 100% 정확한 상태 추론** (confidence=1.0)
- Claude Code의 `PreToolUse`, `PostToolUse`, `Stop` 등 hook에서 `ham hook` 커맨드로 데몬에 상태 전송
- 환경변수 `HAM_AGENT_ID`로 에이전트 식별 (PTY 실행 시 자동 주입)
- focus / reopen / stop / message 전송 지원
- v1의 기준 모드
- **hook 미설정 시 기존 PTY 출력 키워드 매칭으로 fallback** (정확도 낮음)

Hook 기반 상태 매핑:

| Hook 이벤트 | AgentStatus | confidence |
|---|---|---|
| 프로세스 시작 | `booting` | 1.0 |
| `PreToolUse` Read/Grep/Glob | `reading` | 1.0 |
| `PreToolUse` Edit/Write/Bash | `running_tool` | 1.0 |
| `PostToolUse` (any) | `thinking` | 1.0 |
| assistant 응답 중 | `thinking` | 1.0 |
| 프롬프트 대기 (stop_reason: end_turn) | `waiting_input` | 1.0 | ※ Claude Code hook API는 이 이벤트를 직접 emit하지 않음. PTY silence 감지 fallback으로 추론. |
| `Stop` 정상 | `done` | 1.0 |
| `Stop` 에러 | `error` | 1.0 |

서브에이전트 지원:

- 부모의 `PreToolUse "Agent"` → 자식 햄스터 등록 (존재 표시만)
- 부모의 `PostToolUse "Agent"` → 자식 햄스터 제거/완료
- 서브에이전트 내부 상태는 추적 불가 (Claude Code가 subagent에 고유 ID를 넘기지 않음)
- UI에서 미니 햄스터로 부모 근처에 시각적 표현

예시:

```bash
ham run claude --project ~/src/app --role reviewer
```

설정 플로우:

```bash
brew install ham-agents     # 바이너리 설치
ham setup                   # 대화형: Claude Code 감지 → ~/.claude/settings.json에 hooks 자동 추가 (사용자 확인)
```

### Attached

이미 존재하는 iTerm2 세션을 선택해서 붙는 모드.

특징:

- 세션 탐지 가능
- 일부 제어 가능
- 초기 context 부족으로 정확도는 Managed보다 낮음
- 사용자가 명시적으로 attach 해야 함

예시:

```bash
ham attach --pick-iterm-session
```

### Observed

transcript/log/file watch만으로 추적하는 모드.

특징:

- 가장 넓은 호환성
- 가장 낮은 제어력
- 상태는 휴리스틱 + confidence 기반
- hook 미지원 프로바이더를 위한 fallback 경로
- 클릭 시 원 세션을 직접 열지 못할 수도 있음

원칙은 단순하다.
**모든 agent를 같은 확신도로 보이게 하면 안 된다.**
mode와 confidence를 UI에 드러내야 한다.

### 프로바이더 우선순위 전략

v1에서는 **Claude Code 하나를 정확하게 지원**하는 것이 최우선이다.
다른 프로바이더(Codex, Gemini CLI 등)는 Phase 3에서 각 프로바이더 전용 어댑터를 추가한다.
범용 추론 엔진은 hook 미지원 프로바이더의 fallback으로만 유지한다.

---

## 8. UX 개요

### 메뉴바 상시 경험

메뉴바에는 작은 햄스터 1마리가 항상 노출된다.

기본 동작:

- idle이면 느린 걷기 또는 졸기
- running agent가 늘면 속도 증가
- waiting_input이 있으면 말풍선/느낌표
- error가 있으면 경고 배지 또는 순간 멈춤
- 모든 작업 완료 시 만족 모션

핵심 원칙은 **보여주되 시끄럽지 않다**다.  
아이콘만 봐도 팀 상태를 대략 알아야 하고, 과한 점멸은 금지다.

### 클릭 시 팝오버

메뉴바 햄스터를 클릭하면 `Ham Office` 팝오버가 열린다.

구성은 네 영역이다.

1. 상단 헤더  
   현재 workspace, 총 agent 수, running/waiting/done 수 표시
2. 중앙 픽셀 오피스 캔버스  
   햄스터들이 책상/도서관/휴게공간/경고구역에 배치됨
3. 우측 또는 하단 상세 패널  
   선택된 agent의 최근 상태, role, 프로젝트, 액션 버튼
4. activity feed  
   “리뷰 시작”, “파일 수정”, “질문 필요”, “완료”, “세션 종료” 등 이벤트 흐름

### 세부 보기

agent를 클릭하면 다음 액션을 제공한다.

- Focus in iTerm2
- Open project folder
- Send quick message
- Pause notifications
- Rename role
- Stop tracking / kill session

---

## 9. 오피스 UI 스펙

### 단일 오피스 공간 (v2 — 4존 그리드에서 변경)

기존 4존 고정 그리드(Desk/Library/Kitchen/Alert Corner)를 **단일 오피스 공간**으로 변경한다.

변경 이유:

- 4존 그리드는 존당 3마리가 한계 → 에이전트 4개 + 서브에이전트면 터짐
- 상태 변경 시 햄스터가 존 사이를 점프 → 부자연스러움
- 대부분 1-2칸만 사용되고 나머지는 비어 공간 낭비

새 구조:

- **하나의 넓은 오피스 공간**에 가구를 배치 (책상, 책장, 소파, 경고등)
- 햄스터는 상태에 따라 **가구 근처에 자연스럽게 배치**
- 서브에이전트는 부모 근처에 **미니 햄스터**로 클러스터 배치
- 에이전트 수 증가에 유연하게 대응 가능

가구 기반 영역 암시:

- 책상 앞 = 작업 중 (thinking, running_tool, booting)
- 책장 앞 = 읽기/분석 (reading)
- 소파/주방 = 휴식/완료 (idle, sleeping, done)
- 경고등 근처 = 주의 필요 (error, waiting_input, disconnected)

상태 전달 방식 (존 구분 없이 3가지로 전달):

1. **스프라이트 애니메이션** — typing, reading, thinking, sleeping, celebrating 등
2. **위치** — 어떤 가구 근처에 있는지
3. **머리 위 아이콘** — ⚠️ (에러), ❓ (입력 대기), ✅ (완료)

서브에이전트 시각화:

- 부모 햄스터 근처에 작은 미니 햄스터로 표현
- 내부 상태는 모르므로 "활동 중" 정도의 기본 애니메이션만 표시
- 생성/소멸은 부모의 Agent tool hook으로 감지

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

상태 매핑 예시:

- `booting` → 책상으로 걸어감
- `thinking` → 제자리 생각 모션
- `running_tool` → 빠른 타이핑
- `reading` → 문서 읽기 모션
- `waiting_input` → 손들기/느낌표 + 머리 위 ❓
- `done` → 작은 점프 + 머리 위 ✅
- `error` → 멈춤/경고 + 머리 위 ⚠️
- `disconnected` → 회색화

원칙은 하나다.
**귀여워도 정보는 숨기지 않는다.**
긴 텍스트는 detail panel과 feed에서 보고, 캔버스는 상태를 직관적으로 보여주는 역할만 맡는다.

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

1. 구조화된 신호가 있으면 그걸 최우선 사용
2. 없으면 휴리스틱 사용
3. 휴리스틱 사용 시 confidence 계산
4. confidence가 낮으면 과한 표현을 피함

confidence 레벨:

- High: managed + structured events
- Medium: attached + session text + timing
- Low: observed + transcript/log only

예시 전이:

- `booting` → `thinking`
- `thinking` → `running_tool`
- `running_tool` → `reading`
- `reading` → `waiting_input`
- `running_tool` → `done`
- `any` → `error`
- `any` → `disconnected`
- `done` → `sleeping`

---

## 11. 알림 스펙

알림은 핵심 기능이다. 많아도 실패고 적어도 실패다.

알림 종류:

- 완료 알림
- 입력 필요 알림
- 실패 알림
- 장시간 침묵 알림
- 팀 요약 알림

기본 정책:

- `done`: long-running task에만 즉시 알림
- `waiting_input`: 즉시 알림
- `error`: 즉시 알림
- noisy progress: 기본 비활성

방해 금지:

- 집중 시간대 설정
- 특정 team/agent mute
- 연속 유사 알림 dedupe
- 상태 flap은 묶어서 1건 처리

---

## 12. CLI 스펙

CLI는 귀여운 제품의 뼈대다. UI보다 먼저 안정적이어야 한다.

핵심 명령:

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
ham setup                   # Claude Code hooks 등 초기 설정
ham hook <event> [args...]  # Claude Code hook에서 호출되는 내부 커맨드
```

의미 요약:

- `ham run`: 새 agent 세션 생성
- `ham attach`: 기존 iTerm2 세션 연결
- `ham list` / `ham status`: 상태 조회
- `ham open`: 해당 세션 포커스
- `ham ask`: 빠른 텍스트 전송
- `ham doctor`: 참조 깨짐, 권한 문제, transcript path 이상 등 진단
- `ham ui`: 메뉴바 앱/팝오버 실행 또는 포커스

자동화 친화성을 위해 `ham list --json`, `ham status --json`, `ham logs --json` 지원을 목표로 한다.

---

## 13. 시스템 아키텍처

구성 요소는 다섯 개다.

1. `ham` CLI  
   사용자가 호출하는 명령줄 인터페이스
2. `hamd` daemon  
   상태 수집, 추론, 이벤트 저장, IPC 담당
3. macOS menu bar app  
   메뉴바 햄스터, 팝오버, 픽셀 오피스, 알림
4. adapter layer  
   iTerm2 adapter / transcript adapter / generic process adapter
5. local store  
   SQLite 또는 file-based event store

데이터 흐름:

1. `ham run` 또는 `ham attach`
2. daemon이 agent 등록
3. adapter가 session/transcript/log 이벤트 수집
4. state engine이 status와 confidence 계산
5. menu bar app이 스트림 구독
6. UI/알림/feed 업데이트

IPC는 Unix domain socket + JSON event stream 정도면 충분하다.

---

## 14. iTerm2 연동 스펙

v1에서 iTerm2는 1급 통합 대상이다.

필수 기능:

- 세션 목록 가져오기
- 현재 활성 세션 식별
- 세션 종료 감지
- 레이아웃 변경 감지
- 세션 포커스 이동
- 선택적 텍스트 전송

지원 수준:

- Must: attach picker, focus session, detect termination, metadata sync
- Should: quick prompt 전송, team 단위 focus
- Won't v1: 모든 shell prompt 포맷 자동 인식, 완전한 bidirectional parser

---

## 15. 상태 추론 엔진

### 1차 경로: Claude Code Hooks (정확)

Claude Code의 hook 시스템을 통해 **추론 없이 사실 기반 상태**를 받는다.

입력 신호:

- `PreToolUse` / `PostToolUse` hook 이벤트
- `Stop` hook 이벤트
- `Agent` tool 사용 (서브에이전트 생성/종료)
- process exit

이 경로의 confidence는 항상 1.0이다.

### 2차 경로: Fallback 추론 (hook 미설정 또는 다른 프로바이더)

hook이 설정되지 않았거나 hook을 지원하지 않는 프로바이더에 대한 fallback.

입력 신호:

- PTY 출력 텍스트 키워드 매칭
- transcript file changes
- session output tail
- silence duration
- known tool markers
- process exit

전략:

- 이벤트 우선 룰베이스
- provider-specific adapter 힌트 추가
- 최종 출력은 `(status, confidence, reason)` 3종 세트

예시:

```json
{
  "status": "waiting_input",
  "confidence": 0.72,
  "reason": "no output for 18s after question-like prompt"
}
```

UI 반영 기준:

- `0.85` 이상: 강한 상태 표현
- `0.5~0.84`: 중간 표현
- `0.5` 미만: neutral/unknown 위주

### `ham setup`과 `ham doctor`

- `ham setup`: Claude Code 감지 시 `~/.claude/settings.json`에 hooks 자동 추가 (사용자 확인 후 merge, 기존 설정 보존)
- `ham doctor`: hook 설정 상태 진단 포함 — hooks 누락 시 fallback 모드임을 안내

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

민감 정보 최소화 원칙:

- 전체 transcript 저장은 opt-in
- 기본은 event summary 위주
- 알림 본문은 최소 노출
- 세션 제어 권한은 mode별 차등 적용
- 민감 경로/환경변수는 마스킹

---

## 17. 설정 화면 스펙

섹션은 다섯 개면 충분하다.

### General

- Launch at login
- Compact mode
- Show menu bar animation always
- Theme auto/day/night

### Integrations

- iTerm2 access
- Transcript directories
- Provider adapters on/off

### Notifications

- done/error/waiting_input 토글
- quiet hours
- preview text on/off

### Privacy

- local-only mode
- event history retention
- transcript excerpt storage on/off

### Appearance

- 햄스터 스킨/모자/책상 테마
- animation speed multiplier
- reduce motion

---

## 18. 대표 UX 플로우

### 새 에이전트 실행

```bash
ham run claude --role reviewer
```

- 새 세션 생성
- 메뉴바 햄스터가 달리기 시작
- 팝오버에 새 햄스터 등장
- 상태가 `booting` → `thinking` → `running_tool` 등으로 변화

### 기존 세션 붙이기

```bash
ham attach
```

- attach 가능한 iTerm2 세션 목록 표시
- 하나를 선택하면 새 agent 생성
- mode는 `attached`, confidence는 중간 이하로 시작
- 이벤트가 쌓이며 안정화

### 질문 필요 알림

- `waiting_input` 판정
- 메뉴바 햄스터에 말풍선
- macOS notification 발송
- 클릭 시 detail 또는 iTerm2 세션으로 이동

### 작업 완료

- `done` 판정
- celebrate 모션
- activity feed 기록
- 조건 충족 시 완료 알림

---

## 19. 성능 목표

메뉴바 앱은 가벼워야 한다.

권장 목표:

- 메뉴바 앱 idle CPU: 평균 `2%` 미만
- daemon idle CPU: 평균 `1%` 미만
- 앱 메모리: `150MB` 미만 목표
- daemon 메모리: `100MB` 미만 목표
- 팝오버 오픈: `200ms` 이하 체감
- 상태 반영 지연: managed `1~2초`, attached `2~5초`, observed `3~8초`

---

## 20. 단계별 범위

### v0.1 알파

- 메뉴바 햄스터 1종
- `ham run`
- `ham attach` (iTerm2 only)
- 기본 agent list UI
- `running / waiting / done / error` 상태
- 완료/질문/실패 알림
- `ham status`

### v0.2 베타

- 픽셀 오피스 캔버스
- role/팀 개념
- activity feed
- quick message 전송
- settings 일부
- confidence 표시

### v1.0

- 여러 햄스터 스킨
- workspace/team filter
- improved heuristics
- 더 나은 detach/reattach UX
- exportable logs
- 풍부한 notification rules

중요 원칙:

- 위 단계는 릴리스 단위 참고용이다.
- 최종 목표는 `spec.md` 전체 구현이다.
- 구현은 작은 vertical slice로 누적한다.

---

## 21. 성공 지표

제품 지표:

- 하루 평균 활성 agent 수
- 알림 클릭률
- 완료 후 open-session 없이 확인하는 비율
- attach 대비 managed 사용 비율
- 세션 추적 이탈률

경험 지표:

- “지금 누가 뭘 하는지 알기 쉬움” 만족도
- false waiting / false done 신고율
- 메뉴바 햄스터를 켜둔 유지 시간

---

## 22. 리스크

가장 큰 리스크는 상태 오판이다.  
해결책은 confidence 노출과 mode 구분이다.

그다음은 외부 세션 일반화 실패다.  
모든 CLI와 프롬프트를 일반화하기 어렵기 때문에 provider adapter 우선 전략이 필요하다.

세 번째는 귀여움이 기능을 가리는 문제다.  
픽셀 오피스가 핵심 정보를 숨기면 실패다. 텍스트 상태와 detail panel은 반드시 살아 있어야 한다.

마지막은 메뉴바 피로감이다.  
항상 보이는 제품은 작은 irritant가 치명적이라, 점멸·배지·알림 정책을 보수적으로 잡아야 한다.

---

## 23. 추천 구현 순서

1단계는 CLI + daemon + 메뉴바 상태점이다.  
귀여움보다 정확한 session registry를 먼저 만든다.

2단계는 `ham run`, `ham attach`, `ham status`, 기본 알림이다.  
여기서 실제 효용이 생긴다.

3단계는 픽셀 햄스터 + 오피스 UI다.  
여기서 제품 캐릭터가 완성된다.

4단계는 상태 추론 고도화 + settings + team model이다.  
여기서 “재미있는 장난감”에서 “계속 켜두는 도구”로 넘어간다.

---

## 24. 최종 요약

ham-agents는 “귀여운 메뉴바 펫”이 아니라, **터미널 AI 세션을 팀처럼 관리하는 로컬 운영 레이어**다.

- `ham run` 으로 agent를 띄우고
- 메뉴바 햄스터가 팀 상태를 요약하고
- 클릭하면 픽셀 오피스에서 누가 일하는지 보이고
- 질문/실패/완료가 알림으로 오고
- 필요하면 해당 세션으로 즉시 점프한다

승부처는 딱 두 개다.

1. state detection의 신뢰감
2. 귀여움이 기능을 가리는 게 아니라 기능을 더 자주 보게 만드는 경험

이 두 개만 잡으면 ham-agents는 밈 프로젝트가 아니라 진짜 매일 켜두는 도구가 된다.
