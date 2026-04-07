# ham Studio 기능 명세서

> Phase 2 "Terminal IDE" | 2026.04 | ham-agents v2.0+

---

## 0. Overview — ham Studio is the Primary UX

Phase 2 에서 ham Studio 는 ham-agents 의 **primary UX** 가 된다. 사용자는 ham Studio 윈도우를 열고, 새 탭을 만들어서 Claude Code 세션을 **Studio 탭 안에서 직접** 실행한다. 더 이상 별도 터미널 앱(iTerm/tmux) 을 먼저 열 필요가 없다.

**핵심 전환**:
- Before: 메뉴바 ambient + 기존 터미널 세션을 관찰
- After: Studio 윈도우 primary + 탭 안에 내장 PTY 가 Claude Code 를 돌림
- 메뉴바 MenuBarExtra 는 ambient 알림 surface 로 유지

Phase 1 Mission Control (관찰) 위에 Phase 2 ham Studio (Direct + Govern) 가 올라간다. 기존 attached / observed 모드는 legacy fallback 으로 유지되어 이미 터미널에서 돌고 있는 세션도 흡수한다.

---

## 목차

1. [개요](#1-개요)
2. [윈도우 아키텍처](#2-윈도우-아키텍처)
3. [3-패널 레이아웃](#3-3-패널-레이아웃)
4. [데이터 흐름](#4-데이터-흐름)
5. [기능별 상세 명세](#5-기능별-상세-명세)
   - [5.1 ham Studio 윈도우](#51-ham-studio-윈도우)
   - [5.2 Agent Team Orchestrator](#52-agent-team-orchestrator)
   - [5.3 Playbooks / Recipes](#53-playbooks--recipes)
   - [5.4 Git/CI/Issue 연동](#54-gitciissue-연동)
   - [5.5 Review Loop](#55-review-loop)
   - [5.6 Approval Inbox 업그레이드](#56-approval-inbox-업그레이드)
6. [Graceful Degradation 전략](#6-graceful-degradation-전략)
7. [경쟁 제품 참조](#7-경쟁-제품-참조)

---

## 1. 개요

ham Studio는 ham-agents의 Phase 2 "Terminal IDE"로, 현재 메뉴바 앱이 제공하는 "glance(한눈에 보기)" 기능을 넘어 "control(조작)" 기능을 제공하는 전용 윈도우 애플리케이션이다.

**핵심 원칙:**
- 메뉴바 = "상태 확인" (ambient UI, 항상 떠 있음)
- Studio = "조작" (필요할 때 열고, 적극적으로 개입)
- ham은 새로운 에디터가 아니라 **에이전트 조종석**이다. 편집은 VS Code/Cursor에 맡기고, 운영 시야를 제공한다.

**Phase 1 선행 의존성:**
- P1-0: 신뢰성 기반 (버그 수정, IPC enum 동기화)
- P1-1: 이벤트 스키마 확장 + Artifact Capture
- P1-2: 실시간 Session Graph
- P1-3: Notification Inbox (읽기 전용)
- P1-4: 비용/토큰 텔레메트리 v1
- P1-5: 이벤트 브로드캐스트 기반

---

## 왜 ham-agents 를 써야 하는가

### 시나리오 1: 멀티세션 조종

사용자가 ham Studio 에서 탭 3 개를 열어 세 저장소에서 Claude Code 를 돌린다. 한 탭에서 "잠깐, 이 변경 말고 다른 방향으로 가봐" 라고 직접 타이핑해서 에이전트에 지시한다. 다른 탭은 hook.notification 이 울려서 ambient 배지로 주의를 끈다.

### 시나리오 2: 승인 워크플로 (Direct)

Claude Code 가 `rm -rf build/` 를 시도. hook.permission-request 가 발생하는 순간 hamd 는 해당 탭의 PTY 를 블록 상태로 잡고, Studio UI 에 approve/deny 모달을 띄운다. 사용자가 Deny 를 누르면 hamd 는 Claude Code 의 permission 응답으로 차단을 리턴하고, PTY 는 다음 입력을 기다린다.

### 시나리오 3: 팀 오케스트레이션 (Govern)

Agent Team Orchestrator 탭에서 lead 에이전트와 worker 에이전트 3 개를 구성. Policy Engine 이 "worker 는 prod DB 접근 금지" 룰을 적용. worker 가 prod 접근을 시도하면 Phase 2 PTY 층에서 차단되고 lead 에게 알림이 간다.

---

## P2-1. Embedded PTY Runtime

### 목표
ham Studio 탭 안에서 Claude Code 프로세스가 직접 돌아가게 만든다. hamd 는 PTY master 를 소유하고, Studio 는 PTY 데이터를 IPC 스트림으로 받아 SwiftTerm 에 렌더한다.

### 기술 스택
- **Swift 터미널 에뮬레이터**: SwiftTerm (https://github.com/migueldeicaza/SwiftTerm) — xterm-256color 호환, AppKit 네이티브, 오픈소스 (MIT)
- **Transport**: IPC NDJSON 스트림 — 자세한 설계는 `tech-migration.md` ADR-2 참조
- **PTY allocation**: hamd 의 ManagedService 에 기존 `go/cmd/ham/pty.go` 의 openPTY 패턴을 이식 (`/dev/ptmx` + `TIOCPTYGRANT` + `Setsid/Setctty`)

### Go 측 변경
- `go/internal/runtime/managed.go` ManagedService.Start: 기존 StdoutPipe/StderrPipe 경로 → PTY master 할당으로 교체 (새 sub-case, 기존 plain-pipe 경로는 fallback 옵션으로 유지)
- `go/internal/runtime/managed.go` managedProcess 구조체에 `ptmx *os.File`, `subs []chan []byte`, `subsMu sync.Mutex` 추가
- `go/internal/ipc/ipc.go` Command 상수에 `CommandFollowPTY = "pty.follow"`, `CommandWritePTY = "pty.write"`, `CommandResizePTY = "pty.resize"` 추가 (기존 52 개 → 55 개)
- `go/internal/ipc/server.go` dispatch 에 PTY case 3 개 추가. `handleFollowPTY` 는 CommandFollowEvents 를 모델로 long-poll 방식
- `go/internal/core/agent.go` Agent 구조체: 기존 `SessionTTY string` 을 managed 모드에서도 활용하도록 `RecordManagedStarted` 에서 ptmx 경로 설정. 신규 `PtySubscribers int` (관찰용) 필드 선택

### Swift 측 변경
- 신규 모듈 `Sources/HamApp/PTY/` 에 SwiftTerm 통합. 하위 파일:
  - `PtyClient.swift` — `followPTY(agentID:) -> AsyncStream<PtyFrame>`, `writePTY(agentID:data:)`, `resizePTY(agentID:cols:rows:)`
  - `PtyTabView.swift` — SwiftUI + SwiftTerm 호스트. SwiftTerm 은 AppKit 기반이라 `NSViewRepresentable` 로 래핑
  - `PtyFrameDecoder.swift` — NDJSON base64 디코딩
- `Sources/HamCore/DaemonIPC.swift` `DaemonCommand` enum 에 `ptyFollow`, `ptyWrite`, `ptyResize` 3 개 케이스 추가 (기존 16 → 19)
- `Sources/HamApp/StudioWindow.swift` (신규 또는 확장): 탭 컨테이너, 각 탭은 PtyTabView 인스턴스

### IPC 변경
ADR-2 Option 1 (NDJSON stream upgrade) 적용:

- `CommandFollowPTY` — 장기 연결. `{agent_id}` 를 받아 base64 인코딩 PTY 프레임을 NDJSON 으로 스트림
- `CommandWritePTY` — 일회성. `{agent_id, data: base64}` 를 받아 ptmx 에 write
- `CommandResizePTY` — 일회성. `{agent_id, cols, rows}` 를 받아 ptmx 에 TIOCSWINSZ

프레임 포맷:
```json
{"type":"pty_data","agent_id":"a-1","seq":1,"data":"..."}
{"type":"pty_exit","agent_id":"a-1","exit_code":0}
```

### 의존성
- ADR-2 (tech-migration.md) — transport 결정
- ADR-1 (mission-control.md) — PTY 바이트는 별도로, SessionEvent 는 라인 tee 로 병행 기록
- 기존 `go/cmd/ham/pty.go` openPTY 코드 재사용 (복사 or 공용 패키지로 추출)

---

## P2-2. Session Launcher

Studio 헤더의 "New Session" 버튼으로 새 탭 생성.

### 플로우
1. 사용자 클릭 → Launcher 시트 표시
2. 입력: workspace (디렉토리 피커), 모델 (`claude` / `claude-3-opus` 등), playbook (옵션), skill preset (옵션)
3. Studio 가 `CommandRegisterManaged` 로 Agent 등록 + `CommandRunManaged` 로 hamd 스폰 요청
4. hamd 는 PTY 할당 후 Claude Code 프로세스 start. PID/ptmx 저장
5. Studio 는 `CommandFollowPTY` 구독 시작 → 탭 렌더링

### 재시작 / resume
- Studio 크래시 후 재시작 시, 열려있던 탭의 `agent_id` 를 저장소에서 읽어 `CommandFollowPTY(agent_id, resume_from_seq: N)` 재구독
- hamd 쪽은 각 subscriber 에 대해 seq 기반 ring buffer (최근 N 프레임) 유지

---

## P2-3. Approval Interception (Govern 축의 핵심)

> **Spike Validated (2026-04-08)**: Phase 2 Step 0b spike confirmed that hook.permission-request → hamd → Claude Code 전송 체인이 동기 블로킹이라는 것을 정적 분석 + 동적 실험으로 검증함. P2-3 은 기존 설계대로 진행 가능. 필요 작업은 (1) `Response.PermissionDecision` 필드 추가, (2) 새 IPC 커맨드 `decision.permission` 추가, (3) `CommandHookPermissionReq` 핸들러에 wait primitive, (4) `runHook` 이 decision 을 `hookSpecificOutput` JSON 으로 stdout 에 emit. 상세 근거는 `docs/spec/tech-migration.md` ADR-2 Spike Results (2026-04-08) 참조.

### 문제
Phase 1 에서는 `hook.permission-request` 가 발생하면 Inbox 에 읽기 전용으로 기록되고, 실제 Claude Code 는 `ham ask` 시스템을 거치지 않고 자체 결정을 내렸다. 즉 ham-agents 는 "알림만" 줬다.

### Phase 2 해결
PTY 층이 존재하므로 hamd 가 permission 요청을 가로챌 수 있다.

**플로우**:
1. Claude Code 가 destructive tool 호출 전 `hook.permission-request` 를 발사
2. hamd 는 hook 수신 즉시 해당 세션의 PTY write 를 일시 정지 (subs 채널 drain 하지 않음)
3. hamd 는 Swift Studio 에 `hook.permission-request` 이벤트를 push (P1-3 Inbox + 신규 approval modal)
4. Studio UI 는 탭 오버레이로 approve/deny 모달 표시
5. 사용자 결정 → Studio 가 `CommandAnswerPermission(agent_id, request_id, approved: bool)` 호출
6. hamd 는 Claude Code 에 permission 응답을 stdin 으로 주입 (PTY write 재개 포함)
7. 사용자가 일정 시간 응답하지 않으면 `policy.default_deny_after: 30s` 규칙에 따라 자동 처리

### 제약
- **Phase 2 P2-1 spike 필수**: 현재 `go/cmd/ham/commands.go` 의 hook.permission-request 는 fire-and-forget 이다. 실제로 Claude Code 가 tool 실행 전 hook 응답을 기다리는지, 또는 PTY 일시 정지 + stdin 으로 permission 응답 주입이 동작하는지를 fixture 스크립트로 검증해야 한다. 검증 실패 시 P2-3 은 alert-only 모드로 강등되고 P3-2 Alert Policy Engine 의 Realism Check tool blocking 행은 다시 ✗ (또는 "managed-mode best-effort") 로 되돌린다. 따라서 P2-3 구현 커밋은 spike PASS 를 전제로 한다.
- Policy Engine (Phase 3 P3-2) 과 연동 시 "자동 승인/거부" 룰 적용. Phase 2 에서는 수동 승인이 기본

### 의존성
- Phase 1 P1-3 Notification Inbox (읽기 전용 버전) 이 먼저 존재해야 함
- Phase 3 P3-2 Policy Engine 이 이 approval 경로를 extension point 로 사용

---

## P2-X. Legacy Input Modes (attached / observed)

기존 attached / observed 모드는 PTY 내장 primary 전환 후에도 유지된다:

- **attached mode**: 이미 열려있는 iTerm / tmux 세션에 ham 이 AppleScript / tmux control-mode 로 붙어서 관찰 + `send-keys` 로 메시지 송신. 내장 PTY 가 필요 없는 사용자 (이미 vim/neovim 등 자기 환경에서 Claude Code 돌리는 사람) 흡수
- **observed mode**: `~/.claude/projects/*/sessions/*/transcript` 파일을 감시만. 완전 read-only. 스크린스크레이핑 용
- **Studio 에서의 표시**: 새 탭 타입 "외부 세션" 으로 표시. PTY 렌더 대신 이벤트 타임라인 + inbox 뷰만 렌더
- **차이점**: Direct 축은 제한적 (attached 는 send-keys 로 가능, observed 는 불가). Govern 축은 hook 기반 알림만 (PTY 차단은 불가)
- **제거 금지**: 라운드 3 에서도 legacy 모드는 유지한다

---

## 2. 윈도우 아키텍처

### 현재 구조

```
HamMenuBarApp (@main App)
├── MenuBarExtra (.window style)  ← 메뉴바 팝업
│   └── MenuBarContentView
└── HamOfficeWindowPresenter (싱글톤)  ← NSWindow로 별도 창
    └── MenuBarContentView (동일 뷰 재사용)
```

- `AppDelegate.applicationDidFinishLaunching`에서 `NSApp.setActivationPolicy(.accessory)` → Dock 아이콘 없음
- `HamOfficeWindowPresenter.show()`에서 `NSApp.activate(ignoringOtherApps: true)` → 일시적 활성화

### Studio 도입 후 구조

```
HamMenuBarApp (@main App)
├── MenuBarExtra (.window style)        ← 기존 메뉴바 (변경 없음)
│   └── MenuBarContentView
├── HamOfficeWindowPresenter            ← 기존 Office 창 (유지)
│   └── MenuBarContentView
└── HamStudioWindowPresenter (신규 싱글톤)  ← Studio 전용 창
    └── HamStudioRootView (신규)
```

### NSWindow 생명주기

```swift
// HamStudioWindowPresenter — HamOfficeWindowPresenter 패턴을 따름
@MainActor
final class HamStudioWindowPresenter {
    static let shared = HamStudioWindowPresenter()
    private var window: NSWindow?

    func show(viewModel: StudioViewModel) {
        if let window {
            window.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }

        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1200, height: 800),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "ham Studio"
        window.minSize = NSSize(width: 900, height: 600)
        window.isReleasedWhenClosed = false
        window.delegate = self  // windowWillClose에서 정리
        // ... contentViewController 설정
        self.window = window
    }
}
```

### Activation Policy 전환

```
상태 1: 메뉴바만 사용 중
  → activationPolicy = .accessory (Dock 아이콘 없음)

상태 2: Studio 창 열림
  → activationPolicy = .regular (Dock 아이콘 표시, Cmd+Tab 진입 가능)

상태 3: Studio 창 닫힘
  → activationPolicy = .accessory (다시 숨김)
```

**전환 시점:**
- `HamStudioWindowPresenter.show()` → `.regular` 전환
- `NSWindowDelegate.windowWillClose()` → Office 창도 없으면 `.accessory` 복귀
- 주의: `.regular` ↔ `.accessory` 전환 시 메뉴바 MenuBarExtra가 사라지지 않도록 검증 필요 (macOS 14+ 동작 확인)

### NSWindow Activation Policy (재정의)

Phase 1 에서는 `.accessory` (메뉴바 전용) 가 default 였다. Phase 2 ham Studio 도입과 함께 default 전환:

- **Primary**: `.regular` — ham Studio 윈도우가 기본. Dock 아이콘 표시, cmd+tab 대상
- **Secondary**: MenuBarExtra 는 계속 유지하되 "ambient 알림 + 빠른 요약" 용도로 강등
- **Transition**: Studio 윈도우를 모두 닫으면 `.accessory` 모드로 fallback (Phase 1 식 동작). 이 전환은 `NSApplication.setActivationPolicy` 로 구현
- **검증 필요**: `.accessory ↔ .regular` 전환 시 MenuBarExtra 가 사라지지 않는지 macOS 14/15 에서 실측. Phase 2 P2-1 구현 초기에 spike 필요

### Session Lifecycle (per-tab)

- **탭당 1 프로세스**: 각 Studio 탭은 하나의 managed Claude Code 프로세스에 대응. 탭 닫기 = `CommandStopManaged` + SIGTERM/SIGKILL (기존 ManagedService.Stop 경로 재사용)
- **최대 동시 탭**: 권장 상한 6 (Claude Code 프로세스당 대략 500MB~1GB 메모리 소모 가정). 이를 초과하면 Launcher 가 경고. Hard limit 아님
- **크래시 복구**: Claude Code 프로세스 비정상 종료 시 hamd 가 `hook.agent-finished` 이벤트 발사, Studio 탭은 "세션 종료됨" 뷰 + "재시작" 버튼 표시
- **idle 자동 정리**: 장시간 idle (기본 2 시간) 인 탭은 사용자에게 정리 제안. 자동 삭제 아님

### 창 간 관계

| 창 | 용도 | 최소 크기 | 열기 방식 |
|---|---|---|---|
| MenuBarExtra popover | 빠른 상태 확인 | 380x400 | 메뉴바 아이콘 클릭 |
| Ham Office (기존) | 상세 보기 (detach) | 420x520 | 메뉴바 "Open in Window" |
| ham Studio | 전체 조작 | 900x600 | 메뉴바 "Open Studio" / Dock 아이콘 / 단축키 |

---

## 3. 3-패널 레이아웃

```
+------------------+----------------------------+--------------------+
|                  |                            |                    |
|  Left Panel      |  Center Panel              |  Right Panel       |
|  (에이전트 트리)  |  (라이브 터미널/로그/출력)   |  (인스펙터)        |
|                  |                            |                    |
|  - 에이전트 목록  |  - 터미널 에뮬레이션        |  - Diff 뷰         |
|  - 팀 그룹핑     |  - 이벤트 로그 스트림       |  - Approval 목록    |
|  - 상태 뱃지     |  - 출력 하이라이트          |  - 비용/토큰        |
|  - 빠른 필터     |  - 탭 (에이전트별)          |  - Context 사용량   |
|                  |                            |  - Checkpoints      |
|  너비: 240-320   |  너비: flex (나머지)        |  너비: 280-400      |
|                  |                            |                    |
+------------------+----------------------------+--------------------+
|                        Status Bar                                  |
|  [전체 에이전트 수] [실행 중] [대기] [에러] [총 비용]  [연결 상태] |
+--------------------------------------------------------------------+
```

### 패널 크기 제약

| 패널 | 최소 너비 | 기본 너비 | 최대 너비 | 축소 가능 |
|---|---|---|---|---|
| Left | 200px | 260px | 400px | sidebar toggle |
| Center | 400px | flex | - | 불가 |
| Right | 240px | 320px | 500px | inspector toggle |

### SwiftUI 구조

```swift
// HamStudioRootView
struct HamStudioRootView: View {
    @StateObject var viewModel: StudioViewModel

    var body: some View {
        NavigationSplitView {
            StudioSidebarView(viewModel: viewModel)         // Left
        } detail: {
            HSplitView {
                StudioCenterView(viewModel: viewModel)      // Center
                StudioInspectorView(viewModel: viewModel)   // Right
            }
        }
        .toolbar { StudioToolbar(viewModel: viewModel) }
    }
}
```

**대안 검토:** `NavigationSplitView`는 3-column을 지원하지만 center+right 비율 제어가 제한적. `HSplitView` (AppKit)를 NSHostingController 안에서 사용하는 방안도 고려. 현재 HamOfficeWindowPresenter가 NSHostingController 패턴을 이미 사용하므로 이 경로가 호환성 측면에서 안전하다.

---

## 4. 데이터 흐름

### 현재 구조 (메뉴바)

```
hamd (Go daemon)
  ↓ Unix socket (request-response)
UnixSocketDaemonTransport
  ↓
HamDaemonClient (11 methods)
  ↓
MenuBarSummaryService → HamMenuBarSummary
  ↓
MenuBarViewModel (5초 polling + 15초 event follow)
  ↓
MenuBarContentView / PixelOfficeView
```

### Studio 구조 (신규)

```
hamd (Go daemon)
  ↓ Unix socket (동일 transport 재사용)
UnixSocketDaemonTransport
  ↓
HamDaemonClient (기존 11 methods + 신규 methods)
  ↓
StudioViewModel (1초 polling + 200ms event follow)
  ↓ @Published properties
  ├── StudioSidebarView (Left)
  ├── StudioCenterView (Center)
  └── StudioInspectorView (Right)
```

### 핵심 설계 결정

**1. ViewModel 분리:**
- `MenuBarViewModel`(기존)과 `StudioViewModel`(신규)은 별도 인스턴스
- 동일한 `HamDaemonClient`를 공유하되, polling 주기가 다름
- Studio가 열려 있을 때 MenuBarViewModel은 polling 주기를 30초로 늦춤 (리소스 절약)

**2. Event Follow 강화:**
- 현재: `waitMilliseconds: 15000` (15초 long-poll)
- Studio: `waitMilliseconds: 200` (200ms, 60초 max) → 거의 실시간
- Phase 1의 P1-5 EventBus가 완료되면 subscription 방식으로 전환

**3. Transport 공유:**
- `UnixSocketDaemonTransport`는 `Sendable`이므로 ViewModel 간 공유 가능
- 각 request는 독립 소켓 연결 (connect-per-request) → 동시 요청 안전

---

## 5. 기능별 상세 명세

---

### 5.1 ham Studio 윈도우

#### 기능 설명 + 사용자 시나리오

3-패널 구성의 전용 윈도우. 개발자가 모노레포에서 5개의 Claude Code 세션을 병렬로 돌리고 있을 때, 메뉴바에서는 "5개 중 2개가 대기 중"이라는 요약만 보인다. Studio를 열면 각 세션의 구체적인 상태, 로그, diff, 비용을 한 화면에서 확인하고 조작할 수 있다.

**시나리오 1: 아침 출근 시 상태 확인**
1. Dock의 ham 아이콘 클릭 → Studio 열림
2. 좌측 트리에서 밤새 돌린 에이전트 3개 상태 확인
3. 1개는 done, 1개는 error (permission denied), 1개는 waiting_input
4. error 에이전트 선택 → 중앙에 로그, 우측에 에러 상세
5. "Go to Terminal" 클릭 → iTerm2에서 해당 세션으로 점프

**시나리오 2: 실시간 모니터링**
1. 복잡한 리팩토링 작업을 Claude Code에 맡김
2. Studio 열어두고 중앙 패널에서 로그 스트림 실시간 확인
3. tool call이 많은 구간에서 우측 패널의 비용 그래프 상승 확인
4. 필요 시 메시지 입력으로 방향 조정

#### 필요한 데이터

**현재 있는 것:**
- `Agent` 모델: ID, DisplayName, Status, StatusConfidence, SubAgents, TeamRole 등 (76개 필드 — `core/agent.go`)
- `AgentEventPayload`: ID, AgentID, Type, Summary, OccurredAt 등
- `DaemonRuntimeSnapshotPayload`: agents 배열, attention 정보
- `DaemonTeamPayload`: id, displayName, memberAgentIDs
- `SessionTarget`: itermSession, tmuxPane, externalURL, workspace

**새로 만들어야 하는 것:**
- `SessionGraph` / `SessionNode`: parent-child 트리 구조 (Phase 1 P1-2에서 정의됨)
- `InboxItem`: 알림/승인 요청 통합 모델 (Phase 1 P1-3에서 정의됨)
- `CostRecord`: 토큰/비용 집계 (Phase 1 P1-4에서 정의됨)
- `ArtifactPayload`: diff, output, error 등 artifact 데이터 (Phase 1 P1-1에서 정의됨)
- Studio 전용 UI 상태: 선택된 에이전트, 패널 토글, 필터 조건

#### Go 변경사항

| 파일 | 함수/타입 | 설명 |
|---|---|---|
| `go/internal/core/graph.go` | `SessionGraph`, `SessionNode`, `BuildGraph()` | P1-2에서 신규 생성. Studio의 좌측 트리 데이터 소스 |
| `go/internal/ipc/ipc.go` | `CommandSessionGraph` | 신규 커맨드 추가 |
| `go/internal/ipc/server.go` | `handleSessionGraph()` | SessionGraph 응답 핸들러 |
| `go/internal/runtime/registry.go` | (변경 없음) | 기존 `Agents()` 메서드로 데이터 제공 |

#### Swift 변경사항

| 파일 | 뷰/모델 | 설명 |
|---|---|---|
| `HamMenuBarApp.swift` | `HamStudioWindowPresenter` | 신규 싱글톤. NSWindow 생성, activation policy 전환 |
| `Sources/HamAppServices/StudioViewModel.swift` | `StudioViewModel` | 신규. 1초 polling, 200ms event follow, 패널 상태 관리 |
| `apps/macos/.../StudioRootView.swift` | `HamStudioRootView` | 신규. 3-패널 레이아웃 루트 |
| `apps/macos/.../StudioSidebarView.swift` | `StudioSidebarView` | 신규. 에이전트/팀 트리 |
| `apps/macos/.../StudioCenterView.swift` | `StudioCenterView` | 신규. 로그/출력 표시 영역 |
| `apps/macos/.../StudioInspectorView.swift` | `StudioInspectorView` | 신규. diff, approvals, cost, context |
| `Sources/HamCore/DaemonIPC.swift` | `DaemonCommand.sessionGraph` | 신규 커맨드 enum case |

#### IPC 변경사항

**신규 커맨드: `session.graph`**

```json
// Request
{ "command": "session.graph" }

// Response
{
  "session_graph": {
    "roots": [
      {
        "agent": { /* Agent */ },
        "children": [ /* SessionNode[] */ ],
        "block_reason": "waiting_input",
        "depth": 0
      }
    ],
    "total_count": 5,
    "blocked_count": 2,
    "generated_at": "2026-04-06T10:00:00Z"
  }
}
```

#### 선행 작업 / 의존성

- **필수:** P1-0 (IPC enum 동기화 H-10), P1-2 (SessionGraph 데이터 모델)
- **권장:** P1-1 (이벤트 스키마 확장 — artifact 표시에 필요)
- **권장:** P1-5 (EventBus — 실시간 업데이트 품질 향상)

#### 구현 불가능한 부분과 대안

| 불가능 | 이유 | 대안 |
|---|---|---|
| 터미널 에뮬레이션 (중앙 패널) | macOS SwiftUI에는 내장 터미널 위젯 없음. VT100 파서 직접 구현은 범위 초과 | **이벤트 로그 스트림**으로 대체. tool call, status 변경, assistant message를 시간순으로 표시. "Go to Terminal" 버튼으로 실제 터미널(iTerm2/tmux)로 점프 |
| Claude Code 세션 내부 stdin/stdout 직접 접근 | Claude Code는 외부에 터미널 I/O를 노출하지 않음 | hook 이벤트 + event follow로 간접 관측. `last_assistant_message`, `last_user_visible_summary` 활용 |
| 창 간 실시간 동기화 | MenuBarViewModel과 StudioViewModel이 별도 polling | 동일 DaemonClient 공유 + EventBus(P1-5) 완료 시 단일 subscription으로 전환 |

---

### 5.2 Agent Team Orchestrator

Phase 2 에서는 embedded PTY 탭과 동일한 Studio 윈도우 안에서 제공된다.

#### 기능 설명 + 사용자 시나리오

팀 리드/작업자 구조를 시각화하고 오케스트레이션을 관리하는 UI. 현재 메뉴바의 crown badge와 subagent tree를 정식 오케스트레이션 인터페이스로 확장한다.

**시나리오: 대규모 리팩토링**
1. 팀 리드 에이전트가 "API v2 마이그레이션"을 분해
2. Studio 좌측에서 팀 트리 확인: Lead → Worker A (routes), Worker B (models), Worker C (tests)
3. 각 worker의 worktree 격리 상태를 우측 패널에서 확인 (branch 이름, 변경 파일 수)
4. Worker B가 conflict 발생 → 상태가 `error`로 변경
5. concurrency budget (동시 3개) 내에서 Worker D를 추가 투입

#### 필요한 데이터

**현재 있는 것:**
- `Agent.SubAgents: []SubAgentInfo` — 현재 sub-agent 목록 (agent_id, display_name, status, role)
- `Agent.SubAgentCount: int`
- `Agent.TeamRole: string` — "lead" | "worker" | ""
- `Agent.TeamTaskTotal / TeamTaskCompleted: int`
- `DaemonTeamPayload`: id, displayName, memberAgentIDs
- `DaemonCommand.createTeam`, `addTeamMember`, `listTeams` — 기존 IPC

**새로 만들어야 하는 것:**
- `TeamOrchestratorState`: concurrency budget, merge gate 상태, task contract 목록
- `WorktreeInfo`: branch name, changed files count, conflict status
- `TaskContract`: task name, assignee, input spec, success criteria, status
- `MergeGate`: pending merges, conflict detection, auto-merge eligibility

#### Go 변경사항

| 파일 | 함수/타입 | 설명 |
|---|---|---|
| `go/internal/core/team.go` | `TeamOrchestratorState`, `TaskContract`, `WorktreeInfo`, `MergeGate` | 신규 타입 정의 |
| `go/internal/runtime/orchestrator.go` | `Orchestrator` | 신규. team 상태 관리, concurrency budget 적용 |
| `go/internal/ipc/ipc.go` | `CommandTeamOrchestrate`, `CommandTeamTaskList`, `CommandTeamMergeGate` | 신규 커맨드 3개 |
| `go/internal/ipc/server.go` | `handleTeamOrchestrate()`, `handleTeamTaskList()`, `handleTeamMergeGate()` | 신규 핸들러 |
| `go/internal/adapters/git.go` | `WorktreeScanner` | 신규. `git worktree list` 파싱, conflict 탐지 |

#### Swift 변경사항

| 파일 | 뷰/모델 | 설명 |
|---|---|---|
| `StudioSidebarView.swift` | Team tree section | 팀 그룹핑, lead/worker 아이콘 구분, task progress bar |
| `StudioInspectorView.swift` | Team detail section | worktree 격리 상태, merge gate, concurrency budget 표시 |
| `Sources/HamAppServices/TeamOrchestratorModel.swift` | `TeamOrchestratorModel` | 신규. team orchestration 상태 뷰 모델 |
| `Sources/HamCore/DaemonIPC.swift` | 신규 커맨드 3개, 신규 payload 타입들 | IPC 타입 확장 |

#### IPC 변경사항

**신규 커맨드: `teams.orchestrate`**

```json
// Request
{
  "command": "teams.orchestrate",
  "team_ref": "team-api-migration",
  "concurrency_budget": 3,
  "merge_strategy": "sequential"
}

// Response
{
  "team_orchestrator": {
    "team_id": "team-api-migration",
    "concurrency_budget": 3,
    "active_workers": 2,
    "tasks": [ /* TaskContract[] */ ],
    "merge_gate": { "pending": 1, "conflicts": 0 },
    "worktrees": [ /* WorktreeInfo[] */ ]
  }
}
```

**신규 커맨드: `teams.tasks`**

```json
// Request
{ "command": "teams.tasks", "team_ref": "team-api-migration" }

// Response
{
  "tasks": [
    {
      "id": "task-001",
      "name": "Migrate route handlers",
      "assignee_agent_id": "agent-a",
      "status": "in_progress",
      "success_criteria": "All /v1/* routes have /v2/* equivalents",
      "created_at": "2026-04-06T09:00:00Z"
    }
  ]
}
```

**신규 커맨드: `teams.merge_gate`**

```json
// Request
{ "command": "teams.merge_gate", "team_ref": "team-api-migration" }

// Response
{
  "merge_gate": {
    "pending_merges": [
      {
        "worker_agent_id": "agent-a",
        "branch": "worktree/api-migration-routes",
        "changed_files": 12,
        "conflicts_with": [],
        "eligible_for_auto_merge": true
      }
    ]
  }
}
```

#### 선행 작업 / 의존성

- **필수:** P1-2 (SessionGraph — parent-child 트리 구조)
- **필수:** 5.1 ham Studio 윈도우 (3-패널 레이아웃)
- **권장:** P1-1 (이벤트 스키마 — task 이벤트 캡처)

#### 구현 불가능한 부분과 대안

| 불가능 | 이유 | 대안 |
|---|---|---|
| Claude Code subagent 직접 생성/제어 | Claude Code에 외부에서 subagent를 spawn하는 API 없음 | ham은 **관측 + 시각화** 역할. Claude Code의 agent teams가 자체적으로 subagent를 생성하면 hook을 통해 감지하고 표시. ham 측 managed agent(`run.managed`)로 별도 세션을 시작하는 것은 가능 |
| Worktree 자동 생성/삭제 | git worktree 조작은 Claude Code 세션 내부에서 수행됨 | `git.go` adapter로 worktree 목록을 **읽기 전용**으로 스캔. 생성/삭제는 playbook이나 사용자 명령으로 |
| 실시간 conflict 감지 | 파일 변경 감시(FSEvents)를 모든 worktree에 걸면 리소스 과다 | team task가 `done`으로 전환될 때 한 번 conflict 체크 수행 |

---

### 5.3 Playbooks / Recipes

Phase 2 에서는 embedded PTY 탭과 동일한 Studio 윈도우 안에서 제공된다.

#### 기능 설명 + 사용자 시나리오

반복 작업을 선언형으로 저장하고 실행하는 시스템. Playbook은 단순 프롬프트 템플릿이 아니라, 목적/입력/성공 기준/도구/hooks/approvals/post-check을 묶은 **작업 계약서**다.

**시나리오 1: PR 리뷰 playbook**
1. Studio에서 "Playbooks" 탭 열기
2. "PR Review Standard" playbook 선택
3. PR URL 입력 → playbook이 정의한 단계 실행:
   - Step 1: checkout + diff 분석 (자동)
   - Step 2: 보안 이슈 스캔 (자동)
   - Step 3: 리뷰 코멘트 생성 (human review 필요 → approval 요청)
   - Step 4: 코멘트 게시 (approval 후 자동)

**시나리오 2: 팀 공유 playbook**
1. 시니어 엔지니어가 `.ham/playbooks/service-migration.yaml` 작성
2. git push → 팀원 모두 해당 repo에서 playbook 사용 가능
3. Studio에서 repo-local playbook 목록 표시

#### 필요한 데이터

**현재 있는 것:**
- Claude Code의 skills/plugins 포맷 (CLAUDE.md 기반)
- `DaemonCommand.runManaged` — managed agent 실행 (playbook 실행 시 활용 가능)
- Agent의 `Role` 필드 — playbook 단계별 역할 지정에 활용

**새로 만들어야 하는 것:**
- `Playbook` 스키마: YAML 기반 선언형 포맷
- `PlaybookExecution`: 실행 인스턴스 (진행률, 단계별 상태)
- `PlaybookRegistry`: repo-local / user-global / (향후) org-shared 검색 경로
- Playbook → Claude Code skills 변환 레이어

#### Playbook 스키마 (초안)

```yaml
# .ham/playbooks/pr-review.yaml
name: PR Review Standard
description: 코드 리뷰 표준 프로세스
version: 1
triggers:
  - event: "github.pull_request.opened"  # Phase 2.4 연동 시
inputs:
  - name: pr_url
    type: string
    required: true
steps:
  - name: analyze_diff
    prompt: "PR의 diff를 분석하고 주요 변경점을 요약해줘"
    tools: [Read, Bash]
    auto_approve: true
  - name: security_scan
    prompt: "보안 취약점이 있는지 확인해줘"
    tools: [Read, Bash]
    auto_approve: true
  - name: generate_review
    prompt: "리뷰 코멘트를 작성해줘"
    tools: [Read]
    requires_approval: true
    approval_prompt: "리뷰 코멘트를 확인하시겠습니까?"
  - name: post_review
    prompt: "리뷰를 GitHub에 게시해줘"
    tools: [Bash]
    requires_approval: false  # 이전 단계에서 이미 승인됨
success_criteria: "리뷰가 PR에 코멘트로 게시됨"
```

#### Go 변경사항

| 파일 | 함수/타입 | 설명 |
|---|---|---|
| `go/internal/core/playbook.go` | `Playbook`, `PlaybookStep`, `PlaybookExecution` | 신규 타입 |
| `go/internal/runtime/playbook_runner.go` | `PlaybookRunner` | 신규. playbook 실행 엔진 (step 순회, approval gate) |
| `go/internal/store/playbook_store.go` | `PlaybookStore` | 신규. YAML 로드, 검색 경로 관리 |
| `go/internal/ipc/ipc.go` | `CommandPlaybookList`, `CommandPlaybookRun`, `CommandPlaybookStatus` | 신규 커맨드 3개 |
| `go/internal/ipc/server.go` | 핸들러 3개 | 신규 |

#### Swift 변경사항

| 파일 | 뷰/모델 | 설명 |
|---|---|---|
| `apps/macos/.../StudioPlaybookView.swift` | `StudioPlaybookView` | 신규. playbook 목록, 실행 UI |
| `apps/macos/.../PlaybookExecutionView.swift` | `PlaybookExecutionView` | 신규. 단계별 진행률, approval gate 표시 |
| `Sources/HamAppServices/PlaybookModel.swift` | `PlaybookModel` | 신규. playbook 데이터 + 실행 상태 |
| `Sources/HamCore/DaemonIPC.swift` | 신규 커맨드, `PlaybookPayload`, `PlaybookExecutionPayload` | IPC 타입 확장 |

#### IPC 변경사항

**신규 커맨드: `playbooks.list`**

```json
// Request
{ "command": "playbooks.list", "project_path": "/path/to/repo" }

// Response
{
  "playbooks": [
    {
      "name": "PR Review Standard",
      "source": "repo-local",
      "path": ".ham/playbooks/pr-review.yaml",
      "step_count": 4
    }
  ]
}
```

**신규 커맨드: `playbooks.run`**

```json
// Request
{
  "command": "playbooks.run",
  "playbook_name": "PR Review Standard",
  "project_path": "/path/to/repo",
  "inputs": { "pr_url": "https://github.com/org/repo/pull/123" }
}

// Response
{
  "execution": {
    "id": "exec-001",
    "playbook_name": "PR Review Standard",
    "status": "running",
    "current_step": 0,
    "agent_id": "agent-playbook-001"
  }
}
```

**신규 커맨드: `playbooks.execution`**

```json
// Request
{ "command": "playbooks.execution", "execution_id": "exec-001" }

// Response
{
  "execution": {
    "id": "exec-001",
    "status": "waiting_approval",
    "current_step": 2,
    "steps": [
      { "name": "analyze_diff", "status": "done" },
      { "name": "security_scan", "status": "done" },
      { "name": "generate_review", "status": "waiting_approval" },
      { "name": "post_review", "status": "pending" }
    ]
  }
}
```

#### 선행 작업 / 의존성

- **필수:** 5.1 ham Studio 윈도우
- **필수:** `run.managed` IPC 커맨드 (이미 존재) — playbook step 실행에 사용
- **권장:** P1-1 (Artifact Capture — step 결과물 저장)
- **권장:** P1-3 (Inbox — approval 요청 표시)

#### 구현 불가능한 부분과 대안

| 불가능 | 이유 | 대안 |
|---|---|---|
| Claude Code skills로 직접 컴파일 | skills 포맷이 내부용이고 외부 생성 API 없음 | ham playbook을 자체 포맷으로 유지. 실행 시 `run.managed`로 각 step의 prompt를 Claude Code에 전달 |
| org-shared playbook 동기화 | 중앙 서버 없음 (localOnlyMode 기본) | v1에서는 repo-local + `~/.ham/playbooks/` (user-global)만 지원. org-shared는 git submodule이나 symlink로 우회 |
| Step 간 상태 자동 전달 | Claude Code 세션 간 컨텍스트 공유 메커니즘 없음 | 단일 managed session 안에서 모든 step 순차 실행. step 경계는 ham이 prompt injection으로 구분 |

---

### 5.4 Git/CI/Issue 연동

Phase 2 에서는 embedded PTY 탭과 동일한 Studio 윈도우 안에서 제공된다.

#### 기능 설명 + 사용자 시나리오

PR 업데이트, CI 실패, 이슈 생성 같은 외부 이벤트가 ham을 통해 Claude Code 세션을 트리거한다. 폴링이 아닌 event-driven 방식으로 운영한다.

**시나리오 1: CI 실패 자동 triage**
1. GitHub Actions에서 CI 실패 발생
2. GitHub webhook → ham의 로컬 webhook receiver → InboxItem 생성
3. Studio에서 "CI failure: test_api_routes" 알림 표시
4. "Run Triage Playbook" 클릭 → CI 실패 triage playbook 자동 실행
5. Claude Code가 실패 로그 분석 → 수정 PR 생성

**시나리오 2: PR 코멘트 대응**
1. 리뷰어가 PR에 코멘트 남김
2. GitHub webhook → ham → InboxItem
3. Studio에서 코멘트 내용 확인
4. "Address Comment" 클릭 → 해당 PR의 worktree에서 Claude Code 세션 시작

#### 필요한 데이터

**현재 있는 것:**
- `DaemonIntegrationSettingsPayload`: itermEnabled, transcriptDirs, providerAdapters
- hook 시스템: Claude Code → hamd 방향의 이벤트는 이미 수신 중

**새로 만들어야 하는 것:**
- `ExternalEventSource`: GitHub webhook, CI webhook, file watcher 등 외부 이벤트 소스
- `ExternalEvent`: 외부 이벤트 통합 모델 (PR update, CI failure, issue creation)
- `WebhookReceiver`: 로컬 HTTP 서버 (ham daemon 내부)
- `EventTriggerRule`: 이벤트 → 액션 매핑 규칙

#### Go 변경사항

| 파일 | 함수/타입 | 설명 |
|---|---|---|
| `go/internal/core/external_event.go` | `ExternalEvent`, `ExternalEventSource`, `EventTriggerRule` | 신규 타입 |
| `go/internal/adapters/github_webhook.go` | `GitHubWebhookHandler` | 신규. GitHub webhook 파싱 (push, pull_request, check_run, issues) |
| `go/internal/runtime/webhook_server.go` | `WebhookServer` | 신규. 로컬 HTTP 서버 (포트 설정 가능, localhost only) |
| `go/internal/runtime/event_trigger.go` | `EventTriggerEngine` | 신규. rule 기반 이벤트 → 액션 매핑 |
| `go/internal/ipc/ipc.go` | `CommandExternalEvents`, `CommandTriggerRuleList`, `CommandTriggerRuleCreate` | 신규 커맨드 |
| `go/internal/ipc/server.go` | 핸들러 추가 | 신규 |

#### Swift 변경사항

| 파일 | 뷰/모델 | 설명 |
|---|---|---|
| `StudioInspectorView.swift` | External events section | 외부 이벤트 타임라인 표시 |
| `StudioSidebarView.swift` | Event source indicators | 연동된 소스 표시 (GitHub 아이콘, CI 아이콘) |
| `Sources/HamAppServices/ExternalEventModel.swift` | `ExternalEventModel` | 신규. 외부 이벤트 뷰 모델 |
| `Sources/HamCore/DaemonIPC.swift` | 신규 커맨드, `ExternalEventPayload` | IPC 타입 확장 |

#### IPC 변경사항

**신규 커맨드: `external.events`**

```json
// Request
{ "command": "external.events", "limit": 20 }

// Response
{
  "external_events": [
    {
      "id": "ext-001",
      "source": "github",
      "type": "pull_request.review_comment",
      "repo": "org/repo",
      "ref": "PR #123",
      "summary": "리뷰어가 코멘트를 남겼습니다: 'Error handling 추가 필요'",
      "occurred_at": "2026-04-06T10:30:00Z",
      "actionable": true,
      "suggested_playbook": "address-review-comment"
    }
  ]
}
```

**신규 커맨드: `triggers.list` / `triggers.create`**

```json
// Request (create)
{
  "command": "triggers.create",
  "trigger": {
    "event_source": "github",
    "event_type": "check_run.completed",
    "condition": "conclusion == 'failure'",
    "action": "playbook:ci-failure-triage",
    "project_path": "/path/to/repo"
  }
}
```

#### P2-4 Webhook Security Model

- **바인딩**: localhost (127.0.0.1) only, 외부 네트워크 노출 금지
- **인증**: 랜덤 shared secret, 생성 시 keychain 저장 (Swift Keychain API)
- **GitHub secret 보관**: OS keychain 전용, 파일 저장 금지
- **최소 scope**: GitHub token 은 `repo:status`, `pull_requests:read` 만. admin/write 금지
- **실패 모드**: 시크릿 누출 감지 시 토큰 즉시 revoke 요청, 사용자에게 메뉴바 경고
- **로그**: webhook request body 에서 secret 은 마스킹 저장

#### 선행 작업 / 의존성

- **필수:** 5.1 ham Studio 윈도우
- **필수:** P1-3 (Inbox — 외부 이벤트를 InboxItem으로 통합)
- **필수:** P1-5 (EventBus — 외부 이벤트 브로드캐스트)
- **권장:** 5.3 Playbooks (이벤트 → playbook 자동 실행)

#### 구현 불가능한 부분과 대안

| 불가능 | 이유 | 대안 |
|---|---|---|
| GitHub App 수준의 webhook 수신 | 로컬 머신에 퍼블릭 endpoint 없음 | **방법 A:** `gh` CLI로 polling (`gh api repos/.../events` — 30초 간격). **방법 B:** ngrok/cloudflared 터널 (사용자 설정). **방법 C:** GitHub Actions에서 `ham` CLI를 통해 push 알림 |
| 모든 CI 시스템 통합 | Jenkins, CircleCI, GitLab CI 등 파편화 | v1에서는 GitHub Actions만 지원. 범용 webhook endpoint(`POST /webhook/generic`)로 다른 CI 연동 가능하게 열어둠 |
| 실시간 push | 로컬 앱이 webhook을 직접 수신하려면 터널 필요 | Claude Code의 `channels` 기능 활용 검토. 또는 `gh` CLI polling + EventBus로 준실시간 달성 |

---

### 5.5 Review Loop

Phase 2 에서는 embedded PTY 탭과 동일한 Studio 윈도우 안에서 제공된다.

#### 기능 설명 + 사용자 시나리오

에이전트의 완료 기준을 "코드를 생성했다"에서 "검토 가능한 산출물로 정리됐다"로 격상한다. 체크포인트, 되감기, human review queue, ready-for-merge 상태를 관리한다.

**시나리오: 리팩토링 결과 리뷰**
1. Claude Code가 리팩토링 완료 → status `done`
2. ham이 자동으로 checkpoint 생성 (git stash 또는 branch snapshot)
3. Studio 우측 패널에 "Review Queue"에 항목 추가
4. 개발자가 diff 확인 → 부분 수정 요청
5. "Rewind to Checkpoint #2" → 해당 시점으로 복원
6. 수정 후 "Approve" → ready-for-merge 상태로 전환
7. merge gate 통과 → 자동 merge (또는 수동 merge)

#### 필요한 데이터

**현재 있는 것:**
- `Agent.Status: done` — 에이전트 완료 감지
- `AgentEventPayload` — 이벤트 히스토리로 작업 추적
- `Agent.LastAssistantMessage` — 최종 메시지

**새로 만들어야 하는 것:**
- `Checkpoint`: git ref (commit hash / stash ref), 생성 시점, 에이전트 ID, 요약
- `ReviewItem`: review queue 항목 (agent, checkpoint, diff summary, status)
- `ReviewStatus`: pending / in_review / approved / changes_requested / merged
- `RewindRequest`: 특정 checkpoint로 복원 요청

#### Go 변경사항

| 파일 | 함수/타입 | 설명 |
|---|---|---|
| `go/internal/core/review.go` | `Checkpoint`, `ReviewItem`, `ReviewStatus` | 신규 타입 |
| `go/internal/runtime/checkpoint_manager.go` | `CheckpointManager` | 신규. agent done 이벤트 시 자동 checkpoint 생성 |
| `go/internal/runtime/review_queue.go` | `ReviewQueue` | 신규. review item 관리, status 전환 |
| `go/internal/adapters/git.go` | `CreateCheckpoint()`, `RewindToCheckpoint()` | git stash / branch 조작 |
| `go/internal/ipc/ipc.go` | `CommandReviewList`, `CommandReviewApprove`, `CommandCheckpointRewind` | 신규 커맨드 3개 |
| `go/internal/ipc/server.go` | 핸들러 3개 | 신규 |

#### Swift 변경사항

| 파일 | 뷰/모델 | 설명 |
|---|---|---|
| `StudioInspectorView.swift` | Review Queue section | review 항목 목록, diff 요약, approve/request changes 버튼 |
| `apps/macos/.../CheckpointTimelineView.swift` | `CheckpointTimelineView` | 신규. 체크포인트 타임라인, 되감기 UI |
| `Sources/HamAppServices/ReviewModel.swift` | `ReviewModel` | 신규. review queue + checkpoint 뷰 모델 |
| `Sources/HamCore/DaemonIPC.swift` | 신규 커맨드, `CheckpointPayload`, `ReviewItemPayload` | IPC 타입 확장 |

#### IPC 변경사항

**신규 커맨드: `review.list`**

```json
// Request
{ "command": "review.list" }

// Response
{
  "review_items": [
    {
      "id": "review-001",
      "agent_id": "agent-refactor",
      "agent_name": "api-refactor",
      "status": "pending",
      "checkpoint_id": "chk-003",
      "diff_summary": "+142 -38 across 5 files",
      "created_at": "2026-04-06T11:00:00Z"
    }
  ]
}
```

**신규 커맨드: `review.approve`**

```json
// Request
{
  "command": "review.approve",
  "review_id": "review-001",
  "action": "approve"  // "approve" | "changes_requested" | "reject"
}
```

**신규 커맨드: `checkpoints.rewind`**

```json
// Request
{
  "command": "checkpoints.rewind",
  "checkpoint_id": "chk-002",
  "agent_id": "agent-refactor"
}

// Response
{
  "rewind": {
    "checkpoint_id": "chk-002",
    "git_ref": "abc1234",
    "status": "rewound",
    "files_restored": 5
  }
}
```

#### 선행 작업 / 의존성

- **필수:** 5.1 ham Studio 윈도우
- **필수:** P1-1 (Artifact Capture — diff를 artifact로 저장)
- **필수:** P1-5 (EventBus — agent done 이벤트 구독)
- **권장:** 5.2 Agent Team Orchestrator (merge gate 연동)

#### 구현 불가능한 부분과 대안

| 불가능 | 이유 | 대안 |
|---|---|---|
| Claude Code 세션 상태 되감기 | Claude Code에 세션 내부 상태를 외부에서 조작하는 API 없음 | **git 수준 되감기**만 지원. `git checkout <checkpoint-ref>` 또는 `git stash pop`. Claude Code 세션은 새로 시작해야 함 |
| Inline diff 에디터 | SwiftUI에 Monaco Editor 급 diff 뷰 없음 | `git diff` 출력을 syntax-highlighted text로 표시. 상세 편집이 필요하면 "Open in VS Code" 버튼 |
| 자동 merge conflict 해결 | 안전성 보장 불가 | conflict 감지 + 알림만 수행. 해결은 사용자 또는 Claude Code 세션에 위임 |

---

### 5.6 Approval Inbox 업그레이드

> **라운드 3 업데이트**: 아래 5.6 Approval Inbox 섹션은 라운드 2 내용으로, 당시에는 "외부 permission API 미확인" 가정 하에 3 가지 대안 (A/B/C) 을 검토했다. 라운드 3 에서는 P2-3 Approval Interception (PTY 기반 차단) 이 primary path 로 채택됐다. 아래 5.6 섹션은 **P2-3 spike 가 실패할 경우의 fallback 대안** 으로 남겨둔다.

Phase 2 에서는 embedded PTY 탭과 동일한 Studio 윈도우 안에서 제공된다.

#### 기능 설명 + 사용자 시나리오

Phase 1의 읽기 전용 Notification Inbox를 승인/거절 가능한 Approval Inbox로 업그레이드한다. **단, ADR-7 (Phase 2 P2-3 Approval Interception) 에서 확인된 경우에만 구현.**

**시나리오 (API 가용 시):**
1. Claude Code가 `rm -rf` 실행 시도 → permission request 발생
2. hook을 통해 ham에 permission_request 이벤트 도달
3. Studio 우측 패널 "Approval Queue"에 표시:
   - Agent: api-refactor
   - Tool: Bash
   - Command: `rm -rf ./dist`
   - Risk: HIGH
4. 개발자가 "Approve" 또는 "Deny" 클릭
5. ham → Claude Code에 approval 전달 → 실행 계속 또는 중단

#### 필요한 데이터

**현재 있는 것:**
- `InboxItem` (P1-3): type이 `permission_request`인 항목
- `InboxItem.Actionable`: Phase 1에서는 항상 `false`
- hook payload: `hook.permission-request`에서 ToolName, Description 등

**새로 만들어야 하는 것 (API 가용 시):**
- `ApprovalAction`: approve / deny / approve_once / approve_always
- `ApprovalTransport`: Claude Code에 approval을 전달하는 메커니즘
- `ApprovalPolicy`: tool별 자동 승인 규칙 (Read는 항상 승인, Bash는 항상 확인 등)

#### Go 변경사항 (API 가용 시)

| 파일 | 함수/타입 | 설명 |
|---|---|---|
| `go/internal/core/inbox.go` | `ApprovalAction`, `ApprovalPolicy` | 기존 InboxItem에 필드 추가 |
| `go/internal/runtime/approval_bridge.go` | `ApprovalBridge` | 신규. Claude Code ↔ ham 승인 전달 브릿지 |
| `go/internal/ipc/ipc.go` | `CommandInboxApprove` | 신규 커맨드 |
| `go/internal/ipc/server.go` | `handleInboxApprove()` | 신규 핸들러 |

#### Swift 변경사항 (API 가용 시)

| 파일 | 뷰/모델 | 설명 |
|---|---|---|
| `StudioInspectorView.swift` | Approval Queue section | Approve/Deny 버튼 활성화, risk level 표시 |
| `Sources/HamAppServices/InboxViewModel.swift` | `approveItem()`, `denyItem()` | P1-3의 읽기 전용 모델에 액션 추가 |
| `Sources/HamCore/DaemonIPC.swift` | `CommandInboxApprove`, `ApprovalActionPayload` | IPC 타입 확장 |

#### IPC 변경사항 (API 가용 시)

**신규 커맨드: `inbox.approve`**

```json
// Request
{
  "command": "inbox.approve",
  "inbox_item_id": "inbox-001",
  "action": "approve",          // "approve" | "deny" | "approve_once" | "approve_always"
  "policy_scope": "session"     // "once" | "session" | "agent" | "global"
}

// Response
{
  "approval": {
    "inbox_item_id": "inbox-001",
    "action": "approve",
    "delivered": true,
    "agent_resumed": true
  }
}
```

#### 선행 작업 / 의존성

- **필수:** ADR-7 결론 (P2-3 Approval Interception 을 통해 해소)
- **필수:** P1-3 (Notification Inbox — 기반 인프라)
- **필수:** 5.1 ham Studio 윈도우

#### 구현 불가능한 부분과 대안

| 불가능 | 이유 | 대안 |
|---|---|---|
| Claude Code permission 외부 승인 (ADR-7 / P2-3) | 2026.04 기준 공개 API 미확인 | **대안 A:** Claude Code MCP server로 ham이 approval tool을 제공 → Claude Code가 ham에 tool call로 승인 요청 → ham이 사용자에게 UI 표시 → 결과를 tool response로 반환. **대안 B:** iTerm2 AppleScript로 키 입력 시뮬레이션 (y/n 입력) — fragile하지만 동작 가능. **대안 C:** API 공개 대기 후 Phase 3에서 구현 |
| 실시간 승인 전달 | Unix socket은 request-response. daemon → Claude Code 방향 push 불가 | Claude Code 측에서 ham daemon을 polling하도록 MCP tool 설계. 또는 Claude Code hooks의 return value로 approval 전달 가능한지 조사 |

---

## 6. Graceful Degradation 전략

Phase 1 각 항목의 완료 여부에 따라 Studio 기능이 점진적으로 활성화되도록 설계한다.

### 의존성 매트릭스

| Studio 기능 | P1-0 필수 | P1-1 권장 | P1-2 필수 | P1-3 필수 | P1-4 권장 | P1-5 권장 |
|---|---|---|---|---|---|---|
| 5.1 Studio 윈도우 | O | - | O | - | - | - |
| 5.2 Team Orchestrator | O | - | O | - | - | - |
| 5.3 Playbooks | O | O | - | O | - | - |
| 5.4 Git/CI/Issue | O | - | - | O | - | O |
| 5.5 Review Loop | O | O | - | - | - | O |
| 5.6 Approval Inbox | O | - | - | O (ADR-7 / P2-3) | - | - |

### Phase 1 미완 시 대응

| P1 항목 미완 | 영향받는 Studio 기능 | Degradation 전략 |
|---|---|---|
| P1-1 미완 (Artifact) | 중앙 패널에 artifact/diff 표시 불가 | `last_assistant_message`와 `last_user_visible_summary`만 표시. artifact 영역은 "Phase 1 완료 후 활성화" placeholder |
| P1-2 미완 (SessionGraph) | 좌측 트리가 flat list로 표시 | 현재 `MenuBarViewModel`의 agents 배열을 그대로 사용. SubAgents 필드로 1-depth 그룹핑만 수행 |
| P1-3 미완 (Inbox) | 알림/승인 표시 불가 | 우측 패널에서 Inbox 섹션 숨김. 에이전트 status로만 attention 판단 |
| P1-4 미완 (Cost) | 비용 표시 불가 | 우측 패널 비용 섹션 숨김. "비용 데이터 수집 설정 필요" 안내 |
| P1-5 미완 (EventBus) | 실시간성 저하 | 기존 `events.follow` long-polling 유지. 200ms polling으로 준실시간 |

### 기능 가용성 확인 패턴

```swift
// StudioViewModel에서 daemon capabilities 확인
struct DaemonCapabilities {
    var hasSessionGraph: Bool      // P1-2
    var hasInbox: Bool             // P1-3
    var hasCostTelemetry: Bool     // P1-4
    var hasEventBus: Bool          // P1-5
    var hasArtifacts: Bool         // P1-1
    var hasApprovalAPI: Bool       // ADR-7
}

// 사용 예시
if viewModel.capabilities.hasSessionGraph {
    SessionTreeView(graph: viewModel.sessionGraph)
} else {
    FlatAgentListView(agents: viewModel.agents)
}
```

daemon에 `capabilities` 커맨드를 추가하여 지원 기능을 응답하거나, 기존 커맨드에 대한 error response로 감지한다.

---

## 7. 경쟁 제품 참조

### Cursor Sidebar (Agent Panel)

- **URL:** https://docs.cursor.com/chat/overview
- **참고 포인트:**
  - Chat + Agent 모드 전환이 하나의 사이드 패널 안에서 이루어짐
  - Agent 실행 중 tool call을 inline으로 표시 (파일 읽기, 편집, 터미널 실행)
  - "Accept All" / "Reject All" 버튼으로 변경 일괄 처리
  - Checkpoint 자동 생성 → 되감기 지원
- **ham Studio에 적용할 것:** Checkpoint + 되감기 UX. 단, Cursor는 에디터 내장이고 ham은 외부 관측자이므로 git-level checkpoint만 구현
- **적용하지 않을 것:** 인라인 에디터. ham은 코드 편집기가 아님

### Warp Terminal (Agent Mode)

- **URL:** https://docs.warp.dev/features/warp-ai/agent-mode
- **참고 포인트:**
  - 터미널 안에서 AI agent가 명령을 제안하고 실행
  - Step-by-step 실행 with human approval
  - 실행 결과를 블록 단위로 구분하여 표시
- **ham Studio에 적용할 것:** Step-by-step approval UX 패턴. 블록 단위 결과 표시
- **적용하지 않을 것:** 터미널 에뮬레이션 자체. ham은 기존 터미널(iTerm2/tmux)로 위임

### Windsurf (Cascade)

- **URL:** https://docs.windsurf.com/windsurf/cascade
- **참고 포인트:**
  - Multi-file 변경을 하나의 "flow"로 묶어 표시
  - Flow 내에서 각 파일 변경의 diff를 접었다 펼 수 있음
  - "Memory" 시스템으로 프로젝트 컨텍스트 유지
- **ham Studio에 적용할 것:** 작업 단위(flow) 개념을 Review Loop의 ReviewItem에 적용. Artifact 그룹핑
- **적용하지 않을 것:** 에디터 내장 diff. ham은 외부 diff viewer로 위임

### Anthropic Claude Code (Teams / Headless)

- **URL:** https://docs.anthropic.com/en/docs/claude-code
- **참고 포인트:**
  - Agent teams: lead + workers 패턴
  - Headless mode: CI/scheduled 실행
  - Hooks: 외부 시스템 연동 포인트
  - Channels: 외부에서 세션에 메시지 전달
- **ham Studio에 적용할 것:** Claude Code의 native 기능을 최대한 활용. ham은 이 기능들의 **운영 레이어**를 제공. 자체 agent runtime을 만들지 않음
- **적용하지 않을 것:** Claude Code 기능 중복 구현. ham은 보완재이지 대체재가 아님

---

## 부록: 구현 우선순위

Phase 2 내부 구현 순서 제안:

```
Phase 2.1: Studio 윈도우 셸 (5.1)
  → NSWindow, 3-패널 레이아웃, activation policy 전환
  → StudioViewModel + 기존 데이터로 동작하는 최소 UI

Phase 2.2: Review Loop (5.5)
  → checkpoint, review queue, diff 표시
  → 개발자 1명이 즉시 체감하는 가치

Phase 2.3: Team Orchestrator (5.2)
  → team tree, worktree 표시, task contract
  → multi-agent 사용자에게 가치

Phase 2.4: Playbooks (5.3)
  → YAML 스키마, 실행 엔진, UI
  → 반복 작업 자동화

Phase 2.5: Git/CI/Issue 연동 (5.4)
  → webhook receiver, trigger rules
  → event-driven 운영

Phase 2.6: Approval Inbox 업그레이드 (5.6)
  → ADR-7 / P2-3 결과에 따라 구현 여부 결정
```

이 순서는 "Studio 셸 → 단일 사용자 가치 → 팀 가치 → 자동화 → 외부 연동"의 점진적 확장을 따른다.
