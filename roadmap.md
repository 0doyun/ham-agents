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

> **방향 전환 (2026-03-28):** 범용 프로바이더 지원보다 **Claude Code 하나를 정확하게** 지원하는 것을 최우선으로 한다.
> 기존 PTY 출력 키워드 매칭 기반 상태 추론은 정확도가 낮아 (대부분 `thinking`으로 분류),
> Claude Code hooks 기반 사실 전달 방식으로 전환한다.

### Phase 1 — Claude Code 정확한 상태 추적
- `ham hook` 서브커맨드 구현 (Claude Code hook → 데몬 IPC)
- Claude Code hook 연동 (PreToolUse, PostToolUse, Stop)
- `ham setup` 커맨드 (Claude Code hooks 자동 설정)
- 서브에이전트 존재 감지 (Agent tool PreToolUse/PostToolUse)
- 기존 PTY 키워드 추론은 hook 미설정 시 fallback으로 유지

### Phase 2 — UI 재설계 + UX 개선
- 4존 그리드 → 단일 오피스 공간 재설계
- 가구 기반 영역 배치 (책상, 책장, 소파, 경고등)
- 미니 햄스터 (서브에이전트) 렌더링
- 도구별 스프라이트 분화 (Read → 책 읽기, Bash → 타이핑 등)
- 머리 위 상태 아이콘 (⚠️, ❓, ✅)
- 정확한 상태 기반 알림 개선

### Phase 3 — 멀티 프로바이더 확장
- Codex, Gemini CLI 등 전용 어댑터 추가
- 각 프로바이더 hook/출력 패턴에 맞는 상태 추론
- `ham setup codex`, `ham setup gemini` 등
- 범용 추론 엔진은 hook 미지원 프로바이더 fallback으로 유지

---

## 3. Phase별 상세 방향

### Phase 1 방향
목표:
**Claude Code의 상태를 100% 정확하게 추적하는 기반 확립**

범위:
- `ham hook` 서브커맨드 + IPC 커맨드 정의
- Claude Code hooks 연동 (PreToolUse, PostToolUse, Stop)
- `ham setup` 커맨드 (Claude Code 감지 → hooks 자동 설정)
- `~/.claude/settings.json` 안전한 merge 로직 (기존 설정 보존)
- 서브에이전트 존재 감지 (Agent tool hook으로 등록/해제)
- `ham doctor`에 hook 설정 상태 진단 추가
- 기존 PTY 키워드 추론은 hook 미설정 시 fallback 경로로 유지

### Phase 2 방향
목표:
**정확한 상태 데이터 위에 몰입감 있는 픽셀 오피스 경험 구축**

범위:
- 4존 그리드 → 단일 오피스 공간 재설계 (Swift UI)
- 가구 배치로 영역 암시 (책상, 책장, 소파, 경고등)
- 미니 햄스터 렌더링 (서브에이전트 시각화)
- 도구별 스프라이트 분화 (Read → 책 읽기, Bash → 타이핑)
- 머리 위 상태 아이콘 (⚠️ ❓ ✅)
- 정확한 상태 기반 알림 고도화

### Phase 3 방향
목표:
**Claude Code 이외 프로바이더로 확장**

범위:
- Codex, Gemini CLI 등 전용 어댑터
- 각 프로바이더별 hook/출력 패턴에 맞는 상태 추론
- `ham setup codex`, `ham setup gemini`
- 범용 추론 엔진은 fallback으로만 유지

---

## 4. Phase 3 이후 아이디어

- 햄스터 스킨
- 테마별 office
- team templates
- saved agent squads
- transcript adapters
- summary cards
- recommended attention queue
- local analytics dashboard
- Claude Code subagent 내부 상태 추적 (Claude Code에서 CLAUDE_SUBAGENT_ID 같은 기능 제공 시)

---

## 5. 우선순위 원칙

1. 귀여움보다 효용 우선
2. **Claude Code 정확한 상태 추적이 최우선** — 범용성보다 정확성
3. managed mode (hook 기반) 우선
4. 단일 오피스 UI는 hook 기반 안정 후
5. 다른 프로바이더 확장은 Claude Code 완성 후
6. 설정/프라이버시는 너무 늦추지 않음

---

## 6. 문서 우선순위 원칙

분석/구현 시 기본 원칙:
1. 현재 활성 작업 문서
2. tasks/worklog 문서
3. AGENTS.md
4. spec.md
5. roadmap.md

즉, **roadmap.md는 미래 방향 참고용**이다.

---

## 7. 방향 전환 기록

### 2026-03-28: Hook 기반 상태 추적으로 전환

**문제:** 기존 PTY 출력 키워드 매칭 방식으로는 Claude Code 상태를 정확히 판별할 수 없었다. `RecordManagedOutput`의 기본값이 `thinking`이라, 키워드가 매칭되지 않는 대부분의 출력이 `thinking`으로 분류됨. Claude Code는 구조화된 JSON을 stdout으로 보내지 않아 `provider_hints.go`도 실질적으로 동작하지 않음.

**결정:**
- Claude Code hooks (PreToolUse, PostToolUse, Stop) 기반으로 전환
- 범용 프로바이더 지원보다 Claude Code 하나를 정확하게 지원하는 것을 최우선
- 기존 버전 체계(v0.1/v0.2/v0.3/v1.0)를 Phase 1/2/3으로 재구성
- 4존 그리드 UI를 단일 오피스 공간으로 재설계 예정 (서브에이전트 수용, 공간 효율)

**영향:** spec.md §7, §9, §12, §15 업데이트. tasks.md에 Epic 18/19/20 추가.
