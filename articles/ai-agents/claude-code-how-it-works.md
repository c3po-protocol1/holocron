# Claude Code 내부 동작 방식 — 완전 해부

> Agentic Loop, Context Loading, Tool Execution, Query Engine까지 소스 기반 분석

**출처:** [How Claude Code works (Mintlify)](https://www.mintlify.com/VineeTagarwaL-code/claude-code/concepts/how-it-works)
**날짜:** 2026-04-03
**태그:** `#ClaudeCode` `#아키텍처` `#에이전트루프` `#컨텍스트` `#도구실행`

---

## 개요

Claude Code는 터미널에서 실행되는 **연속 에이전트 루프(Agentic Loop)** 기반 코딩 에이전트다. 사용자 요청을 받으면 추론 → 도구 호출 → 결과 관찰 → 반복의 사이클을 돌며, 작업이 완료되거나 사용자 입력이 필요할 때까지 자동으로 순환한다.

핵심적으로 **원격 실행 서버가 없다.** 모든 것이 로컬 터미널 프로세스에서 동작하며, 파일/셸/인증정보가 머신을 떠나지 않는다 (WebFetch, WebSearch, MCP 서버 등 명시적 네트워크 도구 제외).

---

## 1. The Agentic Loop — 6단계 핵심 사이클

모든 상호작용은 동일한 사이클을 따른다:

### Step 1: 사용자 메시지 전송

터미널 입력(Interactive 모드) 또는 `--print`/stdin(Non-interactive/headless 모드)으로 메시지 전달. 메시지가 대화 히스토리에 추가된다.

### Step 2: 컨텍스트 조립

API 호출 전에 Claude Code가 시스템 프롬프트를 구성한다:

- 현재 날짜
- Git 상태 (브랜치, 최근 커밋, working tree 상태)
- CLAUDE.md 메모리 파일
- 사용 가능한 도구 목록

이 컨텍스트는 `context.ts`에서 처리되며, **대화 내내 memoize**된다 (매 턴마다 재계산하지 않음).

### Step 3: Claude 추론 + 도구 선택

조립된 대화가 Anthropic API로 전송된다. 모델이 추론 후 하나 이상의 `tool_use` 블록을 생성한다. 각 블록은 **도구 이름 + 구조화된 JSON 입력**을 포함.

### Step 4: 권한 체크 (Permission Check)

각 도구 호출 실행 전에 현재 permission mode와 allow/deny 규칙을 평가한다. 결과에 따라:

- 자동 승인 (allow)
- 사용자 확인 요청 (ask)
- 차단 (deny)

### Step 5: 도구 실행 + 결과 반환

승인된 도구가 실행되고, 결과(파일 내용, 명령 출력, 검색 결과 등)가 `tool_result` 블록으로 대화에 추가된다.

### Step 6: 루프 반복

모델이 도구 결과를 받고, 추가 도구를 호출하거나 최종 텍스트 응답을 생성한다. **tool call이 더 이상 없을 때까지 반복.**

---

## 2. Context Loading — 세션 시작 시 주입되는 것들

매 대화 시작 시 **2개의 컨텍스트 블록**이 모든 API 호출에 prepend된다:

### System Context (`getSystemContext()`)

- **Git 상태** — 현재 브랜치, default/main 브랜치, git username, `git status --short` 출력 (2,000자 초과 시 truncate), 최근 5개 커밋 (`git log --oneline`)
- **Cache-breaking injection** — 디버깅 시 서버 사이드 프롬프트 캐시를 무효화하는 임시 문자열
- `CLAUDE_CODE_REMOTE=1` 설정 시 git 상태 수집을 스킵

### User Context (`getUserContext()`)

- **CLAUDE.md 메모리** — 4단계 계층 탐색: managed → user → project → local. `CLAUDE_CODE_DISABLE_CLAUDE_MDS=1`로 비활성화 가능
- **현재 날짜** — `Today's date is YYYY-MM-DD` 형태로 주입하여 모델이 항상 날짜를 인식

### Memoization 전략

두 컨텍스트 모두 `lodash/memoize`로 **대화 내내 캐싱**된다. 즉 첫 번째 API 호출 시 한 번 계산되고, 이후 턴에서는 캐시된 값을 재사용. `setSystemPromptInjection()`을 호출하면 즉시 캐시가 무효화된다.

---

## 3. Tool Execution Model — 권한 시스템

Claude Code는 기본적으로 도구 호출을 **자율적으로 실행하지 않는다.** 모든 도구에 `checkPermissions` 메서드가 있고, 3가지 결과를 반환한다:

| 결과 | 동작 |
|------|------|
| `allow` | 즉시 실행, 결과를 대화에 추가 |
| `ask` | 사용자에게 확인 다이얼로그 표시 후 대기 |
| `deny` | 거부, 모델에 에러 결과 반환 |

### Permission Mode별 동작

- **`bypassPermissions`** — 모든 권한 체크 스킵. 완전 자동 실행 (CI/CD, 스크립트 용도)
- **`acceptEdits`** — 파일 편집 도구(`Edit`, `Write`)는 자동 승인, bash 명령은 여전히 확인 필요
- **기본 모드** — 모든 쓰기 작업에 사용자 확인 필요

**예외:** Read-only 도구 (`Read`, `Glob`, `Grep`)는 **모든 모드에서 자동 승인**된다. 읽기만 하므로 안전하다고 판단.

---

## 4. Interactive vs Non-interactive 모드

### Interactive (REPL) 모드

기본 모드. **React/Ink 기반 터미널 UI**를 렌더링한다:

- 스트리밍 출력 (토큰이 생성되는 대로 표시)
- 도구 사용 확인 다이얼로그
- 스피너 애니메이션
- 세션 내 메시지 유지

### Non-interactive (Print) 모드

`--print` 플래그 또는 stdin 파이프로 활성화:

- UI 없음
- stdout으로 출력
- CI/CD 파이프라인이나 스크립트 자동화에 적합
- 원샷 작업용

---

## 5. Sub-agents (Task Tool)

Claude는 `Task` 도구(`AgentTool`)를 통해 **서브 에이전트를 생성**할 수 있다:

- 각 서브 에이전트는 **독립된 대화 컨텍스트**에서 자체 agentic loop 실행
- 선택적으로 **제한된 도구 세트** 지정 가능 (예: 읽기 전용 도구만)
- 로컬(인프로세스) 또는 원격 컴퓨트에서 실행
- 완료 시 결과를 부모 에이전트에 반환

이를 통해 복잡한 작업을 **분할 정복** 방식으로 처리할 수 있다. 부모 에이전트는 전체 전략을, 서브 에이전트는 개별 파일이나 모듈 단위의 작업을 담당.

---

## 6. Conversation Storage & Resumption

### 저장 방식

대화는 **JSON 트랜스크립트 파일**로 `~/.claude/` 디렉토리에 저장된다. 각 대화에 고유 세션 ID가 부여된다.

### 재개 방법

- `--resume <session-id>` — 특정 세션 직접 재개
- `--resume` (인자 없이) — 이전 세션 목록에서 선택

### 재개 시 동작

1. **전체 메시지 히스토리**를 디스크에서 로드
2. **메모리 파일 재탐색** — 최초 대화 시점과 달라질 수 있음 (파일이 수정/추가/삭제되었을 수 있으므로)
3. **Permission mode**는 설정된 기본값으로 리셋 (세션에 저장된 경우 제외)

### Compaction — 긴 대화 관리

긴 대화는 주기적으로 **오래된 메시지를 요약(compact)**하여 컨텍스트 윈도우를 관리한다:

- 원본 트랜스크립트는 **디스크에 항상 보존**
- Compaction은 **API에 보내는 것만** 영향
- 즉, 대화 이력의 무결성은 유지하면서 API 호출 비용과 컨텍스트 한계를 관리

이것은 에이전트 설계에서 가장 중요한 메커니즘 중 하나다. 무한히 긴 대화를 어떻게 유한한 컨텍스트 윈도우 안에서 처리할 것인가의 문제.

---

## 7. Query Engine — 턴 단위 실행 엔진

각 "턴"은 `query.ts`의 **query** 호출로 구동된다. 현재 메시지 리스트를 Anthropic API에 보내고 스트리밍 응답을 받는다.

Query Engine이 처리하는 것들:

1. **스트리밍 토큰 출력** — 실시간으로 터미널에 표시
2. **`tool_use` 블록 디스패치** — 적절한 도구 핸들러로 라우팅
3. **턴당 예산 적용** — 토큰 수와 도구 호출 횟수에 제한
4. **도구 결과 수집** — 다음 모델 호출 전에 대화에 추가
5. **Compaction 트리거** — 컨텍스트 윈도우가 가득 차면 자동 실행

### 결과 크기 제한 (`maxResultSizeChars`)

각 도구에 `maxResultSizeChars` 속성이 있다. 도구 결과가 이 한도를 초과하면:

1. 전체 내용을 **임시 파일에 저장**
2. 모델에는 **미리보기 + 파일 경로**만 전달
3. 컨텍스트 윈도우 오버플로우 방지

이를 통해 대용량 파일을 `Read`하더라도 컨텍스트가 폭발하지 않는다.

---

## 아키텍처 다이어그램 (개념)

```
User Input
    │
    ▼
┌─────────────────────────────────┐
│         Context Assembly         │
│  (git status + CLAUDE.md + date) │
│         [memoized]               │
└──────────────┬──────────────────┘
               │
               ▼
┌─────────────────────────────────┐
│        Anthropic API Call        │
│     (query.ts → streaming)       │
└──────────────┬──────────────────┘
               │
          ┌────┴────┐
          │ text?   │ tool_use?
          │         │
          ▼         ▼
     [Response]  ┌──────────────┐
                 │ Permission   │
                 │    Check     │
                 └──────┬───────┘
                   allow│ask│deny
                        │
                        ▼
                 ┌──────────────┐
                 │ Tool Execute │
                 │  + Result    │
                 └──────┬───────┘
                        │
                        ▼
                 [Append to conversation]
                        │
                        ▼
                 [Loop continues ↑]
```

---

## 핵심 시사점

### 1. 완전 로컬 실행 아키텍처

원격 서버 없이 터미널에서 모든 것이 돌아간다. 보안 관점에서 강점이며, 동시에 사용자 머신의 리소스에 의존한다는 제약도 있다.

### 2. Compaction이 에이전트의 생명선

긴 대화에서 컨텍스트 윈도우를 어떻게 관리하느냐가 에이전트 설계의 핵심 과제다. Claude Code의 접근법:
- 원본은 항상 보존 (감사 추적 가능)
- API에 보내는 것만 요약
- 도구 결과 크기도 `maxResultSizeChars`로 제한

### 3. Permission 3단계 모델

allow/ask/deny 패턴은 **AI 에이전트의 안전성과 자율성 사이의 균형**을 잡는 사실상의 표준이 되어가고 있다. Read-only는 항상 허용, 쓰기는 모드에 따라 — 이 단순하지만 효과적인 구분이 핵심.

### 4. Sub-agent 패턴의 가능성

Task tool을 통한 서브 에이전트 생성은 **에이전트 오케스트레이션**의 기본 패턴이다. 격리된 컨텍스트 + 제한된 도구 세트로 안전하게 병렬 작업 가능. 이것이 multi-agent 시스템의 출발점.
