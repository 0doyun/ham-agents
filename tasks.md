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
- [ ] Epic 20: 멀티 프로바이더 확장 (Phase 3)

---

## Active Scope

**Epic 18: Claude Code Hook 기반 상태 추적 (Phase 1)**

방향 전환 배경: 기존 PTY 출력 키워드 매칭 방식은 Claude Code에서 대부분 `thinking`으로 분류되어 상태 판별이 사실상 불가능했다. Claude Code의 hook 시스템을 활용해 100% 정확한 상태 추적으로 전환한다.

### Current Slice Checklist

- [x] `ham hook` IPC 커맨드 정의 (`hook.tool-start`, `hook.tool-done`, `hook.session-end`, `hook.agent-spawned`, `hook.agent-finished`)
- [x] `ipc.go`에 hook 관련 Command 상수 추가
- [x] `server.go`에 hook 커맨드 핸들러 구현
- [x] `cmd/ham/main.go`에 `hook` 서브커맨드 라우팅
- [x] `managed_state.go`에서 hook 이벤트 기반 상태 전환 로직 (기존 키워드 매칭보다 우선)
- [x] `ham setup` 커맨드 구현 (Claude Code 감지 + hooks 설정)
- [x] `~/.claude/settings.json` 안전한 merge 로직 (기존 설정 보존, 사용자 확인)
- [x] 서브에이전트 등록/해제 (Agent tool PreToolUse/PostToolUse 감지)
- [x] 기존 PTY 키워드 추론은 hook 미설정 시 fallback 경로로 유지
- [x] `ham doctor`에 hook 설정 상태 진단 추가
- [x] `go test ./...`

#### Acceptance Criteria
- [x] Claude Code hooks가 설정된 상태에서 `ham run claude` 실행 시, 에이전트 상태가 hook 이벤트 기반으로 정확하게 전환됨 (booting → thinking → reading → running_tool → waiting_input → done 등)
- [x] hook 미설정 시 기존 PTY 키워드 매칭 fallback이 동작함
- [x] `ham setup` 실행 시 Claude Code를 감지하고 `~/.claude/settings.json`에 hooks를 안전하게 추가함
- [x] 서브에이전트 생성/종료 시 자식 햄스터가 등록/해제됨
- [x] `ham doctor`가 hook 설정 여부를 진단하고 미설정 시 안내함

---

## Next Epics (순서대로)

### Epic 19: 단일 오피스 UI 재설계 (Phase 2)
4존 그리드(Desk/Library/Kitchen/Alert Corner) → 단일 오피스 공간으로 변경.

- [x] 4존 그리드 레이아웃 제거, 단일 오피스 캔버스 구현
- [x] 가구 배치 (책상, 책장, 소파, 경고등) 및 영역 암시
- [x] 상태 기반 햄스터 위치 배치 로직 (가구 근처 자연 배치)
- [x] 미니 햄스터 렌더링 (서브에이전트 시각화)
- [x] 머리 위 상태 아이콘 (⚠️ ❓ ✅)
- [x] 도구별 스프라이트 분화 (Read → 책 읽기, Bash → 타이핑)
- [x] 정확한 상태 기반 알림 개선
- [x] Swift tests

#### Acceptance Criteria
- [x] 단일 오피스 공간에 햄스터들이 가구 근처에 자연스럽게 배치됨
- [x] 서브에이전트가 부모 근처 미니 햄스터로 보임
- [x] 상태가 스프라이트 + 위치 + 아이콘 3가지로 전달됨
- [x] 에이전트 6개 이상에서도 공간이 부족하지 않음

### Epic 20: 멀티 프로바이더 확장 (Phase 3)
Claude Code 이외 프로바이더 지원 추가.

- [ ] Codex 전용 어댑터
- [ ] Gemini CLI 전용 어댑터
- [ ] `ham setup codex`, `ham setup gemini`
- [ ] 범용 추론 엔진은 hook 미지원 프로바이더 fallback으로 유지
- [ ] Go tests

#### Acceptance Criteria
- [ ] Codex/Gemini CLI로 `ham run` 시 해당 어댑터가 상태를 정확하게 추론함
- [ ] `ham setup`이 각 프로바이더별 설정을 안내함

### Epic 10: Team and Workspace
spec §6 Team/Workspace, §12 `ham team` CLI, §11 팀 요약 알림, §14 team 단위 focus.

- [ ] Team domain model 추가 (Go core) — team_id, display_name, member agent_ids
- [ ] Workspace domain model 추가 — project_path 기반 자동 그룹핑
- [ ] `ham team create <name>` / `ham team add <name> <agent>` CLI
- [ ] daemon IPC에 team CRUD surface 추가
- [ ] `ham ask <team> "..."` — team 대상 메시지 브로드캐스트
- [ ] team 단위 focus — team의 agent들을 한 번에 열기 (§14 Should)
- [ ] 팀 요약 알림 — team 전체 상태 요약 notification (§11)
- [ ] menu bar popover에 workspace/team filter 추가
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] agent를 team으로 묶을 수 있음
- [ ] CLI와 menu bar에서 team/workspace 단위로 필터/조회 가능
- [ ] team 없는 agent도 정상 동작
- [ ] team 대상 메시지가 모든 멤버에게 전달됨

### Epic 11: Managed Process Lifecycle
`ham run`이 실제 provider 세션을 spawn하고, `ham stop`이 실제로 종료하는 것. spec §7 Managed, §12 `ham run`/`ham stop`, §13 데이터 흐름.

- [ ] `ham run`이 실제 child process를 spawn (provider별 command 결정)
- [ ] process stdout/stderr를 structured event로 수집
- [ ] structured launch events를 daemon event log로 연결 (§15)
- [ ] process exit 감지 → done/error 상태 전이
- [ ] `ham stop`이 실제 process signal/termination 수행
- [ ] managed mode의 high-confidence 상태 추론 — structured events 기반 (§15)
- [ ] Go tests

#### Acceptance Criteria
- [ ] `ham run claude --project ... --role ...`로 실제 세션이 뜸
- [ ] 세션 종료가 자동으로 agent 상태에 반영됨
- [ ] `ham stop`이 진짜 세션을 멈춤
- [ ] managed agent는 structured events 덕분에 highest confidence

### Epic 12: Pixel Office Experience
spec §8 메뉴바 햄스터, §9 오피스 UI, §17 Appearance 중 sprite 관련. 제품의 핵심 비주얼.

- [ ] `avatar_variant` 필드를 agent model에 추가 (§6)
- [ ] 메뉴바 햄스터 아이콘 애니메이션 — idle/running/waiting/error/done 상태별 (§8)
- [ ] room layout 구현 (Desk/Library/Kitchen/Alert zone) (§9)
- [ ] sprite asset 기본 세트 (idle/walk/run/type/read/think/sleep/celebrate/alert/error) (§9)
- [ ] 상태 → zone/animation 매핑 (§9)
- [ ] SpriteKit 또는 Canvas 기반 렌더링
- [ ] popover 내 캔버스 통합
- [ ] Appearance 설정 — animation speed multiplier, reduce motion (§17)
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] 메뉴바 아이콘이 상태에 따라 시각적으로 변화
- [ ] popover에 pixel office가 렌더링됨
- [ ] 상태가 시각적으로 구분됨
- [ ] 귀여움이 정보 전달을 가리지 않음

### Epic 13: Notification Completeness
spec §11 알림 스펙의 누락분. 현재 기본 알림은 동작하지만 고급 정책이 빠져 있음.

- [ ] 팀 요약 알림 — Epic 10 이후 team이 있을 때 team 단위 digest (§11)
- [ ] 상태 flap bundling — 같은 agent가 짧은 시간에 상태를 왕복하면 묶어서 1건 처리 (§11)
- [ ] 연속 유사 알림 dedupe 강화 — 현재 transition-based dedupe 위에 time-window dedupe 추가 (§11)
- [ ] notification history 저장 — 과거 알림 이력을 store에 기록 (§16)
- [ ] `done` 알림을 long-running task에만 제한하는 정책 (§11 기본 정책)
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] 상태가 빠르게 왕복해도 알림이 1건만 옴
- [ ] 과거 알림 이력을 조회할 수 있음
- [ ] team 요약 알림이 동작함

### Epic 14: Settings Completeness
spec §17 설정 화면의 누락분. 현재 notifications/appearance.theme/integrations.iterm_enabled만 구현됨.

- [ ] General — Launch at login (§17)
- [ ] General — Compact mode (§17)
- [ ] General — Show menu bar animation always (§17)
- [ ] Integrations — Transcript directories 설정 (§17)
- [ ] Integrations — Provider adapters on/off (§17)
- [ ] Privacy — local-only mode (§17)
- [ ] Privacy — event history retention period (§17)
- [ ] Privacy — transcript excerpt storage on/off (§17)
- [ ] Appearance — 햄스터 스킨/모자/책상 테마 (§17, Epic 12 이후)
- [ ] daemon settings schema 확장 + CLI/Swift round-trip
- [ ] Go/Swift tests

#### Acceptance Criteria
- [ ] spec §17의 5개 섹션이 모두 동작함
- [ ] CLI와 menu bar에서 모든 설정을 수정 가능

### Epic 15: Provider Adapter Layer
spec §13 adapter layer, §15 provider-specific adapter 힌트. 현재 iTerm2 adapter만 있고 generic process adapter / transcript adapter가 없음.

- [ ] transcript adapter — transcript/log 디렉터리 감시, 파일 변경 기반 event 생성 (§13, §15)
- [ ] generic process adapter — process exit/signal 감지 (§13)
- [ ] provider-specific adapter 힌트 — Claude CLI 등 known provider의 structured output 파싱 (§15)
- [ ] adapter on/off를 settings에서 제어 (§17 Integrations)
- [ ] Go tests

#### Acceptance Criteria
- [ ] transcript 디렉터리를 watch하면 observed agent 상태가 자동 갱신됨
- [ ] known provider의 structured output이 higher-confidence 추론에 반영됨

### Epic 16: Final Polish and Performance
spec의 나머지 품질 요구사항. 모든 기능 epic 완료 후 실행.

- [ ] exportable logs — `ham logs --export` 또는 파일 내보내기 (§20 v1.0)
- [ ] detach/reattach UX 개선 (§20 v1.0)
- [ ] 알림 flap bundling 고도화 (§11)
- [ ] observed inference heuristic 고도화 (§15)
- [ ] 민감 경로/환경변수 마스킹 (§16)
- [ ] 성능 목표 검증 및 최적화 (§19) — idle CPU <2%/<1%, 메모리 <150MB/<100MB, 팝오버 <200ms
- [ ] 디자인 polish / sprite variation
- [ ] iTerm2 레이아웃 변경 감지 (§14, Won't v1 이지만 best-effort)
- [ ] Go/Swift tests + 성능 벤치마크

#### Acceptance Criteria
- [ ] spec §19 성능 목표 달성
- [ ] 민감 정보가 마스킹됨
- [ ] 전체 UX 플로우(§18)가 end-to-end로 동작

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
20. Epic 20: 멀티 프로바이더 확장 (Phase 3)

---

## Spec Coverage Map
어떤 epic이 spec의 어떤 섹션을 커버하는지 참조용.

| Spec 섹션 | 커버하는 Epic |
|-----------|-------------|
| §6 Agent 필드 | Epic 2 ✅ + Epic 12 (`avatar_variant`) |
| §6 Team / Workspace | Epic 10 |
| §7 세션 모드 (Managed/Attached/Observed) | Epic 7 ✅ + Epic 11 (managed lifecycle) |
| §8 메뉴바 상시 경험 | Epic 4 ✅ + Epic 12 (아이콘 애니메이션) |
| §9 오피스 UI | Epic 12 |
| §10 상태 머신 | Epic 8 ✅ |
| §11 알림 | Epic 5 ✅ + Epic 13 (flap/team summary/history) |
| §12 CLI | Epic 2–8 ✅ + Epic 10 (`ham team`) + Epic 16 (`--export`) |
| §13 아키텍처 | Epic 3 ✅ + Epic 15 (adapter layer) |
| §14 iTerm2 연동 | Epic 6 ✅ + Epic 10 (team focus) + Epic 16 (레이아웃 감지) |
| §15 상태 추론 엔진 | Epic 8 ✅ + Epic 11 (structured events) + Epic 15 (provider adapter) |
| §16 저장/프라이버시 | Epic 3 ✅ + Epic 13 (notification history) + Epic 14 (Privacy 설정) + Epic 16 (마스킹) |
| §17 설정 화면 | Epic 5 ✅ (일부) + Epic 14 (전체) |
| §18 UX 플로우 | Epic 16 (end-to-end 검증) |
| §19 성능 목표 | Epic 16 |
| §20 단계별 범위 | Epic 12 (v0.2 pixel office) + Epic 16 (v1.0 exportable logs, detach/reattach) |

---

## Notes
- `spec.md`가 최종 목표 문서다.
- `roadmap.md`는 참고용이며 현재 범위를 제한하지 않는다.
- UI는 Swift, CLI/runtime은 Go로 분리하는 방향을 현재 기준 아키텍처로 본다.
- Ralph/autonomous 실행은 항상 가장 높은 우선순위의 미완료 epic부터 이어간다.
- 각 slice 완료 시 `docs/progress.md`, `docs/assumptions.md`, 테스트 결과를 함께 갱신한다.
- Active Scope 체크리스트가 전부 완료되면 다음 epic으로 이동한다. 완료된 scope를 더 다듬지 않는다.
