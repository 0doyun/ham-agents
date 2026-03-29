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
- [x] **Epic 26: 자율 모드 heartbeat 알림** ✅
- [ ] **Epic 27: Hook 확장 + 정확도 향상** ← 현재 활성
- [ ] Epic 28: Agent Teams 연동
- [ ] Epic 29: Worktree 시각화
- [ ] Epic 20: 멀티 프로바이더 확장 (Phase 3, 후순위)

---

## Active Scope

**Epic 27: Hook 확장 + 정확도 향상**

Claude Code의 공식 hook 25종 중 추가 활용 가능한 이벤트를 연동하여 상태 추적 정확도를 높인다.

### Current Slice Checklist

**Phase 1: Notification hook → waiting_input 정확 감지**
- [ ] `ham hook notification` 서브커맨드 추가
- [ ] `Notification` hook 이벤트 처리 — stdin JSON에서 `notification_type` 파싱
  - `idle_prompt` → AgentStatusWaitingInput (confidence=1.0)
  - `permission_prompt` → AgentStatusWaitingInput (confidence=1.0)
- [ ] `ham setup`에서 `Notification` hook 자동 추가
- [ ] spec.md의 "waiting_input PTY fallback" 주석 제거, hook 기반으로 갱신
- [ ] Go tests

**Phase 2: SubagentStart/SubagentStop hook**
- [ ] 현재 `PreToolUse "Agent"` / `PostToolUse "Agent"`로 감지하는 서브에이전트를 전용 hook으로 교체
- [ ] `SubagentStart` hook 처리 — stdin에서 `agent_id`, `agent_type` 파싱
- [ ] `SubagentStop` hook 처리 — `agent_transcript_path`로 서브에이전트 작업 결과 요약 가능
- [ ] `ham setup`에서 `SubagentStart`, `SubagentStop` hook 추가
- [ ] Go tests

**Phase 3: StopFailure hook → 에러 분류**
- [ ] `ham hook stop-failure` 서브커맨드 추가
- [ ] `StopFailure` hook 처리 — error type 파싱 (rate_limit, billing_error, server_error 등)
- [ ] Agent에 `error_type` 필드 추가, 디테일 패널에 에러 유형 표시
- [ ] `ham setup`에서 `StopFailure` hook 추가
- [ ] Go tests

**Phase 4: SessionStart/SessionEnd hook**
- [ ] `SessionStart` hook — 세션 시작 시 정확한 session_id 수신 (stdin JSON의 session_id 필드)
- [ ] `SessionEnd` hook — Stop 외에 세션 종료 케이스 추가 커버 (clear, logout, prompt_input_exit 등)
- [ ] `ham setup`에서 `SessionStart`, `SessionEnd` hook 추가
- [ ] Go tests

**Phase 5: hook stdin JSON 파싱 인프라**
- [ ] `ham hook` 커맨드에서 stdin JSON 파싱 지원 (현재는 CLI args만 사용)
- [ ] session_id를 stdin에서 직접 읽어 HAM_AGENT_ID 환경변수 보조/대체 가능하게
- [ ] Go tests

**커밋 + 테스트:**
- [ ] Phase 1~2 완료 후 커밋 (핵심 hook 확장)
- [ ] Phase 3~5 완료 후 커밋 (에러 분류 + 세션 + stdin 파싱)
- [ ] `go test ./...` + `swift build --disable-sandbox` 최종 통과

#### Acceptance Criteria
- [ ] waiting_input이 Notification hook으로 confidence=1.0 감지됨
- [ ] 서브에이전트가 SubagentStart/Stop hook으로 정확하게 추적됨
- [ ] 에러 유형(rate_limit, server_error 등)이 디테일 패널에 표시됨
- [ ] ham setup이 확장된 hook 전체를 자동 설정함
- [ ] 기존 PreToolUse/PostToolUse/Stop 동작이 깨지지 않음

---

## Next Epics (순서대로)

### Epic 28: Agent Teams 연동
Claude Code 내장 Agent Teams 기능과 연동하여 팀 작업을 오피스에서 시각화.

- [ ] `TeammateIdle` hook 처리 — teammate가 idle 전환 시 해당 햄스터 상태 반영
- [ ] `TaskCreated` hook 처리 — 팀 task 생성 시 이벤트 로그 + UI 표시
- [ ] `TaskCompleted` hook 처리 — 팀 task 완료 시 알림 + 이벤트
- [ ] Agent 모델에 `team_role` 필드 추가 (lead/teammate)
- [ ] 팀 리드 햄스터에 왕관/리더 표시, teammate는 팀 뱃지
- [ ] `ham setup`에서 Agent Teams hook 추가 (TeammateIdle, TaskCreated, TaskCompleted)
- [ ] 디테일 패널에 팀 task 진행 상황 표시
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] Agent Teams 모드에서 teammate들이 각각 별도 햄스터로 표시됨
- [ ] 팀 리드와 teammate 구분이 시각적으로 보임
- [ ] task 완료 시 알림이 발송됨

### Epic 29: Worktree 시각화
Claude Code의 git worktree 연동으로 병렬 개발을 오피스에서 표현.

- [ ] `WorktreeCreate` hook 처리 — worktree 생성 시 에이전트에 worktree 정보 연결
- [ ] `WorktreeRemove` hook 처리 — worktree 삭제 시 정리
- [ ] Agent 모델에 `worktree_branch` 필드 추가
- [ ] 디테일 패널에 worktree 브랜치명 표시
- [ ] 오피스에서 같은 프로젝트의 다른 worktree 에이전트를 시각적으로 그루핑
- [ ] `ham setup`에서 WorktreeCreate/Remove hook 추가
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] worktree 기반 병렬 개발 시 각 worktree가 별도 햄스터로 보임
- [ ] 브랜치명이 디테일에 표시됨
- [ ] worktree 삭제 시 정리됨

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
26. ~~Epic 26: 자율 모드 heartbeat 알림~~ ✅
27. **Epic 27: Hook 확장 + 정확도 향상** ← 현재
28. Epic 28: Agent Teams 연동
29. Epic 29: Worktree 시각화
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
