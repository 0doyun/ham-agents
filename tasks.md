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
- [x] **Epic 27: Claude Code 공식 hook 확장 + 정확도 향상** ✅
- [ ] Epic 28: Agent Teams 연동 ← 현재 활성
- [ ] Epic 29: Worktree 시각화
- [ ] Epic 20: 멀티 프로바이더 확장 (Phase 3, 후순위)

---

## Active Scope

**Epic 28: Agent Teams 연동**

기존 Team 인프라 위에 Claude Code Agent Teams hook을 연결해 팀 작업을 오피스에서 시각화한다.

### Current Slice Checklist

**Phase 1: Team hook 수집**
- [ ] `TeammateIdle` hook 처리 — 기존 Team/agent 표현에 teammate idle 상태 반영
- [ ] `TaskCreated` hook 처리 — 팀 task 생성 시 이벤트 로그 + UI 표시
- [ ] `TaskCompleted` hook 처리 — 팀 task 완료 시 알림 + 이벤트
- [ ] `ham setup`에서 Agent Teams hook 추가 (TeammateIdle, TaskCreated, TaskCompleted)
- [ ] Go tests

**Phase 2: Team 표현 연결**
- [ ] Agent 모델에 `team_role` 필드 추가 (lead/teammate)
- [ ] 팀 리드 햄스터에 왕관/리더 표시, teammate는 별도 햄스터처럼 보이되 기존 팀 모델과 연결
- [ ] 디테일 패널에 팀 task 진행 상황 표시
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] Agent Teams 모드에서 teammate들이 각각 별도 햄스터로 표시됨
- [ ] 팀 리드와 teammate 구분이 시각적으로 보임
- [ ] task 완료 시 알림이 발송됨

---

## Next Epics (순서대로)

### Epic 29: Worktree 시각화
Claude Code의 git worktree hook을 받아 metadata-first MVP부터 도입하고, richer visualization은 후속으로 미룬다.

- [ ] `WorktreeCreate` hook 처리 — worktree 생성 시 에이전트에 worktree metadata 연결
- [ ] `WorktreeRemove` hook 처리 — worktree 삭제 시 metadata 정리
- [ ] Agent 모델에 `worktree_branch` 필드 추가
- [ ] 디테일 패널에 worktree 브랜치명 표시
- [ ] richer office grouping은 후속 slice로 미루고, MVP 범위를 metadata + detail 표시로 제한
- [ ] `ham setup`에서 WorktreeCreate/Remove hook 추가
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] worktree metadata가 detail 패널에서 확인 가능함
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
22. ~~Epic 22: 테스트 안정화 + tmux 지원~~ ✅
23. Epic 23: 에이전트 출력 요약
24. Epic 24: OMC 모드 인식
25. Epic 25: 알림 고도화
26. ~~Epic 26: 자율 모드 heartbeat 알림~~ ✅
27. Epic 27: Claude Code 공식 hook 확장 + 정확도 향상
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
