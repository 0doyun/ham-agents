# tasks.md

## Purpose
이 문서는 **현재 활성 작업 범위와 실행 체크리스트**를 관리한다.

원칙:
- 현재 Active Scope의 체크리스트가 전부 완료되면 다음 epic으로 이동한다.
- 완료된 scope에서 추가 polish/refinement 항목을 발명하지 않는다.
- 미래 기능은 Execution Order에 남기되 현재 체크리스트에는 넣지 않는다.

---

## Current Status
- [x] Epic 1–8 완료 (상세 내역은 아래 Completed Epics 참조)
- [x] Epic 9: Code Cleanup
- [x] Epic 10: Team and Workspace
- [x] Epic 11: Managed Process Lifecycle
- [x] Epic 12: Pixel Office Experience
- [x] Epic 13: Notification Completeness
- [x] Epic 14: Settings Completeness
- [x] Epic 15: Provider Adapter Layer
- [x] Epic 16: Final Polish and Performance
- [x] Epic 17: One-Command Bootstrap
- [x] **Epic 18: Claude Code Hook 기반 상태 추적 (Phase 1)** ✅
- [x] Epic 19: 단일 오피스 UI 재설계 (Phase 2) ✅
- [x] **Epic 21: 오피스 사이드뷰 전환 + 상태 정리** ✅
- [x] **Epic 22: 테스트 안정화 + tmux 지원** ✅
- [x] **Epic 23: 에이전트 출력 요약** ✅
- [x] **Epic 24: OMC 모드 인식** ✅
- [x] **Epic 25: 알림 고도화** ✅
- [ ] **Epic 26: 자율 모드 heartbeat 알림** ← 현재 활성
- [ ] Epic 20: 멀티 프로바이더 확장 (Phase 3, 후순위)

---

## Active Scope

**Epic 26: 자율 모드 heartbeat 알림**

autopilot/ralph 같은 장시간 자율 실행에 대한 주기적 상태 알림.

### Current Slice Checklist

- [ ] heartbeat 알림 설정 (간격: 10분/30분/1시간, 기본 off)
- [ ] 장시간 실행 중 "N분째 실행 중, 현재 상태: thinking" 알림
- [ ] 에러 발생 시 즉시 알림 (기존)
- [ ] settings에 heartbeat 간격 설정 추가 (CLI + UI)
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] heartbeat 설정 시 주기적 알림이 발송됨
- [ ] 에러 시 heartbeat 간격과 관계없이 즉시 알림
- [ ] 기본값 off로 기존 사용자에게 영향 없음

---

## Next Epics (순서대로)

### Epic 23: 에이전트 출력 요약
터미널을 안 열어보고도 에이전트가 뭘 하는지 파악할 수 있게.

- [ ] hook 이벤트에서 구조화된 정보 수집 (도구 이름, 파일 경로 등)
- [ ] `lastUserVisibleSummary`를 구조화된 요약으로 교체
  - "Read: go/internal/ipc/server.go"
  - "Edit: go/cmd/ham/main.go"
  - "Bash: go test ./..."
  - "Agent spawned: test-runner"
- [ ] 디테일 패널에 최근 도구 사용 히스토리 표시 (최근 5개)
- [ ] `ham list`에서 마지막 활동 요약 표시 개선
- [ ] Go/Swift 모델 업데이트 + tests

#### Acceptance Criteria
- [ ] 디테일 패널에서 에이전트의 최근 활동이 구조화된 형태로 보임
- [ ] PTY 원시 출력 대신 "Read: file.go", "Bash: command" 형태의 요약
- [ ] ham list에서도 마지막 활동 요약이 읽기 좋게 표시됨

### Epic 24: OMC 모드 인식
Claude Code + OMC 사용 시 어떤 모드(autopilot, ralph, team 등)로 실행 중인지 표시.

- [ ] OMC 환경변수 감지 방법 조사 (OMC가 어떤 변수를 세팅하는지 확인)
- [ ] Agent 모델에 `omc_mode` 필드 추가 (Go core + Swift)
- [ ] hook command에 OMC 모드 전달 경로 구현 (환경변수 → hook → IPC)
- [ ] UI: 햄스터 이름 옆에 모드 뱃지 (`[autopilot]`, `[ralph]`, `[team]`)
- [ ] `ham list`/`ham status`에서 OMC 모드 표시
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] OMC autopilot/ralph/team 실행 시 해당 모드가 UI에 표시됨
- [ ] OMC 없이 실행 시 모드 필드 미표시 (기존 동작 유지)

### Epic 25: 알림 고도화
waiting_input/error 시 터미널 안 열고도 판단할 수 있도록 알림에 컨텍스트 추가.

- [ ] waiting_input 알림에 마지막 요약 포함 (뭘 물어보는지 미리보기)
- [ ] error 알림에 에러 메시지 요약 포함
- [ ] 알림 클릭 시 해당 에이전트 디테일로 이동 (메뉴바 팝오버 열림)
- [ ] 알림 액션 버튼: "Open Terminal" / "Dismiss"
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] waiting_input 알림에 컨텍스트가 포함되어 터미널 안 열고 판단 가능
- [ ] error 알림에 에러 메시지가 보임
- [ ] 알림 클릭 시 해당 에이전트가 선택된 상태로 팝오버 열림

### Epic 26: 자율 모드 heartbeat 알림
autopilot/ralph 같은 장시간 자율 실행에 대한 주기적 상태 알림.

- [ ] heartbeat 알림 설정 (간격: 10분/30분/1시간, 기본 off)
- [ ] 장시간 실행 중 "N분째 실행 중, 현재 상태: thinking" 알림
- [ ] 에러 발생 시 즉시 알림 (기존)
- [ ] settings에 heartbeat 간격 설정 추가 (CLI + UI)
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] heartbeat 설정 시 주기적 알림이 발송됨
- [ ] 에러 시 heartbeat 간격과 관계없이 즉시 알림
- [ ] 기본값 off로 기존 사용자에게 영향 없음

### Epic 20: 멀티 프로바이더 확장 (Phase 3, 후순위)
Claude Code 이외 프로바이더 지원 추가.

- [ ] Codex 전용 어댑터
- [ ] Gemini CLI 전용 어댑터
- [ ] `ham setup codex`, `ham setup gemini`
- [ ] 범용 추론 엔진은 hook 미지원 프로바이더 fallback으로 유지
- [ ] Go tests

#### Acceptance Criteria
- [ ] Codex/Gemini CLI로 `ham run` 시 해당 어댑터가 상태를 정확하게 추론함
- [ ] `ham setup`이 각 프로바이더별 설정을 안내함

---

## Completed Epics (Archive)

<details>
<summary>Epic 1–8 완료 항목 (클릭해서 펼치기)</summary>

### Epic 1: Repository and Build Bootstrap ✅
- Swift package 생성, 모듈 경계 정의, 기본 테스트 타깃 생성
- GitHub origin 연결, hybrid repository layout 정렬

### Epic 2: Managed Session Foundation ✅
- agent domain model, local registry/persistence
- `ham run`, `ham list`, `ham status` 구현

### Epic 3: Local Runtime and Event Flow ✅
- daemon IPC (Unix socket + JSON), event log (JSONL)
- `ham events`, event query/feed surface
- runtime coordinator, lifecycle transitions, event semantics 확장

### Epic 4: Menu Bar Baseline ✅
- macOS menu bar app target, `MenuBarExtra` 기반 UI
- daemon polling + event-driven refresh
- popover: agent list, detail panel, recent events, actions
- attention queue, severity-aware feed ordering

### Epic 5: Notifications ✅
- done/waiting_input/error/silence 알림
- quiet hours (시간대 기반), preview text masking
- notification settings (daemon-backed round-trip)

### Epic 6: iTerm2 Integration ✅
- session listing, attach picker, focus, termination detection
- quick message (AppleScript write + clipboard fallback)
- richer session identification (`iterm2://session/<id>`)

### Epic 7: Attached and Observed Modes ✅
- `ham attach`, `ham observe` 구현
- attached: iTerm metadata sync (title, cwd, tty, pid, command, activity)
- observed: source file polling, heuristic inference
- disconnect/reconnect detection

### Epic 8: Inference and Attention UX ✅
- confidence + reason 3종 세트
- observed phrase inference (thinking, sleeping, booting, idle, disconnected, error, tool, reading, reconnection)
- attention queue (daemon-backed ordering, subtitles)
- humanized status labels
- CLI confidence/reason/attention visibility
- `ham stop`, `ham logs`, `ham doctor`, `ham ui`, `ham open`, `ham ask`
- lifecycle-aware event presentation
- daemon-backed event presentation hints/metadata

</details>

---

## Execution Order

1. ~~Epic 1: Repository and Build Bootstrap~~ ✅
2. ~~Epic 2: Managed Session Foundation~~ ✅
3. ~~Epic 3: Local Runtime and Event Flow~~ ✅
4. ~~Epic 4: Menu Bar Baseline~~ ✅
5. ~~Epic 5: Notifications~~ ✅
6. ~~Epic 6: iTerm2 Integration~~ ✅
7. ~~Epic 7: Attached and Observed Modes~~ ✅
8. ~~Epic 8: Inference and Attention UX~~ ✅
9. ~~Epic 9: Code Cleanup~~ ✅
10. ~~Epic 10: Team and Workspace~~ ✅
11. ~~Epic 11: Managed Process Lifecycle~~ ✅
12. ~~Epic 12: Pixel Office Experience~~ ✅
13. ~~Epic 13: Notification Completeness~~ ✅
14. ~~Epic 14: Settings Completeness~~ ✅
15. ~~Epic 15: Provider Adapter Layer~~ ✅
16. ~~Epic 16: Final Polish and Performance~~ ✅
17. ~~Epic 17: One-Command Bootstrap~~ ✅
18. ~~Epic 18: Claude Code Hook 기반 상태 추적 (Phase 1)~~ ✅
19. ~~Epic 19: 단일 오피스 UI 재설계 (Phase 2)~~ ✅
21. ~~Epic 21: 오피스 사이드뷰 전환 + 상태 정리~~ ✅
22. **Epic 22: 테스트 안정화 + tmux 지원** ← 현재
23. Epic 23: 에이전트 출력 요약
24. Epic 24: OMC 모드 인식
25. Epic 25: 알림 고도화
26. Epic 26: 자율 모드 heartbeat 알림
20. Epic 20: 멀티 프로바이더 확장 (Phase 3, 후순위)

---

## Spec Coverage Map
어떤 epic이 spec의 어떤 섹션을 커버하는지 참조용.

| Spec 섹션 | 커버하는 Epic |
|-----------|-------------|
| §6 Agent 필드 | Epic 2 ✅ + Epic 12 ✅ + Epic 23 (RecentTools) + Epic 24 (OmcMode) |
| §7 세션 모드 | Epic 7 ✅ + Epic 11 ✅ + Epic 18 ✅ (hook 기반) |
| §8 메뉴바 상시 경험 | Epic 4 ✅ + Epic 12 ✅ |
| §9 오피스 UI | Epic 12 ✅ + Epic 19 ✅ + Epic 21 ✅ (그리드 워크스테이션) |
| §10 상태 머신 | Epic 8 ✅ + Epic 18 ✅ (hook 기반) + Epic 21 ✅ (done 제거) |
| §11 알림 | Epic 5 ✅ + Epic 13 ✅ + Epic 25 (컨텍스트 알림) + Epic 26 (heartbeat) |
| §12 CLI | Epic 2–8 ✅ + Epic 10 ✅ + Epic 18 ✅ (ham hook/setup) + Epic 22 (tmux) |
| §13 아키텍처 | Epic 3 ✅ + Epic 15 ✅ + Epic 22 (tmux adapter) |
| §14 iTerm2/tmux 연동 | Epic 6 ✅ + Epic 22 (tmux 지원) |
| §15 상태 추론 엔진 | Epic 8 ✅ + Epic 18 ✅ (hook 기반) + Epic 23 (출력 요약) |
| §17 설정 화면 | Epic 5 ✅ + Epic 14 ✅ + Epic 26 (heartbeat 설정) |

---

## Notes
- `spec.md`가 최종 목표 문서다.
- `roadmap.md`는 참고용이며 현재 범위를 제한하지 않는다.
- UI는 Swift, CLI/runtime은 Go로 분리하는 방향을 현재 기준 아키텍처로 본다.
- Ralph/autonomous 실행은 항상 가장 높은 우선순위의 미완료 epic부터 이어간다.
- 각 slice 완료 시 `docs/progress.md`, `docs/assumptions.md`, 테스트 결과를 함께 갱신한다.
- Active Scope 체크리스트가 전부 완료되면 다음 epic으로 이동한다. 완료된 scope를 더 다듬지 않는다.
