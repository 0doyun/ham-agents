# tasks.md

## Purpose
이 문서는 **현재 활성 작업 범위와 실행 체크리스트**를 관리한다.

원칙:
- 분석 전에 이 문서를 과하게 확정하지 않는다.
- 먼저 `spec.md`와 `roadmap.md`를 읽고 현재 구현 범위를 정리한다.
- 이후 현재 버전에 필요한 작업만 작은 vertical slice로 쪼갠다.
- 미래 기능은 아이디어로 남길 수 있지만 현재 체크리스트에는 넣지 않는다.

---

## Current Status
- [x] spec / roadmap 기반 분석 완료
- [x] 전체 스펙 기준의 장기 backlog 정의 시작
- [x] 현재 활성 구현 범위 정의
- [x] architecture 초안 정리
- [x] assumptions 초안 정리
- [x] progress 로그 시작

---

## Product Goal

- [x] 최종 목표는 `spec.md`의 전체 제품 경험 구현
- [x] 구현은 작은 vertical slice 단위로 누적
- [x] 각 slice는 build/test 가능한 green 상태 유지

---

## Active Scope

현재 활성 범위는 **observed explicit error summary refinement baseline** 다.

- [x] 상세 스펙 복원 및 제품 truth 강화
- [x] `Swift UI + Go CLI/runtime` 방향으로 아키텍처 정렬
- [x] 장기 backlog를 에픽 단위로 정리
- [x] Ralph용 PRD / test spec 아티팩트 생성
- [x] 핵심 모듈 골격 생성
- [x] Git 원격과 연결된 실제 작업 트리로 전환
- [x] 저장소 레이아웃을 Swift UI / Go runtime 방향으로 실제 정렬
- [x] Go workspace bootstrap 추가: `go/cmd/ham`, `go/cmd/hamd`, `go/internal/{core,runtime,store,ipc,adapters}`
- [x] 첫 hybrid implementation slice 완료: managed session registry + `ham status/list`
- [x] `ham` ↔ `hamd` 실제 IPC 연결로 direct store path 축소
- [x] runtime event log / lifecycle foundation 추가
- [x] event feed를 CLI/daemon에서 조회 가능하게 노출
- [x] Swift가 daemon snapshot/event payload를 decode 할 수 있게 정렬
- [x] Swift가 daemon socket/command surface를 통해 snapshot + events를 읽을 수 있게 연결
- [x] Swift menu bar executable target과 baseline status surface 추가
- [x] menu bar 상태 surface가 launch 이후에도 daemon 상태를 주기적으로 따라가게 만들기
- [x] status transition 기반 notification trigger foundation 추가
- [x] actual macOS notification delivery sink 연결
- [x] popover에서 선택 agent detail + recent event context 표시
- [x] popover에서 최소 agent action 연결
- [x] iTerm/workspace opening action baseline 추가
- [x] notification permission 상태를 popover에서 인지/요청 가능하게 만들기
- [x] sessionRef URL 이 있으면 이를 우선 사용하고 없으면 workspace fallback 하도록 세분화
- [x] popover에서 agent별 notification pause/resume action 추가
- [x] popover에서 quick message baseline action 추가
- [x] iTerm이 있는 경우 quick message를 실제 terminal write 로 보내는 baseline 추가
- [x] quick message 성공/실패 feedback baseline 추가
- [x] notification pause/resume 을 daemon persistence 로 이관
- [x] selected agent role rename action 추가
- [x] selected agent stop-tracking baseline 추가
- [x] mode/confidence 를 popover에서 명시적으로 표시
- [x] `ham attach` minimal flow 추가
- [x] `ham observe` minimal flow 추가
- [x] observed source contents를 읽어 status/confidence를 갱신하는 baseline 추가
- [x] daemon serve 중 observed source polling 추가
- [x] `ham open <agent>` baseline 추가
- [x] backend settings state baseline 추가
- [x] Swift menu bar에서 settings를 읽고 일부 토글을 수정할 수 있게 연결
- [x] stored notification settings가 실제 delivery behavior 에 반영되게 연결
- [x] quiet hours enabled setting이 notification suppression에 반영되게 연결
- [x] daemon-backed `ham ask <agent> "..."` baseline 추가
- [x] quiet hours 시간대 범위를 저장/적용하는 baseline 추가
- [x] richer attached/iTerm session identification baseline 추가
- [x] attach picker / iTerm session listing baseline 추가
- [x] attached session termination detection baseline 추가
- [x] broader settings sections baseline 추가
- [x] live event stream / follow baseline 추가
- [x] richer attached metadata sync baseline 추가
- [x] stronger settings sections baseline 추가
- [x] event-driven UI refresh baseline 추가
- [x] richer attached cwd/activity metadata baseline 추가
- [x] higher-fidelity event-driven UI update baseline 추가
- [x] richer attached shell-state fidelity baseline 추가
- [x] stronger event semantics baseline 추가
- [x] lower-latency UI update baseline 추가
- [x] richer event-driven UI semantics baseline 추가
- [x] lower-latency visual updates baseline 추가
- [x] stronger feed semantics baseline 추가
- [x] lower-latency visual polish baseline 추가
- [x] attached shell-state heuristic refinement baseline 추가
- [x] observed lifecycle event baseline 추가
- [x] status reason baseline 추가
- [x] confidence/reason refinement baseline 추가
- [x] attention queue baseline 추가
- [x] severity-aware feed ordering baseline 추가
- [x] runtime lifecycle transition baseline 추가
- [x] runtime coordinator baseline 추가
- [x] runtime transition consistency baseline 추가
- [x] richer attached shell-state fidelity follow-up 추가
- [x] runtime coordinator follow-up 추가
- [x] runtime lifecycle coverage follow-up 추가
- [x] attention queue follow-up 추가
- [x] CLI confidence/reason visibility baseline 추가
- [x] CLI attention detail baseline 추가
- [x] CLI attention breakdown baseline 추가
- [x] CLI stop baseline 추가
- [x] CLI logs baseline 추가
- [x] CLI list summary baseline 추가
- [x] CLI doctor baseline 추가
- [x] severity-aware feed scanning baseline 추가
- [x] event JSON writer consistency baseline 추가
- [x] daemon-backed attention summary baseline 추가
- [x] daemon-backed attention breakdown UI baseline 추가
- [x] daemon-backed attention ordering baseline 추가
- [x] daemon-backed attention subtitle baseline 추가
- [x] CLI status attention contract baseline 추가
- [x] CLI status attention subtitle contract baseline 추가
- [x] CLI ui baseline 추가
- [x] lifecycle-aware event presentation baseline 추가
- [x] daemon-backed event presentation hint baseline 추가
- [x] CLI event presentation hint contract baseline 추가
- [x] daemon-backed lifecycle summary baseline 추가
- [x] CLI event presentation summary contract baseline 추가
- [x] daemon-backed lifecycle metadata baseline 추가
- [x] CLI event lifecycle metadata contract baseline 추가
- [x] daemon-backed lifecycle reason baseline 추가
- [x] CLI event lifecycle reason contract baseline 추가
- [x] daemon-backed lifecycle confidence baseline 추가
- [x] CLI event lifecycle confidence contract baseline 추가
- [x] latest-event lifecycle detail baseline 추가
- [x] daemon-backed lifecycle detail follow-up 추가
- [x] daemon-backed lifecycle detail baseline 추가
- [x] low-confidence lifecycle event presentation baseline 추가
- [x] CLI human event detail baseline 추가
- [x] richer lifecycle coverage follow-up 추가
- [x] removed-event lifecycle detail follow-up 추가
- [x] observed inference keyword refinement baseline 추가
- [x] observed inference precedence guard baseline 추가
- [x] observed inference latest-line precedence baseline 추가
- [x] observed inference continuation-line guard baseline 추가
- [x] observed continuation summary baseline 추가
- [x] observed tool-read inference baseline 추가
- [x] tool-read event presentation baseline 추가
- [x] thinking-sleeping event presentation baseline 추가
- [x] humanized status label baseline 추가
- [x] attention subtitle humanization baseline 추가
- [x] notification fallback humanization baseline 추가
- [x] human attention breakdown wording baseline 추가
- [x] observed thinking phrase inference baseline 추가
- [x] observed status summary alignment baseline 추가
- [x] observed sleeping phrase inference baseline 추가
- [x] observed booting phrase inference baseline 추가
- [x] booting event presentation baseline 추가
- [x] observed idle phrase inference baseline 추가
- [x] observed disconnected phrase inference baseline 추가
- [x] observed error phrase refinement baseline 추가
- [x] long-silence notification baseline 추가
- [x] silence notification setting baseline 추가
- [x] silence settings decode coverage baseline 추가
- [x] observed reconnection phrase inference baseline 추가
- [x] observed reconnection event presentation baseline 추가
- [x] silence notification settings UI baseline 추가
- [x] silence notification summary copy baseline 추가
- [x] silence notification preview masking baseline 추가
- [x] observed explicit error summary refinement baseline 추가

### Current Slice Checklist

- [x] settings-aware notification filtering 추가
- [x] active agent 가 silence threshold 를 막 넘길 때 long-silence notification candidate 생성
- [x] long-silence notification 은 threshold crossing 때만 1회 생성되어 반복 spam 을 줄임
- [x] Swift tests로 long-silence notification baseline 보호
- [x] notification settings 에 silence toggle 이 추가됨
- [x] daemon/CLI/Swift 설정 경로가 silence toggle 을 round-trip 함
- [x] silence notification filtering 이 settings.notifications.silence 를 존중함
- [x] macOS menu bar settings section 에 silence toggle 이 노출됨
- [x] MenuBarViewModel settings update path 가 silence toggle UI 와 연결됨
- [x] Swift tests로 silence settings UI baseline 보호
- [x] long-silence notification body 가 lastUserVisibleSummary 를 활용해 더 구체적임
- [x] previewText 가 켜진 silence notification 에서 마지막 관측 요약을 보여줌
- [x] Swift tests로 silence notification summary copy baseline 보호
- [x] previewText=false 일 때 silence notification body 가 generic detail copy 로 마스킹됨
- [x] observed timeout/permission-denied 류 error phrase 가 더 구체적인 reason/summary 를 남김
- [x] explicit error phrase refinement 이 downstream observed lifecycle copy 품질을 높임
- [x] Go tests로 observed explicit error summary refinement baseline 보호
- [x] silence notification 도 기존 preview masking policy 를 동일하게 따름
- [x] Swift tests로 silence notification preview masking baseline 보호
- [x] Go/Swift tests로 silence notification setting baseline 보호
- [x] Swift payload decode 가 explicit silence=true 를 읽음
- [x] Swift payload decode 가 missing silence field 를 false 로 backfill 함
- [x] Swift tests로 silence settings decode coverage baseline 보호
- [x] observed 로그의 explicit reconnected/back online/connection restored line 이 `idle` 로 추론됨
- [x] disconnected phrase layer 와 reconnection phrase layer 가 분리됨
- [x] Go tests로 observed reconnection phrase inference baseline 보호
- [x] observed recovery text 기반 status_updated idle 이 `Idle` 대신 `Reconnected` 로 더 직접 표현됨
- [x] daemon hint 와 Swift presenter 가 observed reconnection wording에 대해 정렬됨
- [x] Go/Swift tests로 observed reconnection event presentation baseline 보호
- [x] preview-text masking behavior 추가
- [x] Swift tests로 settings-driven notification behavior 보호
- [x] daemon-backed message target resolution 재사용
- [x] CLI `ham ask` 구현
- [x] Go adapter sender/fallback 추가
- [x] Go tests/CLI smoke 로 message path 보호
- [x] quiet hours start/end schema 추가
- [x] CLI/UI 에서 quiet hours schedule 수정 가능하게 연결
- [x] current time 기반 quiet hours 판단 추가
- [x] Swift tests로 quiet hours window behavior 보호
- [x] daemon/open-target path 에 richer session identification data 추가
- [x] open/ask path 가 richer session identification 을 재사용하게 정리
- [x] Go/Swift tests로 richer session identification behavior 보호
- [x] iTerm session listing adapter baseline 추가
- [x] attach 가능한 session list surface 를 CLI/UI 쪽에 노출
- [x] Go/Swift tests로 attach picker/listing behavior 보호
- [x] attached session disconnect/termination heuristic 추가
- [x] daemon polling 또는 refresh path 에 disconnect detection 연결
- [x] Go/Swift tests로 attached disconnect behavior 보호
- [x] backend settings schema 에 non-notification section 추가
- [x] CLI/UI 에서 새 settings section 일부를 수정 가능하게 연결
- [x] Go/Swift tests로 broader settings section round-trip 보호
- [x] daemon event follow/read stream surface 추가
- [x] CLI 또는 Swift 가 polling 외의 follow path 를 사용할 수 있게 연결
- [x] Go/Swift tests로 live event follow baseline 보호
- [x] attached session metadata(cwd/title/activity) sync baseline 추가
- [x] daemon/UI 에 richer attached metadata 일부 노출
- [x] Go/Swift tests로 attached metadata sync baseline 보호
- [x] attached shell pid / command metadata 추가
- [x] daemon/UI 에 shell command / pid 일부 노출
- [x] Go/Swift tests로 shell-state fidelity baseline 보호
- [x] daemon event taxonomy 확장 또는 richer event summary 추가
- [x] UI 가 richer event semantics 를 더 직접 활용하게 연결
- [x] Go/Swift tests로 stronger event semantics baseline 보호
- [x] event-driven lane 의 refresh cadence / wakeup cost 줄이기
- [x] UI partial update 경로를 한 단계 더 넓히기
- [x] Go/Swift tests로 lower-latency UI update baseline 보호
- [x] richer event type 별 UI treatment 추가
- [x] activity feed / detail 이 richer event semantics 를 더 직접 활용
- [x] Go/Swift tests로 richer event-driven semantics baseline 보호
- [x] lower-latency visual cue/update baseline 추가
- [x] UI 가 중요 event semantics 를 더 빠르게 시각 반영
- [x] Go/Swift tests로 lower-latency visual update baseline 보호
- [x] activity feed semantics를 더 구조적으로 분류/집계
- [x] feed summary/visual grouping 을 더 직접 활용
- [x] Go/Swift tests로 stronger feed semantics baseline 보호
- [x] lifecycle transition event coverage 확장
- [x] runtime transition tests 강화
- [x] lifecycle summary wording 일관화
- [x] runtime state mutation paths 정리
- [x] coordinator-like transition boundaries 확장
- [x] Go tests로 runtime coordinator baseline 보호
- [x] runtime snapshot/list transition consistency 추가 개선
- [x] transition helper reuse 범위 확장
- [x] Go tests로 transition consistency baseline 보호
- [x] attached shell-state metadata freshness 개선
- [x] shell-state display/value prioritization 다듬기
- [x] Go/Swift tests로 attached shell-state follow-up 보호
- [x] runtime coordinator helper 적용 범위 추가 확장
- [x] read/write transition consistency 추가 정리
- [x] Go tests로 runtime coordinator follow-up 보호
- [x] lifecycle coverage 추가 확장
- [x] transition summary consistency 유지
- [x] Go tests로 lifecycle coverage follow-up 보호
- [x] attention row context/reason 강화
- [x] attention queue scanability 개선
- [x] Go/Swift tests로 attention queue follow-up 보호
- [x] `ham list` human output 에 confidence/reason 일부 노출
- [x] `ham status` human output 에 attention-oriented 요약 추가
- [x] Go tests로 CLI confidence/reason visibility 보호
- [x] `ham status` human output 에 urgent agent detail 일부 노출
- [x] `ham list` human output 의 attention-first scanability refinement
- [x] Go tests로 CLI attention detail baseline 보호
- [x] `ham status` human output 에 attention category breakdown 추가
- [x] attention breakdown 이 human-only formatting 임을 유지
- [x] Go tests로 CLI attention breakdown baseline 보호
- [x] `ham stop <agent>` baseline 추가
- [x] stop output 의 human/JSON path 정리
- [x] Go tests로 CLI stop baseline 보호
- [x] `ham logs <agent>` baseline 추가
- [x] logs output 의 human/JSON path 정리
- [x] Go tests로 CLI logs baseline 보호
- [x] `ham list` human output 에 summary line 추가
- [x] list summary 가 human-only formatting 임을 유지
- [x] Go tests로 CLI list summary baseline 보호
- [x] `ham doctor` baseline 추가
- [x] doctor output 의 human/JSON path 정리
- [x] Go tests로 CLI doctor baseline 보호
- [x] recent activity severity summary 추가
- [x] feed summary 가 severity-first scan path 를 직접 제공
- [x] Go/Swift tests로 severity-aware feed scanning baseline 보호
- [x] empty JSON event output 이 caller writer 를 존중하게 정리
- [x] event render helper regression test 추가
- [x] Go tests로 event JSON writer consistency baseline 보호
- [x] daemon snapshot 에 attention summary 추가
- [x] Swift snapshot decoding / summary surface 에 attention count 연결
- [x] Go/Swift tests로 daemon-backed attention summary baseline 보호
- [x] daemon attention breakdown 을 top summary UI 에 노출
- [x] Swift summary surface 에 daemon-backed breakdown 연결
- [x] Go/Swift tests로 daemon-backed attention breakdown UI baseline 보호
- [x] daemon snapshot 에 attention ordering 추가
- [x] Swift attention list 가 daemon ordering 을 우선 사용
- [x] Go/Swift tests로 daemon-backed attention ordering baseline 보호
- [x] daemon snapshot 에 attention subtitle 추가
- [x] Swift attention row 가 daemon subtitle 을 우선 사용
- [x] Go/Swift tests로 daemon-backed attention subtitle baseline 보호
- [x] `ham status --json` 에 attention summary fields 반영
- [x] status JSON 이 human summary wording 없이 richer attention contract 를 제공
- [x] Go tests로 CLI status attention contract baseline 보호
- [x] `ham status --json` 에 attention subtitle fields 반영
- [x] status JSON 이 daemon attention subtitle contract 를 함께 제공
- [x] Go tests로 CLI status attention subtitle contract baseline 보호
- [x] `ham ui` baseline 추가
- [x] menu bar executable resolution/launch plan 정리
- [x] Go tests로 CLI ui baseline 보호
- [x] `agent.status_updated` presentation 이 lifecycle state 를 더 직접 반영
- [x] registration presentation 이 mode context 를 더 직접 반영
- [x] Swift tests로 lifecycle-aware event presentation baseline 보호
- [x] daemon event payload 에 presentation hint 필드 추가
- [x] Swift presenter 가 daemon hint 를 우선 사용
- [x] Go/Swift tests로 daemon-backed event presentation hint baseline 보호
- [x] `ham events --json` 이 event presentation hint 필드를 유지
- [x] `ham logs --json` 도 same event hint contract 를 유지
- [x] Go tests로 CLI event presentation hint contract baseline 보호
- [x] daemon event payload 에 presentation summary 필드 추가
- [x] Swift recent event row 가 daemon summary hint 를 우선 사용
- [x] Go/Swift tests로 daemon-backed lifecycle summary baseline 보호
- [x] daemon event payload 에 lifecycle metadata 필드 추가
- [x] Swift presenter 가 summary-string inference 대신 daemon lifecycle metadata 를 우선 사용
- [x] Go/Swift tests로 daemon-backed lifecycle metadata baseline 보호
- [x] `ham events --json` 이 lifecycle metadata 필드를 유지
- [x] `ham logs --json` 도 same lifecycle metadata contract 를 유지
- [x] Go tests로 CLI event lifecycle metadata contract baseline 보호
- [x] daemon event payload 에 lifecycle reason 필드 추가
- [x] Swift payload decoding 에 lifecycle reason 연결
- [x] Go/Swift tests로 daemon-backed lifecycle reason baseline 보호
- [x] daemon event payload 에 lifecycle confidence 필드 추가
- [x] Swift payload decoding 에 lifecycle confidence 연결
- [x] Go/Swift tests로 daemon-backed lifecycle confidence baseline 보호
- [x] latestEventSummary 가 daemon-backed lifecycle detail fallback 을 사용
- [x] latest event banner 가 raw status-change 문장 대신 concise detail 을 보여줌
- [x] Swift tests로 latest-event lifecycle detail baseline 보호
- [x] event detail 이 lifecycle reason/confidence 를 더 직접 활용
- [x] lifecycle detail wording follow-up 보호
- [x] low-confidence lifecycle event label/emphasis 완화
- [x] Swift tests로 low-confidence lifecycle event presentation baseline 보호
- [x] human `ham events` / `ham logs` 가 concise lifecycle detail 을 우선 사용
- [x] low-confidence lifecycle detail wording 이 human event rows 에도 반영
- [x] Go tests로 CLI human event detail baseline 보호
- [x] remove event 가 lifecycle metadata 를 유지
- [x] downstream consumers 가 removed event context 를 잃지 않게 유지
- [x] Go tests로 richer lifecycle coverage follow-up 보호
- [x] remove event 가 generic `Tracking stopped.` 대신 lifecycle-aware detail 을 제공
- [x] human CLI / Swift surfaces 가 removed event 에서 더 구체적인 detail 을 보게 유지
- [x] Go tests로 removed-event lifecycle detail follow-up 보호
- [x] observed inference 가 waiting/error/done 신호를 더 구체적으로 해석
- [x] low-confidence observed wording 을 유지하면서 explicit signal confidence 를 끌어올림
- [x] Go tests로 observed inference keyword refinement baseline 보호
- [x] observed inference 가 `no error` / `0 failed` / `not completed` 같은 부정 문맥에 덜 흔들림
- [x] explicit signal precedence 가 generic substring 보다 우선함을 유지
- [x] Go tests로 observed inference precedence guard baseline 보호
- [x] mixed observed logs 에서 최신 line signal 이 오래된 line signal 보다 우선함
- [x] 오래된 error/waiting line 이 최신 done/continue line 을 덮지 않게 정리
- [x] Go tests로 observed inference latest-line precedence baseline 보호
- [x] signal이 없는 최신 continuation line 도 stale waiting/error fallback 을 억제
- [x] 최신 `continuing` / `still working` 류 line 이 thinking fallback 으로 연결됨
- [x] Go tests로 observed inference continuation-line guard baseline 보호
- [x] continuation line 이 generic time-only reason 대신 더 직접적인 thinking reason/summary 를 제공
- [x] observed thinking fallback 이 recent output vs continuation output 을 구분해 표현
- [x] Go tests로 observed continuation summary baseline 보호
- [x] observed 로그에서 tool-like line 이 `running_tool` 로 추론됨
- [x] observed 로그에서 reading/analyzing line 이 `reading` 으로 추론됨
- [x] Go tests로 observed tool-read inference baseline 보호
- [x] `agent.status_updated` 가 `running_tool` / `reading` 을 더 직접적인 event label 로 보여줌
- [x] Swift feed/presenter 와 daemon presentation hint 가 tool/read 상태에 대해 정렬됨
- [x] Go/Swift tests로 tool-read event presentation baseline 보호
- [x] `agent.status_updated` 가 `thinking` / `sleeping` 도 더 직접적인 event label 로 보여줌
- [x] Swift feed/presenter 와 daemon presentation hint 가 thinking/sleeping 상태에 대해 정렬됨
- [x] Go/Swift tests로 thinking-sleeping event presentation baseline 보호
- [x] human CLI/status display 가 underscore status 를 더 사람 친화적으로 보여줌
- [x] Swift detail/status display 가 same humanized status wording 을 사용함
- [x] Go/Swift tests로 humanized status label baseline 보호
- [x] daemon attention subtitle 도 same humanized status wording 을 사용함
- [x] Swift attention subtitle path 가 daemon-provided humanized wording 과 정렬됨
- [x] Go/Swift tests로 attention subtitle humanization baseline 보호
- [x] notification fallback body 도 raw underscore status 대신 humanized wording 을 사용함
- [x] summary 없는 notification candidate 에서 humanized status phrase 가 보임
- [x] Swift tests로 notification fallback humanization baseline 보호
- [x] human `ham status` attention breakdown line 이 raw `waiting_input` 대신 더 읽기 쉬운 wording 을 사용함
- [x] JSON attention breakdown contract 는 그대로 유지됨
- [x] Go tests로 human attention breakdown wording baseline 보호
- [x] observed 로그의 explicit thinking/planning/investigating line 이 generic freshness fallback 전에 `thinking` 으로 추론됨
- [x] continuation phrase 와 plain recent-output fallback 사이에 thinking-like phrase layer 가 생김
- [x] Go tests로 observed thinking phrase inference baseline 보호
- [x] observed `agent.status_updated` summary 가 raw reason 보다 user-visible summary 를 우선 사용함
- [x] observed event row/detail wording 이 observed inference summary 와 더 직접적으로 정렬됨
- [x] Go tests로 observed status summary alignment baseline 보호
- [x] observed 로그의 explicit idle/paused/sleeping line 이 staleness fallback 전에 `sleeping` 으로 추론됨
- [x] age-based sleeping fallback 과 explicit sleeping-like phrase layer 가 분리됨
- [x] Go tests로 observed sleeping phrase inference baseline 보호
- [x] observed 로그의 explicit starting/initializing/booting line 이 generic freshness fallback 전에 `booting` 으로 추론됨
- [x] explicit booting phrase layer 가 thinking-like phrase layer 보다 우선함
- [x] Go tests로 observed booting phrase inference baseline 보호
- [x] `agent.status_updated` 가 `booting` 도 직접적인 event label 로 보여줌
- [x] Swift feed/presenter 와 daemon presentation hint 가 booting 상태에 대해 정렬됨
- [x] Go/Swift tests로 booting event presentation baseline 보호
- [x] observed 로그의 explicit ready/idle/standing-by line 이 `sleeping` 대신 `idle` 로 추론됨
- [x] explicit idle phrase layer 가 sleeping/staleness fallback 과 분리됨
- [x] Go tests로 observed idle phrase inference baseline 보호
- [x] observed 로그의 explicit disconnected/offline/session-lost line 이 file-missing fallback 전에 `disconnected` 로 추론됨
- [x] explicit disconnected phrase layer 가 error/idle fallback 과 분리됨
- [x] Go tests로 observed disconnected phrase inference baseline 보호
- [x] observed 로그의 timeout/permission-denied 류 문구가 더 직접적으로 `error` 로 추론됨
- [x] explicit error phrase coverage 가 generic `error`/`failed` fallback 보다 풍부해짐
- [x] Go tests로 observed error phrase refinement baseline 보호
- [x] `ham events --json` 이 lifecycle_confidence 필드를 유지
- [x] `ham logs --json` 도 same lifecycle_confidence contract 를 유지
- [x] Go tests로 CLI event lifecycle confidence contract baseline 보호
- [x] Swift event detail 이 lifecycle reason/confidence fallback 을 직접 활용
- [x] low-confidence lifecycle detail wording 추가
- [x] Go/Swift tests로 daemon-backed lifecycle detail baseline 보호
- [x] `ham events --json` 이 lifecycle_reason 필드를 유지
- [x] `ham logs --json` 도 same lifecycle_reason contract 를 유지
- [x] Go tests로 CLI event lifecycle reason contract baseline 보호
- [x] `ham events --json` 이 presentation_summary 필드를 유지
- [x] `ham logs --json` 도 same presentation_summary contract 를 유지
- [x] Go tests로 CLI event presentation summary contract baseline 보호
- [x] daemon event payload 에 lifecycle metadata 필드 추가
- [x] Swift presenter 가 summary-string inference 대신 daemon lifecycle metadata 를 우선 사용
- [x] Go/Swift tests로 daemon-backed lifecycle metadata baseline 보호
- [x] latest-event / feed visuals 추가 polish
- [x] low-noise visual hierarchy refinement
- [x] Go/Swift tests로 visual polish baseline 보호
- [x] activity feed semantics를 더 구조적으로 분류/집계
- [x] feed summary/visual grouping 을 더 직접 활용
- [x] Go/Swift tests로 stronger feed semantics baseline 보호
- [x] settings schema 에 appearance 외 추가 section 확장
- [x] CLI/UI 에 새 settings section 일부를 더 노출
- [x] Go/Swift tests로 stronger settings section round-trip 보호
- [x] Swift view model 에 followEvents 기반 refresh lane 추가
- [x] menu bar 가 일부 event-driven update path 를 사용하게 연결
- [x] Go/Swift tests로 event-driven UI refresh baseline 보호
- [x] attached session cwd/activity metadata heuristic 추가
- [x] daemon/UI 에 cwd/activity metadata 일부 노출
- [x] Go/Swift tests로 cwd/activity metadata baseline 보호
- [x] follow event payload 로 partial UI update 범위 넓히기
- [x] polling fallback 대비 event-driven refresh cost 줄이기
- [x] Go/Swift tests로 higher-fidelity event-driven update baseline 보호
- [x] attached shell-state heuristic 정밀도 개선
- [x] daemon/UI 에 richer shell-state metadata 일부 노출
- [x] Go/Swift tests로 shell-state heuristic refinement baseline 보호
- [x] Go/Swift tests로 shell-state fidelity baseline 보호
- [x] observed status 변화 시 lifecycle event 기록 추가
- [x] activity feed 가 observed lifecycle 변화를 반영하게 연결
- [x] Go/Swift tests로 observed lifecycle event baseline 보호
- [x] daemon/agent schema 에 status reason 추가
- [x] observed/attached 상태 변화에 reason 채우기
- [x] Swift UI 에 reason 일부 노출
- [x] Go/Swift tests로 status reason baseline 보호
- [x] reason과 confidence를 함께 읽기 쉬운 형태로 UI refinement
- [x] mode별 low-confidence wording 정리
- [x] Go/Swift tests로 confidence/reason refinement baseline 보호
- [x] attention-required agent grouping/order baseline 추가
- [x] menu bar 에 attention queue/view 추가
- [x] Go/Swift tests로 attention queue baseline 보호
- [x] feed row ordering/priority refinement 추가
- [x] severity-aware feed scanning 개선
- [x] Go/Swift tests로 severity-aware feed ordering baseline 보호
- [x] Swift bootstrap build/test green 유지
- [x] Go tests green 유지

## Out of Scope For Current Slice

- [ ] pixel office 실제 렌더링 구현
- [ ] attached / observed mode의 완전 구현
- [ ] iTerm2 제어의 전체 자동화
- [ ] 고급 상태 추론 휴리스틱 완성
- [ ] production-grade notification policy 완성
- [ ] 디자인 polish / sprite asset 제작

---

## Execution Order

### Epic 1: Repository and Build Bootstrap
- [x] Swift package 생성
- [x] 모듈 경계 정의
- [x] 기본 테스트 타깃 생성
- [x] GitHub origin 연결 확인
- [x] hybrid repository layout로 재정렬 시작

#### Acceptance Criteria
- [x] 저장소 구조가 스펙 아키텍처와 대응된다
- [x] `swift build` / `swift test` 가능해야 한다
- [x] 원격 push 가능한 Git 워크트리여야 한다

### Epic 2: Managed Session Foundation
- [x] agent domain model 확정
- [x] local registry/persistence 초안 구현
- [x] `ham status`
- [x] `ham list`
- [x] `ham run` 최소 구현

#### Acceptance Criteria
- [x] managed agent 생성 및 조회 가능
- [x] CLI에서 현재 상태를 읽을 수 있음
- [x] 최소 persistence 경로가 정의됨

### Epic 3: Local Runtime and Event Flow
- [ ] runtime coordinator 구현
- [x] event log 구조 정의
- [ ] lifecycle transition 정리
- [x] runtime snapshot 제공

#### Acceptance Criteria
- [ ] runtime이 agent 상태를 일관되게 관리함
- [x] 이벤트 기반으로 상태 변경 추적 가능
- [ ] 테스트로 주요 전이 보호

### Epic 4: Menu Bar Baseline
- [x] macOS menu bar app target 생성
- [x] 기본 status indicator 구현
- [x] runtime snapshot 연결
- [x] 최소 팝오버 agent list 구현

#### Acceptance Criteria
- [x] 메뉴바에서 앱이 상주함
- [x] 현재 agent 상태 요약을 볼 수 있음
- [x] CLI/runtime과 상태 소스가 분리되지 않음

### Epic 5: Notifications
- [x] done / waiting_input / error 알림 정의
- [x] dedupe / mute 정책 초안
- [x] notification trigger 연동

#### Acceptance Criteria
- [x] 핵심 상태 알림이 동작함
- [x] 과도한 noisy progress 알림은 기본 비활성

### Epic 6: iTerm2 Integration
- [ ] 세션 식별 방식 결정
- [x] focus/open 연동
- [x] 선택적 message send
- [ ] 종료 감지 초안

#### Acceptance Criteria
- [x] managed session 재오픈 또는 focus 가능
- [x] 연동 실패 시 graceful fallback 존재

### Epic 7: Attached and Observed Modes
- [ ] attach flow 정의
- [ ] confidence 표시 확장
- [ ] observed mode 최소 추적 구현

#### Acceptance Criteria
- [ ] managed 외 세션도 추적 가능
- [ ] confidence와 mode가 UI/CLI에 노출됨

### Epic 8: Inference and Attention UX
- [ ] status inference engine 심화
- [ ] reason/confidence 계산
- [ ] attention queue/feed 설계

#### Acceptance Criteria
- [ ] 구조화 신호 없는 세션도 추론 가능
- [ ] 낮은 confidence는 UI에서 절제해 표시됨

### Epic 9: Pixel Office Experience
- [ ] room layout 구현
- [ ] animation state mapping
- [ ] agent detail interactions

#### Acceptance Criteria
- [ ] 상태가 시각적으로 구분됨
- [ ] 귀여움이 정보 전달을 가리지 않음

---

## Notes
- `spec.md`가 최종 목표 문서다.
- `roadmap.md`는 참고용이며 현재 범위를 제한하지 않는다.
- UI는 Swift, CLI/runtime은 Go로 분리하는 방향을 현재 기준 아키텍처로 본다.
- Ralph/autonomous 실행은 항상 가장 높은 우선순위의 미완료 green slice부터 이어간다.
- 각 slice 완료 시 `docs/progress.md`, `docs/assumptions.md`, 테스트 결과를 함께 갱신한다.
