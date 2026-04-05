# ham-agents 현황 분석서

> Step 1 산출물 | 2026-04-06 작성 | dev/detailed-plan 브랜치

---

## 목차

1. [아키텍처 정밀 분석](#1-아키텍처-정밀-분석)
2. [데이터 모델 분석](#2-데이터-모델-분석)
3. [현재 기능 목록과 한계](#3-현재-기능-목록과-한계)
4. [외부 의존성 및 연동 분석](#4-외부-의존성-및-연동-분석)

---

## 1. 아키텍처 정밀 분석

### 1-1. Go 백엔드 전체 구조

```
go/
├── cmd/
│   ├── hamd/          # 데몬 프로세스 (233줄 + pidfile 73줄)
│   └── ham/           # CLI (27개 명령, 2,700+줄)
└── internal/
    ├── core/          # 데이터 모델 (Agent, Event, Settings, Team, Workspace)
    ├── runtime/       # 상태 엔진 (Registry, ManagedService, hooks, events, attention)
    ├── ipc/           # Unix 소켓 IPC (Command 52개, Server dispatch)
    ├── store/         # 파일 기반 영속화 (JSON, JSONL)
    ├── adapters/      # iTerm2, tmux, transcript, quick message
    └── inference/     # Observed 에이전트 상태 추론
```

총 48개 비테스트 Go 파일, 약 16,122줄.

#### go/cmd/hamd/ - 데몬 엔트리 포인트

| 파일 | 역할 |
|------|------|
| `main.go` (233줄) | 데몬 시작, IPC 서버 + 폴링 루프 실행, SIGTERM 처리 |
| `pidfile.go` (73줄) | flock 기반 중복 인스턴스 방지 |

**시작 순서**: PID lock 획득 → Registry/ManagedService/SettingsService/TeamService 생성 → iTerm2/Tmux/Transcript 어댑터 생성 → IPC Server 시작 → pollRuntimeState 고루틴 (2초 간격)

**폴링 루프**: RefreshObserved → ensureObservedTranscripts → itermAdapter.ListSessions → tmuxAdapter.ListSessions → emitHeartbeatEvents

**launchd 통합**: `com.ham-agents.hamd` 라벨, RunAtLoad=true, KeepAlive.SuccessfulExit=false

#### go/cmd/ham/ - CLI 명령 (27개)

| 명령 | 파일 | 목적 |
|------|------|------|
| `ham run <provider> [name]` | `pty.go` | 관리형 에이전트 등록, PTY에서 provider 실행 |
| `ham attach <session-ref>` | `parse.go` | iTerm2/tmux 세션 연결 |
| `ham attach --pick-iterm-session` | `parse.go` | 대화형 iTerm 세션 선택 |
| `ham attach --pick-tmux-session` | `parse.go` | 대화형 tmux 패널 선택 |
| `ham observe <source-ref>` | `parse.go` | 트랜스크립트 관찰 등록 |
| `ham open <agent-id>` | `parse.go` | 에이전트 세션 열기 |
| `ham ask <agent> <message>` | `parse.go` | 에이전트에 메시지 전송 |
| `ham stop <agent-id>` | `parse.go` | 관리형 에이전트 중지 (SIGTERM) |
| `ham detach <agent-id>` | `parse.go` | 에이전트 추적 해제 |
| `ham rename <agent-id> <name>` | `parse.go` | 에이전트 이름 변경 |
| `ham logs <agent-id>` | `parse.go` | 이벤트 히스토리 조회/내보내기 |
| `ham list` | `parse.go` | 에이전트 목록 (--team, --workspace 필터) |
| `ham status` | `parse.go` | RuntimeSnapshot + attention 카운트 |
| `ham events` | `parse.go` | 이벤트 로그 (--follow 롱폴링) |
| `ham team create/add/list/open` | `parse.go` | 팀 관리 |
| `ham hook <type>` | `parse.go` | Claude Code hook 이벤트 처리 (27개 하위명령) |
| `ham setup` | `setup.go` | 데몬/hooks/launchd 설치 |
| `ham settings [category]` | `parse.go` | 설정 조회/변경 |
| `ham doctor` | `doctor.go` | 진단 리포트 |
| `ham ui` | `ui.go` | 메뉴바 앱 실행 |
| `ham down` | `parse.go` | 메뉴바/데몬/launchd 종료 |
| `ham uninstall` | `parse.go` | 전체 제거 |

#### go/internal/core/ - 데이터 모델

| 파일 | 역할 |
|------|------|
| `agent.go` (232줄) | Agent, ToolActivity, SubAgentInfo, RuntimeSnapshot, Event, EventType, AttachableSession, OpenTarget |
| `settings.go` (302줄) | Settings (General/Notifications/Appearance/Integrations/Privacy) + 검증 |
| `team.go` (15줄) | Team 구조체 |
| `workspace.go` (90줄) | Workspace (에이전트 기반 동적 빌드) |
| `status_helpers.go` (74줄) | 상태 분류 헬퍼 (IsRunning, RequiresAttention, AttentionSeverity) |

#### go/internal/runtime/ - 상태 엔진

| 파일 | 역할 |
|------|------|
| `registry.go` (371줄) | 중앙 상태 관리. mutateAgent 패턴, Snapshot, stale 감지 (5분 무활동→disconnected) |
| `managed_state.go` (933줄) | 모든 RecordHook* / RecordManaged* 메서드. 27개 hook 핸들러 |
| `managed.go` (195줄) | ManagedService - OS 프로세스 관리 (PTY, SIGTERM) |
| `registration.go` (288줄) | 에이전트 등록 (Managed/Attached/Observed). 중복 방지 |
| `registry_attached.go` (292줄) | 연결 에이전트 갱신, disconnect/reconnect 감지, OpenTarget 해석 |
| `registry_observed.go` (73줄) | 관찰 에이전트 갱신 |
| `events.go` (169줄) | 이벤트 조회/FollowEvents 롱폴링 (200ms 간격, 60초 max) |
| `attention.go` (87줄) | attention 분석 (error>waitingInput>disconnected 우선순위) |
| `settings.go` (37줄) | SettingsService |
| `team.go` (101줄) | TeamService CRUD |
| `transcript.go` (45줄) | 트랜스크립트 자동 감지 |

**mutateAgent 패턴** (registry.go:239-286):
```
Lock → LoadAgents → FindByID → SnapshotBefore → CallMutator → SaveAgents + AppendEvent → Unlock
```

**RecordHook* 메서드별 상태 전이 요약**:

| Hook | 결과 상태 | 핵심 동작 |
|------|-----------|-----------|
| `RecordHookToolStart` | reading/writing/searching/spawning/running_tool | 도구명으로 상태 매핑, RecentTools 갱신 |
| `RecordHookToolDone` | thinking | 도구 완료, duration 기록 |
| `RecordHookNotification` | waiting_input (permission) / idle (idle_prompt) | 알림 유형별 분기 |
| `RecordHookStopFailure` | error | ErrorType 설정 |
| `RecordHookSessionStart` | booting | SessionID 기록, **에이전트 자동 등록** |
| `RecordHookSessionEnd` | done | **에이전트 추적 해제** (Remove) |
| `RecordHookStop` | idle | LastAssistantMessage 저장 |
| `RecordHookAgentSpawned` | (변경없음) | SubAgentCount++, SubAgents 추가 (최대 20) |
| `RecordHookAgentFinished` | (변경없음) | SubAgentCount--, SubAgent 완료 처리 |
| `RecordHookTeammateIdle` | (변경없음) | TeamRole 설정, team.teammate_idle 이벤트 |
| `RecordHookTaskCreated` | (변경없음) | TeamRole="lead", TeamTaskTotal++ |
| `RecordHookTaskCompleted` | (변경없음) | TeamTaskCompleted++. 스마트 라우팅: 태스크 없는 에이전트→팀 리드 찾기 |
| `RecordHookToolFailed` | waiting_input (interrupt) / thinking | isInterrupt 분기 |
| `RecordHookUserPrompt` | thinking | 프롬프트 미리보기 (50자) |
| `RecordHookPermissionRequest` | waiting_input | "Approve <tool>?" |
| `RecordHookPermissionDenied` | (변경없음) | "Permission denied: <tool>" |
| `RecordHookPreCompact` | (변경없음) | "Compacting context..." |
| `RecordHookPostCompact` | thinking | compact summary 포함 |
| `RecordHookElicitation` | waiting_input | MCP 사용자 입력 대기 |
| `RecordHookElicitationResult` | thinking | 결과 수신 |
| `RecordHookConfigChange` | (변경없음) | source 포함 |
| `RecordHookWorktreeCreate/Remove` | (변경없음) | worktree 이름/경로 |
| `RecordHookCwdChanged` | (변경없음) | **ProjectPath 업데이트** |
| `RecordHookInstructionsLoaded` | (변경없음) | 파일 경로 |
| `RecordHookFileChanged` | (변경없음) | 파일 경로 + 이벤트 |
| `RecordHookSetup` | (변경없음) | "Setup hook fired." |

**도구명→상태 매핑** (RecordHookToolStart):
- `Read`, `Grep`, `Glob` → reading
- `Write`, `Edit`, `NotebookEdit` → writing
- `WebFetch`, `WebSearch` → searching
- `Agent` → spawning
- 나머지 → running_tool

#### go/internal/ipc/ - IPC 서버

| 파일 | 역할 |
|------|------|
| `ipc.go` (605줄) | 전체 Command 상수 52개, Request/Response 스키마, Client 구현 |
| `server.go` (733줄) | Unix 소켓 서버, dispatch 테이블 |

**프로토콜**: Unix 도메인 소켓, 연결당 1회 JSON Request → JSON Response. 클라이언트 타임아웃 3초.

**소켓 경로**: `HAM_AGENTS_SOCKET` → `HAM_AGENTS_HOME/hamd.sock` → `~/Library/Application Support/ham-agents/hamd.sock`

**prepareHookRequest 자동 등록**: SessionID로 에이전트 검색 → 없으면 `session-start` hook일 때 자동으로 managed 에이전트 생성 (provider="claude", 프로젝트 경로에서 이름 도출)

#### go/internal/store/ - 영속화

| 파일 | 역할 |
|------|------|
| `store.go` (118줄) | FileAgentStore - `managed-agents.json` |
| `events.go` (168줄) | FileEventStore - `events.jsonl` (JSONL, 최대 10,000줄) |
| `settings.go` (97줄) | FileSettingsStore - `settings.json` |
| `team.go` (98줄) | FileTeamStore - `teams.json` |

모든 스토어: mutex + atomic write (tmp → rename). 파일 없으면 기본값 반환.

**저장 경로** (`HAM_AGENTS_HOME` 오버라이드):
| 파일 | 기본 경로 |
|------|-----------|
| 에이전트 상태 | `~/Library/Application Support/ham-agents/managed-agents.json` |
| 이벤트 로그 | `~/Library/Application Support/ham-agents/events.jsonl` |
| 설정 | `~/Library/Application Support/ham-agents/settings.json` |
| 팀 | `~/Library/Application Support/ham-agents/teams.json` |
| 소켓 | `~/Library/Application Support/ham-agents/hamd.sock` |
| PID | `~/Library/Application Support/ham-agents/hamd.sock.pid` |
| launchd | `~/Library/LaunchAgents/com.ham-agents.hamd.plist` |
| 데몬 로그 | `~/Library/Logs/ham-agents/hamd.log` |

#### go/internal/adapters/ - 터미널 어댑터

| 파일 | 역할 |
|------|------|
| `iterm2.go` (301줄) | AppleScript로 iTerm2 세션 목록/포커스. ps+lsof로 프로세스/CWD 보강 |
| `tmux.go` (267줄) | tmux CLI로 세션/윈도우/패널 열거. `tmux://session:window.pane` 형식 |
| `transcript.go` (58줄) | 디렉토리 순회로 .log/.txt/.jsonl/.md 트랜스크립트 감지 |
| `provider_hints.go` (29줄) | Claude 구조화 출력 JSON 파싱 (type → status 매핑) |
| `provider_output.go` (99줄) | 출력 키워드 기반 상태 추론 |
| `generic_process.go` (33줄) | 프로세스 종료 분류 |
| `quick_message.go` (154줄) | 에이전트에 메시지 전송 (AppleScript/tmux send-keys/클립보드) |

### 1-2. Swift 프론트엔드 전체 구조

```
Sources/
├── HamCore/              # 공유 모델 (Agent, DaemonIPC, Payloads)
├── HamAppServices/       # 뷰모델, EventPresentation, PixelOffice, DaemonClient
├── HamNotifications/     # macOS 알림 전달
├── HamPersistence/       # 인메모리 AgentStore (실제 영속화 아님)
├── HamRuntime/           # CLI용 로컬 런타임 (메뉴바 앱에서는 미사용)
├── HamInference/         # 관찰 에이전트 상태 추론 엔진
├── HamAdapters/          # 어댑터 (Package.swift에 정의)
└── HamCLI/               # CLI 모듈 (Package.swift에 정의)

apps/macos/HamMenuBarApp/Sources/
├── HamMenuBarApp.swift   # 앱 진입점, 윈도우 관리
├── MenuBarPlatform.swift # 플랫폼 구현 (iTerm/tmux/Workspace 오프너)
├── MenuBarViews.swift    # 전체 SwiftUI 뷰 (1,101줄)
└── PixelOfficeView.swift # 픽셀 아트 사무실 (1,025줄)
```

Swift tools version 5.10, macOS 13+ 타겟. 6개 라이브러리, 2개 실행파일.

**모듈 의존성 그래프**:
```
HamCore (기반 - 무의존)
├── HamPersistence
├── HamInference
├── HamNotifications
├── HamAdapters
├── HamAppServices (→ HamCore + HamNotifications)
├── HamRuntime (→ HamCore + HamPersistence)
├── HamCLI (→ HamCore + HamPersistence + HamRuntime)
└── HamMenuBarApp (→ HamAppServices + HamCore + HamNotifications)
```

#### Sources/HamCore/ - 공유 모델

| 파일 | 역할 |
|------|------|
| `Agent.swift` | Agent 구조체 (33 필드), AgentMode, AgentStatus (13 케이스), NotificationPolicy, ToolActivity |
| `DaemonIPC.swift` | DaemonCommand (19 케이스), DaemonRequest/Response, 설정 페이로드 |
| `DaemonPayloads.swift` | DaemonStatusPayload, AgentEventPayload, DaemonJSONDecoder |
| `AgentStatusPresentation.swift` | 상태 표시 헬퍼 (humanizedLabel, isRunningActivity, isActiveWork) |

#### Sources/HamAppServices/ - 뷰모델 + 비즈니스 로직

| 파일 | 역할 |
|------|------|
| `DaemonClient.swift` | Unix 소켓 트랜스포트, HamDaemonClient (11 메서드), MenuBarSummaryService |
| `MenuBarViewModel.swift` (933줄) | 중앙 @MainActor ViewModel. 2개 폴링 태스크: 5초 refresh + 15초 event follow |
| `PixelOfficeModel.swift` | 상태→스프라이트 매핑 (OfficeArea, HamsterSpriteState, PixelOfficeMapper) |
| `EventPresentation.swift` | AgentEventPayload→AgentEventPresentation 변환 |
| `SessionTargeting.swift` | SessionTarget (iterm/tmux/url/workspace) 해석 |
| `ProjectOpening.swift` | 프로젝트 열기 프로토콜 |
| `SessionOpening.swift` | 세션 열기 프로토콜 |
| `ItermAppleScripts.swift` | iTerm2 AppleScript 생성 |
| `QuickMessageSending.swift` | 에이전트 메시지 전달 (터미널/클립보드) |

**MenuBarViewModel 폴링 사이클**:
```
start()
├── refreshTask (5초 간격)
│   └── fetchSnapshot + fetchAgents + fetchSettings + fetchSessions + fetchTeams (병렬 async let)
│       → 알림 파이프라인: 이전/현재 에이전트 비교 → StatusChangeNotificationEngine → 필터링 → 발송
└── eventFollowTask (15초 wait 롱폴링)
    └── followLatestEvents() → 새 이벤트 있으면 fetchAgents → merge → applyRefreshedState
```

**StatusBarTint 우선순위**: red (error) > yellow (waitingInput) > blue (activeWork) > green (all done) > gray (idle/empty)

#### Sources/HamNotifications/ - macOS 알림

| 파일 | 역할 |
|------|------|
| `HamNotificationService.swift` | StatusChangeNotificationEngine - 상태 변화 감지 (done/error/waitingInput/silence/heartbeat) |
| `UserNotificationSink.swift` | UNUserNotificationCenter 브릿지. "Open Terminal"/"Dismiss" 액션 |
| `NotificationHistory.swift` | 중복 방지 히스토리 (최대 200개, 파일 영속화) |

**알림 트리거**: done, error, waitingInput (상태 전이), silence (10분 무이벤트+실행중), heartbeat (OMC모드 설정 간격)

#### apps/macos/HamMenuBarApp/ - SwiftUI 뷰

| 파일 | 역할 |
|------|------|
| `HamMenuBarApp.swift` | @main App, MenuBarExtra, accessory 활성화 정책, NSWindow 관리 |
| `MenuBarPlatform.swift` | PreviewDaemonClient, WorkspaceProjectOpener, ItermSessionOpener, ItermQuickMessageSender |
| `MenuBarViews.swift` (1,101줄) | 전체 뷰 계층구조 (아래 참조) |
| `PixelOfficeView.swift` (1,025줄) | 픽셀 아트 사무실, MenuBarHamsterGlyph (18x18), 애니메이션 스프라이트 |

**뷰 계층구조**:
```
MenuBarContentView
├── officeContent
│   ├── SummaryBadge (Total/Run/Wait)
│   ├── PixelOfficeView
│   ├── AttentionAgentRow (error/waiting/disconnected)
│   ├── AgentListCard (compact agent card)
│   └── AgentDetailView
│       ├── StatusBadge + OmcModeBadge + TeamRoleBadge
│       ├── EventSummaryChipsView
│       ├── Quick Message Field
│       └── 액션 버튼 (open project/session, notifications, stop)
└── settingsContent
    ├── NotificationPermissionRow
    ├── GeneralSettingsSection
    ├── NotificationSettingsSection
    ├── AppearanceSettingsSection
    ├── IntegrationSettingsSection
    ├── PrivacySettingsSection
    └── AttachableSessionsSection
```

**PixelOfficeView 스프라이트**:
- 9개 상태: idle, walk, run, type, read, think, sleep, alert, error
- 4개 스킨: default, night, golden, mint
- 데스크 소품: 모니터+커피(thinking), 책(reading), 빨간 모니터(error), 주황 모니터(waitingInput), 닫힌 노트북(idle/sleeping), 연필+종이(writing), 돋보기(searching), 스포닝 인디케이터(spawning)
- 서브에이전트: 주 햄스터 뒤에 최대 6마리 미니 햄스터
- 팀 표시: 리드=왕관, 팀원=사람 아이콘, 태스크 진행률 카운터

### 1-3. IPC 프로토콜 상세

#### 전체 커맨드 목록 (52개)

**에이전트 생명주기 (5개)**:
| 커맨드 | JSON 값 | 용도 |
|--------|---------|------|
| CommandRunManaged | `run.managed` | 관리형 에이전트 등록 + 프로세스 시작 |
| CommandRegisterManaged | `register.managed` | 관리형 에이전트 등록만 (프로세스 시작 없이) |
| CommandNotifyManagedExited | `managed.exited` | 프로세스 종료 알림 |
| CommandRecordOutput | `managed.output` | 프로세스 출력 기록 |
| CommandStopManaged | `managed.stop` | 프로세스 종료 (SIGTERM) |

**에이전트 연결/관찰 (2개)**:
| CommandAttachSession | `attach.session` | 기존 세션 연결 |
| CommandObserveSource | `observe.source` | 트랜스크립트 관찰 |

**에이전트 조회/변경 (6개)**:
| CommandListAgents | `agents.list` | 에이전트 목록 |
| CommandStatus | `agents.status` | RuntimeSnapshot |
| CommandOpenTarget | `agents.open_target` | 세션 열기 대상 해석 |
| CommandSetNotificationPolicy | `agents.set_notification_policy` | 알림 정책 변경 |
| CommandSetRole | `agents.set_role` | 역할 변경 |
| CommandRenameAgent | `agents.rename` | 이름 변경 |
| CommandRemoveAgent | `agents.remove` | 추적 해제 |

**이벤트 (2개)**:
| CommandEvents | `events.list` | 이벤트 목록 (limit, afterEventID) |
| CommandFollowEvents | `events.follow` | 롱폴링 (200ms 간격, 60초 max wait) |

**팀 (3개)**:
| CommandCreateTeam | `teams.create` | 팀 생성 |
| CommandAddTeamMember | `teams.add_member` | 팀 멤버 추가 |
| CommandListTeams | `teams.list` | 팀 목록 |

**세션 (2개)**:
| CommandListItermSessions | `iterm.sessions` | iTerm2 세션 목록 |
| CommandListTmuxSessions | `tmux.sessions` | tmux 패널 목록 |

**설정 (2개)**:
| CommandGetSettings | `settings.get` | 설정 조회 |
| CommandUpdateSettings | `settings.update` | 설정 변경 |

**데몬 (1개)**:
| CommandShutdown | `daemon.shutdown` | 데몬 종료 |

**Hook 커맨드 (29개)**:
| 커맨드 | JSON 값 |
|--------|---------|
| CommandHookToolStart | `hook.tool-start` |
| CommandHookToolDone | `hook.tool-done` |
| CommandHookToolFailed | `hook.tool-failed` |
| CommandHookNotification | `hook.notification` |
| CommandHookStopFailure | `hook.stop-failure` |
| CommandHookSessionStart | `hook.session-start` |
| CommandHookSessionEnd | `hook.session-end` |
| CommandHookStop | `hook.stop` |
| CommandHookAgentSpawned | `hook.agent-spawned` |
| CommandHookAgentFinished | `hook.agent-finished` |
| CommandHookTeammateIdle | `hook.teammate-idle` |
| CommandHookTaskCreated | `hook.task-created` |
| CommandHookTaskCompleted | `hook.task-completed` |
| CommandHookUserPrompt | `hook.user-prompt` |
| CommandHookPermissionReq | `hook.permission-request` |
| CommandHookPermissionDenied | `hook.permission-denied` |
| CommandHookPreCompact | `hook.pre-compact` |
| CommandHookPostCompact | `hook.post-compact` |
| CommandHookSetup | `hook.setup` |
| CommandHookElicitation | `hook.elicitation` |
| CommandHookElicitationResult | `hook.elicitation-result` |
| CommandHookConfigChange | `hook.config-change` |
| CommandHookWorktreeCreate | `hook.worktree-create` |
| CommandHookWorktreeRemove | `hook.worktree-remove` |
| CommandHookInstructions | `hook.instructions-loaded` |
| CommandHookCwdChanged | `hook.cwd-changed` |
| CommandHookFileChanged | `hook.file-changed` |

#### Request 스키마

```go
type Request struct {
    Command          Command
    AgentID          string
    Provider         string
    DisplayName      string
    ProjectPath      string
    Role             string
    SessionRef       string
    TeamRef          string
    MemberAgentID    string
    Limit            int
    AfterEventID     string
    WaitMillis       int
    Policy           string
    Settings         *core.Settings
    ExitError        string
    OutputLine       string
    ToolName         string
    ToolInputPreview string
    OmcMode          string
    SessionID        string
    NotificationType string
    ErrorType        string
    HookType         string
    Description      string
    TeammateName     string
    TeamRole         string
    TaskName         string
    TaskDescription  string
    IsInterrupt      bool
    Prompt           string
    CompactSummary   string
    CompactTrigger   string
    WorktreeName     string
    WorktreePath     string
    OldCwd           string
    NewCwd           string
    FilePath         string
    FileEvent        string
    LastMessage      string
}
```

#### Response 스키마

```go
type Response struct {
    Agent              *core.Agent
    Team               *core.Team
    Agents             []core.Agent
    Teams              []core.Team
    Events             []core.Event
    AttachableSessions []core.AttachableSession
    OpenTarget         *core.OpenTarget
    Settings           *core.Settings
    Snapshot           *core.RuntimeSnapshot
    Error              string
}
```

### 1-4. Hook 시스템 상세

**27개 hook 타입** (CLI `ham hook <type>` → IPC Command → Registry.RecordHook*):

각 hook의 **입력 페이로드**는 `hookPayload` 구조체에서 읽음 (stdin JSON):

```go
// go/cmd/ham/main.go:433-460
type hookPayload struct {
    SessionID           string            `json:"session_id"`
    Cwd                 string            `json:"cwd"`
    ToolName            string            `json:"tool_name"`
    ToolInput           map[string]any    `json:"tool_input"`
    NotificationType    string            `json:"notification_type"`
    ErrorType           string            `json:"error_type"`
    AgentID             string            `json:"agent_id"`
    AgentType           string            `json:"agent_type"`
    AgentTranscriptPath string            `json:"agent_transcript_path"`
    TeammateName        string            `json:"teammate_name"`
    TeamRole            string            `json:"team_role"`
    TaskName            string            `json:"task_name"`
    TaskDescription     string            `json:"task_description"`
    Error               string            `json:"error"`
    IsInterrupt         bool              `json:"is_interrupt"`
    Prompt              string            `json:"prompt"`
    Trigger             string            `json:"trigger"`
    CompactSummary      string            `json:"compact_summary"`
    Name                string            `json:"name"`      // worktree
    WorktreePath        string            `json:"worktree_path"`
    OldCwd              string            `json:"old_cwd"`
    NewCwd              string            `json:"new_cwd"`
    FilePath            string            `json:"file_path"`
    Event               string            `json:"event"`     // file event
    Source              string            `json:"source"`    // config change
    LastAssistantMessage string           `json:"last_assistant_message"`
}
```

**Hook 페이로드에 없는 데이터** (기획 시 주의):
- 토큰/비용 데이터 없음
- 파일 내용/diff 없음 (경로만)
- parent_id 없음 (SubAgents 리스트로 간접 추론)
- 모델 정보 없음 (어떤 Claude 모델 사용 중인지)
- 세션 전체 대화 내용 없음 (transcript_path는 있음)

---

## 2. 데이터 모델 분석

### 2-1. Agent 구조체 Go/Swift 필드 대조

| Go 필드 | JSON 키 | Swift 필드 | 불일치 |
|---------|---------|------------|--------|
| `ID` | `id` | `id` | - |
| `DisplayName` | `display_name` | `displayName` | - |
| `Provider` | `provider` | `provider` | - |
| `Host` | `host` | `host` | - |
| `Mode` | `mode` | `mode` | - |
| `ProjectPath` | `project_path` | `projectPath` | - |
| `Role` | `role` | `role` | - |
| `Status` | `status` | `status` | - |
| `StatusConfidence` | `status_confidence` | `statusConfidence` | - |
| `StatusReason` | `status_reason` | `statusReason` | - |
| `ErrorType` | `error_type` | `errorType` | - |
| `RegisteredAt` | `registered_at` | `registeredAt` | - |
| `LastEventAt` | `last_event_at` | `lastEventAt` | - |
| `LastUserVisibleSummary` | `last_user_visible_summary` | `lastUserVisibleSummary` | - |
| `RecentTools` | `recent_tools` | `recentTools` | - |
| `RecentToolsDetailed` | `recent_tools_detailed` | `recentToolsDetailed` | - |
| `OmcMode` | `omc_mode` | `omcMode` | - |
| `NotificationPolicy` | `notification_policy` | `notificationPolicy` | - |
| `SessionID` | `session_id` | `sessionID` | - |
| `SessionRef` | `session_ref` | `sessionRef` | - |
| `SessionTitle` | `session_title` | `sessionTitle` | - |
| `SessionIsActive` | `session_is_active` | `sessionIsActive` | - |
| `SessionTTY` | `session_tty` | `sessionTTY` | - |
| `SessionWindowIndex` | `session_window_index` | **없음** | **Go만 있음** |
| `SessionTabIndex` | `session_tab_index` | **없음** | **Go만 있음** |
| `SessionWorkingDirectory` | `session_working_directory` | `sessionWorkingDirectory` | - |
| `SessionActivity` | `session_activity` | `sessionActivity` | - |
| `SessionProcessID` | `session_process_id` | `sessionProcessID` | - |
| `SessionCommand` | `session_command` | `sessionCommand` | - |
| `AvatarVariant` | `avatar_variant` | `avatarVariant` | - |
| `LastAssistantMessage` | `last_assistant_message` | **없음** | **Go만 있음** |
| `SubAgentCount` | `sub_agent_count` | `subAgentCount` | - |
| `SubAgents` | `sub_agents` | **없음** | **Go만 있음** (Swift는 count만) |
| `TeamRole` | `team_role` | `teamRole` | - |
| `TeamTaskTotal` | `team_task_total` | `teamTaskTotal` | - |
| `TeamTaskCompleted` | `team_task_completed` | `teamTaskCompleted` | - |

**불일치 요약** (Go에만 있는 필드 4개):
1. `SessionWindowIndex` - iTerm2 윈도우 인덱스
2. `SessionTabIndex` - iTerm2 탭 인덱스
3. `LastAssistantMessage` - 마지막 어시스턴트 응답 미리보기
4. `SubAgents` - 서브에이전트 상세 배열 (Swift는 `subAgentCount`만)

Swift `Agent.init(from:)` 에서 `decodeIfPresent` 사용으로 디코딩 에러는 발생하지 않으나, UI에서 해당 데이터를 활용할 수 없음.

### 2-2. Event 구조체

**Go Event** (core/agent.go:155-168):

| 필드 | 타입 | 용도 |
|------|------|------|
| `ID` | string | `event-<unixnano>-<seq>` 형식 유니크 ID |
| `AgentID` | string | 소속 에이전트 |
| `Type` | EventType | 14개 유형 (아래 참조) |
| `Summary` | string | 사람이 읽을 수 있는 요약 |
| `OccurredAt` | time.Time | 타임스탬프 |
| `PresentationLabel` | string | UI 레이블 ("Thinking", "Error" 등) |
| `PresentationEmphasis` | string | UI 강조 (info/warning/positive/neutral) |
| `PresentationSummary` | string | UI용 요약 |
| `LifecycleStatus` | string | 이벤트 시점 에이전트 상태 |
| `LifecycleMode` | string | 이벤트 시점 에이전트 모드 |
| `LifecycleReason` | string | 상태 사유 |
| `LifecycleConfidence` | float64 | 상태 신뢰도 |

**EventType 14종**:
| 값 | 의미 |
|----|------|
| `agent.registered` | 새 에이전트 추적 시작 |
| `agent.role_updated` | 역할 변경 |
| `agent.notification_policy_updated` | 알림 정책 변경 |
| `agent.disconnected` | 세션 사라짐 |
| `agent.reconnected` | 세션 재등장 |
| `agent.removed` | 추적 해제 |
| `agent.status_updated` | 상태 전이 |
| `agent.process_started` | 관리형 프로세스 시작 |
| `agent.process_output` | 프로세스 출력/hook 정보 |
| `agent.process_exited` | 프로세스 종료 |
| `agent.layout_changed` | 윈도우/탭 위치 변경 |
| `team.teammate_idle` | 팀원 유휴 |
| `team.task_created` | 팀 태스크 생성 |
| `team.task_completed` | 팀 태스크 완료 |

**Swift AgentEventPayload** (DaemonPayloads.swift): 동일한 12개 필드. 불일치 없음.

### 2-3. DaemonCommand Swift/Go 대조

**Swift DaemonCommand** (19개) vs **Go Command** (52개):

Swift에 있는 19개 커맨드:
- `runManaged`, `attachSession`, `observeSource`, `createTeam`, `addTeamMember`, `listTeams`, `listItermSessions`, `listAgents`, `status`, `events`, `followEvents`, `setNotificationPolicy`, `setRole`, `removeAgent`, `getSettings`, `updateSettings`

Go에만 있는 커맨드 (33개):
- 생명주기: `register.managed`, `managed.exited`, `managed.output`, `managed.stop`
- 에이전트: `agents.open_target`, `agents.rename`
- 세션: `tmux.sessions`
- 데몬: `daemon.shutdown`
- Hook 전체: 29개 (Swift 프론트엔드에서 직접 사용하지 않음)

**프론트엔드에서 필요할 수 있는 누락 커맨드**:
- `agents.rename` - UI에서 이름 변경 기능
- `agents.open_target` - 세션 열기 대상 해석
- `tmux.sessions` - tmux 패널 목록
- `daemon.shutdown` - 데몬 종료

### 2-4. Settings 모델

```
Settings
├── General
│   ├── LaunchAtLogin (bool, 기본 false)
│   ├── CompactMode (bool, 기본 false)
│   └── ShowMenuBarAnimationAlways (bool, 기본 false)
├── Notifications
│   ├── Done (bool, 기본 true)
│   ├── Error (bool, 기본 true)
│   ├── WaitingInput (bool, 기본 true)
│   ├── Silence (bool, 기본 false)
│   ├── QuietHoursEnabled (bool, 기본 false)
│   ├── QuietHoursStartHour (int, 기본 22)
│   ├── QuietHoursEndHour (int, 기본 8)
│   ├── PreviewText (bool, 기본 false)
│   └── HeartbeatMinutes (int, 기본 0, 유효값: 0/10/30/60)
├── Appearance
│   ├── Theme (string, 기본 "auto", 유효: auto/day/night)
│   ├── AnimationSpeedMultiplier (float64, 기본 1.0, 범위 0.25-3.0)
│   ├── ReduceMotion (bool, 기본 false)
│   ├── HamsterSkin (string, 기본 "default")
│   ├── Hat (string, 기본 "none")
│   └── DeskTheme (string, 기본 "classic")
├── Integrations
│   ├── ItermEnabled (bool, 기본 true)
│   ├── TranscriptDirs ([]string, 기본 [])
│   └── ProviderAdapters (map, 기본 {claude:true, generic_process:true, transcript:true})
└── Privacy
    ├── LocalOnlyMode (bool, 기본 true)
    ├── EventHistoryRetentionDays (int, 기본 30)
    └── TranscriptExcerptStorage (bool, 기본 true)
```

Go/Swift 양쪽에서 동일한 스키마. 부분 JSON 업데이트 시 기본값 병합.

---

## 3. 현재 기능 목록과 한계

### 3-1. 현재 동작하는 기능 전수 조사

| # | 기능 | 모드 | 상태 | 한계 |
|---|------|------|------|------|
| 1 | 관리형 에이전트 실행 | Managed | 동작 | PTY 래핑 필요, 직접 터미널 사용 불가 |
| 2 | iTerm2 세션 연결 | Attached | 동작 | AppleScript 의존, macOS 전용 |
| 3 | tmux 패널 연결 | Attached | 동작 | tmux CLI 의존 |
| 4 | 트랜스크립트 관찰 | Observed | 동작 | 키워드 기반 추론, 낮은 신뢰도 (0.2-0.65) |
| 5 | Claude Code hook 통합 | Managed | 동작 | 27개 hook 처리, **핵심 기능** |
| 6 | 에이전트 상태 추적 (13개 상태) | All | 동작 | 신뢰도 모델 기반 |
| 7 | 이벤트 로그 (JSONL, 최대 10K) | All | 동작 | 이벤트 검색/필터링 없음 |
| 8 | 롱폴링 (events.follow) | IPC | 동작 | 200ms 간격, 최대 60초 |
| 9 | 메뉴바 픽셀 오피스 | UI | 동작 | 고정 레이아웃, 스크롤 없음 |
| 10 | 햄스터 스프라이트 애니메이션 | UI | 동작 | 9상태, 4스킨 |
| 11 | macOS 알림 (done/error/waiting/silence/heartbeat) | UI | 동작 | "Open Terminal" 액션 포함 |
| 12 | Attention 우선순위 정렬 | UI | 동작 | error > waitingInput > disconnected |
| 13 | 퀵 메시지 전송 | UI/CLI | 동작 | AppleScript/tmux send-keys/클립보드 폴백 |
| 14 | 설정 관리 (5개 카테고리) | UI/CLI | 동작 | 완전한 CRUD |
| 15 | 팀 관리 (생성/멤버추가/목록) | CLI | 동작 | 기본적인 CRUD만 |
| 16 | OMC 모드 감지 (autopilot/ralph/team 등) | CLI | 동작 | 환경변수/.omx/state/ 파일 기반 |
| 17 | 서브에이전트 추적 | Hook | 동작 | count + 최대 20개 상세 |
| 18 | 팀 태스크 추적 | Hook | 동작 | 스마트 라우팅 (팀 리드 자동 검색) |
| 19 | 세션 자동 등록 | Hook | 동작 | session-start hook 시 자동 에이전트 생성 |
| 20 | launchd 통합 | Daemon | 동작 | 자동 시작, 크래시 재시작 |
| 21 | ham setup 설치 마법사 | CLI | 동작 | hooks + launchd + 데몬 설치 |
| 22 | ham doctor 진단 | CLI | 동작 | 소켓/데몬/hooks 상태 점검 |
| 23 | 데스크 소품 (상태별) | UI | 동작 | thinking=모니터+커피, reading=책 등 |
| 24 | 서브에이전트 미니 햄스터 | UI | 동작 | 최대 6마리 |
| 25 | 팀 역할 뱃지 | UI | 동작 | 왕관(lead)/사람(teammate) |

### 3-2. 각 기능의 한계와 확장 가능성

**IPC 제약**:
- 스트리밍 불가 (request-response 1회성)
- events.follow 롱폴링이 유일한 준실시간 옵션
- NDJSON 스트림 도입 시 IPC 서버 구조 리팩터링 필요

**Hook 제약**:
- 단방향 (Claude → hamd). back-channel 없음
- 토큰/비용 데이터 없음
- 파일 내용/diff 없음 (경로만)
- parent_id 없음 (SubAgents 리스트로 간접 추론)
- 모델 정보 없음

**UI 제약**:
- 메뉴바 패널 크기 제한 (macOS MenuBarExtra)
- 별도 윈도우(Studio)는 미구현
- activationPolicy가 `.accessory` (독 아이콘 없음)

**데이터 제약**:
- 이벤트 검색/필터링 API 없음
- 에이전트별 이벤트 필터링만 가능 (AfterEventID 기반 페이지네이션)
- 집계/통계 기능 없음

### 3-3. 미사용/미완성 코드

- `HamPersistence/InMemoryAgentStore`: 실제 디스크 영속화 아님. 메뉴바 앱에서는 항상 데몬에서 fetch
- `HamRuntime/RuntimeRegistry`: CLI용 로컬 레지스트리. 메뉴바 앱에서는 미사용
- `HamInference/StatusInferenceEngine`: Observed 에이전트에서만 사용. 단순한 키워드 매칭으로 신뢰도가 낮음

---

## 4. 외부 의존성 및 연동 분석

### 4-1. Claude Code 최신 스펙 (2026년 4월 기준)

> 출처: [code.claude.com/docs](https://code.claude.com/docs)

#### Hook 시스템

**26개 공식 이벤트 타입** (ham-agents는 27개 처리 - 1개 추가):

| # | 이벤트 | 블로킹 | 매처 지원 |
|---|--------|--------|-----------|
| 1 | `SessionStart` | No | startup, resume, clear, compact |
| 2 | `InstructionsLoaded` | No | session_start, path_glob_match 등 |
| 3 | `UserPromptSubmit` | Yes (exit 2) | - |
| 4 | `PreToolUse` | Yes (exit 2) | 도구명 |
| 5 | `PermissionRequest` | Yes (exit 2) | 도구명 |
| 6 | `PermissionDenied` | No | 도구명 |
| 7 | `PostToolUse` | No | 도구명 |
| 8 | `PostToolUseFailure` | No | 도구명 |
| 9 | `Notification` | No | permission_prompt, idle_prompt 등 |
| 10 | `SubagentStart` | No | 에이전트 타입 |
| 11 | `SubagentStop` | Yes (exit 2) | 에이전트 타입 |
| 12 | `TaskCreated` | Yes (exit 2) | - |
| 13 | `TaskCompleted` | Yes (exit 2) | - |
| 14 | `Stop` | Yes (exit 2) | - |
| 15 | `StopFailure` | No | rate_limit 등 |
| 16 | `TeammateIdle` | Yes (exit 2) | - |
| 17 | `ConfigChange` | Yes (exit 2) | user_settings 등 |
| 18 | `CwdChanged` | No | - |
| 19 | `FileChanged` | No | 파일명 |
| 20 | `WorktreeCreate` | Yes | - |
| 21 | `WorktreeRemove` | No | - |
| 22 | `PreCompact` | No | manual, auto |
| 23 | `PostCompact` | No | manual, auto |
| 24 | `Elicitation` | Yes | MCP 서버명 |
| 25 | `ElicitationResult` | Yes | MCP 서버명 |
| 26 | `SessionEnd` | No | - |

**4가지 핸들러 타입**: command, http, prompt, agent

**핵심 제약**:
- Hook 출력 10,000자 제한
- 타임아웃: Command 600초, Prompt 30초, Agent 60초, HTTP 30초
- `defer` (PreToolUse)는 headless `-p` 모드에서만 동작

#### Agent Teams

> 출처: [code.claude.com/docs/en/agent-teams](https://code.claude.com/docs/en/agent-teams)

- **실험적 기능**: `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` 필요 (v2.1.32+)
- **구조**: Team Lead (1) + Teammates (2-16). 독립 Claude Code 인스턴스
- **통신**: 공유 태스크 리스트 + 메일박스. 파일 락 기반 태스크 클레임
- **표시 모드**: in-process (터미널 내) 또는 split panes (tmux/iTerm2)
- **`.claude/agents/` 정의**: tools allowlist + model 적용됨. skills/mcpServers는 미적용
- **Hook 통합**: TeammateIdle, TaskCreated, TaskCompleted
- **제약**: 세션 복원 불가, 중첩 팀 불가, 리드 고정, 권한 상속

#### Channels / Plugins

> 출처: [code.claude.com/docs/en/channels](https://code.claude.com/docs/en/channels), [code.claude.com/docs/en/plugins](https://code.claude.com/docs/en/plugins)

- **연구 프리뷰**: v2.1.80+ 필요, claude.ai 로그인 필수
- **지원 플랫폼**: Telegram, Discord, iMessage (macOS)
- **양방향**: Claude가 이벤트를 읽고 같은 채널로 응답
- **보안**: 발신자 허용목록, 페어링 코드 플로우
- **플러그인 구조**: commands/, agents/, skills/, hooks/, .mcp.json, .lsp.json

#### Skills

> 출처: [code.claude.com/docs/en/skills](https://code.claude.com/docs/en/skills)

- **Agent Skills 오픈 표준** ([agentskills.io](https://agentskills.io)) - Claude Code, Cursor, Gemini CLI 공통
- **정의**: `SKILL.md` + YAML frontmatter
- **주요 필드**: name, description, allowed-tools, model, effort, context (fork), agent, hooks, paths
- **위치**: enterprise > personal (~/.claude/skills/) > project (.claude/skills/) > plugin
- **내장 스킬**: /batch, /claude-api, /debug, /loop, /simplify
- **동적 컨텍스트**: `` !`command` `` 문법으로 쉘 명령 실행

#### Scheduled Tasks

> 출처: [code.claude.com/docs/en/scheduled-tasks](https://code.claude.com/docs/en/scheduled-tasks)

| 기능 | Cloud | Desktop | /loop (CLI) |
|------|-------|---------|-------------|
| 실행 위치 | Anthropic 클라우드 | 로컬 머신 | 로컬 머신 |
| 머신 필요 | No | Yes | Yes |
| 세션 필요 | No | No | Yes |
| 재시작 유지 | Yes | Yes | No |
| 최소 간격 | 1시간 | 1분 | 1분 |

- 도구: CronCreate, CronList, CronDelete (5필드 cron 표현식)
- 세션당 최대 50개
- 반복 태스크 7일 후 자동 만료

#### Remote Control

> 출처: [code.claude.com/docs/en/remote-control](https://code.claude.com/docs/en/remote-control)

- v2.1.51+ 필요, 모든 플랜 사용 가능
- **Server 모드**: `claude remote-control --spawn worktree --capacity 32`
- **연결**: 세션 URL, QR 코드, claude.ai/code, 모바일 앱
- **Headless 모드**: `claude -p "prompt" --output-format stream-json --bare`
- **제약**: 인바운드 포트 없음 (HTTPS 아웃바운드만), API 키 불가 (OAuth 필수)

#### Desktop / VS Code 표면

| 기능 | CLI | VS Code | Desktop |
|------|-----|---------|---------|
| Computer Use | No | No | Yes (macOS/Windows) |
| Live Preview | No | No | Yes |
| PR 모니터링 | Manual | Manual | Yes (auto-fix, auto-merge) |
| Dispatch (모바일) | No | No | Yes |
| Connectors | No | No | Yes (GitHub, Slack, Linear) |
| Cloud Sessions | No | No | Yes |
| 스케줄 태스크 | /loop (세션) | CLI | 영구 (재시작 유지) |

### 4-2. 경쟁 제품 최신 기능

> 출처: 각 경쟁사 공식 사이트 및 changelog (2026년 4월 기준)

#### Cursor (cursor.com)

| 기능 | 설명 |
|------|------|
| **Cloud Agents** | 클라우드 VM에서 자율 실행, 최대 8개 병렬/사용자, 50/팀. 자체호스팅 옵션 |
| **Composer 2** | 서브에이전트로 코드베이스 탐색, Plan Mode, Debug Mode |
| **Automations** | Slack/GitHub/Linear/PagerDuty/webhook/cron 트리거. 이전 실행 결과 학습 |
| **Bugbot + Autofix** | PR 자동 리뷰 + 수정 제안. 35% 이상 자동수정 머지율 |
| **Rules** | Team/Project/User/Bugbot 레벨 규칙 |
| **CLI** | 터미널에서 전체 에이전트 기능 |
| **토큰 대시보드** | 크레딧 기반 과금, 사용량 추적 |
| **가격** | Pro $20/mo, Business $40/seat/mo |

> 출처: [cursor.com/product](https://cursor.com/product), [cursor.com/changelog](https://cursor.com/changelog), [cursor.com/bugbot](https://cursor.com/bugbot)

#### Windsurf (windsurf.com)

| 기능 | 설명 |
|------|------|
| **Cascade** | 실시간 개발자 액션 추적 기반 AI 플로우 엔진 |
| **Checkpoints** | 대화에서 생성한 프로젝트 스냅샷, 복원 가능 |
| **Workflows** | 반복 작업용 자동 생성 슬래시 커맨드 |
| **AGENTS.md** | 프로젝트 루트 마크다운 에이전트 지시. 크로스툴 표준 |
| **Memories** | 자동 + 수동 메모리. 워크스페이스별 로컬 저장. 크레딧 미소모 |
| **Rules** | .windsurfrules + 프로젝트/시스템 레벨 |
| **Hooks** | 모델 응답/사용자 프롬프트 기반 (감사/로깅) |
| **가격** | Free 25 credit, Pro $15-20/mo, Teams $30/seat/mo |

> 출처: [windsurf.com/cascade](https://windsurf.com/cascade), [docs.windsurf.com](https://docs.windsurf.com), [windsurf.com/pricing](https://windsurf.com/pricing)

#### Warp (warp.dev)

| 기능 | 설명 |
|------|------|
| **Local Agents** | 터미널 내장, PTY 세션 직접 접근. 로컬 코드베이스 인덱싱 |
| **Oz Cloud Agents** | 무제한 병렬 클라우드 에이전트. 멀티레포 지원 |
| **Computer Use** | 클라우드 샌드박스 GUI 조작 |
| **Agent Management Panel** | 로컬+클라우드 에이전트 통합 대시보드. 실시간 모니터링 |
| **Oz 플랫폼** | CLI + SDK + API. 멀티모델 (Claude/Codex/Gemini). 자체호스팅 |
| **가격** | Free (터미널만), Build $20/mo, Business $50/seat/mo |

> 출처: [warp.dev/agents](https://www.warp.dev/agents), [warp.dev/oz](https://www.warp.dev/oz), [docs.warp.dev](https://docs.warp.dev)

#### 비교 매트릭스

| 기능 | Cursor | Windsurf | Warp | ham-agents |
|------|--------|----------|------|------------|
| 유형 | AI IDE (VS Code fork) | AI IDE (custom) | AI Terminal + Cloud | 메뉴바 에이전트 관측 |
| 클라우드 에이전트 | Yes (8/user) | No | Yes (무제한) | No |
| 자동화/트리거 | Yes (다양한 소스) | No (로깅만) | Yes (cron/webhook/API) | No |
| PR 리뷰봇 | Bugbot + Autofix | GitHub 연동 | 없음 | 없음 |
| 터미널 제어 | 샌드박스 쉘 | 통합 터미널 | Full PTY 접근 (**유일**) | 외부 관찰만 |
| 에이전트 관리 UI | 사이드바 | N/A | 통합 대시보드 | 메뉴바 픽셀 오피스 |
| 컨텍스트 엔진 | 코드베이스 인덱싱 | 실시간 액션 추적 | 로컬 임베딩 | Hook 이벤트 |
| 비용 추적 | 토큰 대시보드 | 크레딧 표시 | 크레딧 표시 | **없음** |
| **ham 차별화** | | | | 터미널 무관 Claude Code 전용 mission control |

### 4-3. 커뮤니티 사례 및 오픈소스

> 출처: GitHub, HackerNews, 블로그 등 (2026년 4월 기준)

#### 멀티세션 관리 도구

| 도구 | Stars | 언어 | 접근 방식 | 라이선스 |
|------|-------|------|-----------|----------|
| **cmux** (manaflow-ai) | 12.7K | Swift | 네이티브 macOS 터미널 (libghostty) | - |
| **claude-squad** | 6.8K | Go | TUI + tmux + worktree | AGPL-3.0 |
| **AgentOps.ai** | 5.4K | Python | 에이전트 관측 플랫폼 (SaaS) | MIT |
| **CCManager** | 984 | TypeScript | CLI + worktree + devcontainer | MIT |
| **cmux** (craigsc) | 484 | Shell | 순수 bash worktree 래퍼 | - |
| **AMUX** (mixpeek) | - | Python | tmux + 웹 대시보드 + 칸반 | MIT+CC |

> 출처: [github.com/manaflow-ai/cmux](https://github.com/manaflow-ai/cmux), [github.com/smtg-ai/claude-squad](https://github.com/smtg-ai/claude-squad), [github.com/AgentOps-AI/agentops](https://github.com/AgentOps-AI/agentops), [github.com/kbwo/ccmanager](https://github.com/kbwo/ccmanager), [github.com/mixpeek/amux](https://github.com/mixpeek/amux)

#### 관측 도구

| 도구 | Stars | 라이선스 | 특징 |
|------|-------|----------|------|
| **Langfuse** | 24.4K | MIT | OpenTelemetry 네이티브, 셀프호스팅 |
| **AgentOps.ai** | 5.4K | MIT | 에이전트 전문, 400+ LLM 통합 |
| **LangSmith** | N/A | Proprietary | LangChain 생태계, Polly AI 어시스턴트 |

> 출처: [langfuse.com](https://langfuse.com), [agentops.ai](https://www.agentops.ai), [langchain.com/langsmith](https://www.langchain.com/langsmith)

#### 핵심 커뮤니티 패턴

1. **Git worktree = 격리 primitive**: 거의 모든 도구가 에이전트별 worktree 사용
2. **tmux = 세션 기판**: 멀티 터미널 관리의 de facto 표준
3. **3-에이전트 리서치 패턴**: 병렬 연구 에이전트 3개 (imports/dependency/test) → 3분 vs 순차 10분
4. **도메인 기반 라우팅**: frontend/backend/database 작업을 병렬 에이전트로 분배

> 출처: [Towards Data Science](https://towardsdatascience.com/how-to-run-coding-agents-in-parallell/), [DEV.to](https://dev.to/bredmond1019/multi-agent-orchestration-running-10-claude-instances-in-parallel-part-3-29da), [Anthropic Engineering](https://www.anthropic.com/engineering/building-c-compiler)

#### 참고 사례: Anthropic C 컴파일러

16개 Agent Teams 에이전트가 100K줄 Rust C 컴파일러 작성 (Linux 커널 빌드 가능). ~2,000 세션, ~$20K API 비용.
> 출처: [anthropic.com/engineering/building-c-compiler](https://www.anthropic.com/engineering/building-c-compiler)

### 4-4. 기존 어댑터 현황

#### iTerm2 AppleScript API

현재 지원 범위 (`go/internal/adapters/iterm2.go`):
- 전체 윈도우 > 탭 > 세션 열거
- 세션별: ID, title, TTY, active 여부, window/tab index
- ps/lsof로 보강: foreground PID, working directory, command
- 포커스: `do script "..." in session <id>` (스텁 상태)
- 텍스트 쓰기: `write text` AppleScript

#### tmux CLI

현재 지원 범위 (`go/internal/adapters/tmux.go`):
- `tmux list-sessions/windows/panes` 로 전체 열거
- 패널별: session, window, pane index, active 여부, title, PID
- `tmux send-keys` 로 키 전송
- 현재 패널 감지 (`$TMUX` 환경변수)

---

## 완료 체크리스트

- [x] 모든 Go 파일의 역할 문서화 (48개 비테스트 파일)
- [x] 모든 Swift 파일의 역할 문서화 (7개 모듈)
- [x] IPC 커맨드 전수 목록 (52개) + 각 커맨드의 request/response 스키마
- [x] Hook 전수 목록 (27개) + 각 hook의 페이로드와 상태 전이
- [x] Go-Swift 모델 불일치 목록 (4개 필드)
- [x] 현재 기능 목록 (25개) + 한계점
- [x] Claude Code 최신 스펙 반영 (26 이벤트, 4 핸들러, teams, channels, skills, schedules, remote)
- [x] 경쟁 제품 최신 기능 반영 (Cursor, Windsurf, Warp)
- [x] 커뮤니티 사례 및 오픈소스 도구 조사
- [x] 검색 출처 URL 명시
