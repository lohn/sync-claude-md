# sync-claude-md

[![CI](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml/badge.svg)](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lohn/sync-claude-md)](https://goreportcard.com/report/github.com/lohn/sync-claude-md)

> [English](README.md) | [日本語](README.ja.md) | **한국어**

멀티 에이전트 개발 워크플로우에서 `CLAUDE.md`와 `AGENTS.md`를 자동으로 동기화합니다.

## 문제점

다른 AI 코딩 에이전트는 다른 지시 파일을 사용합니다:

- **Claude Code**는 `CLAUDE.md`를 읽습니다
- **다른 에이전트**(GitHub Copilot, Cursor 등)는 `AGENTS.md`를 읽습니다

여러 개발자가 있는 환경에서 두 파일을 수동으로 관리하는 것은 번거롭고 실수하기 쉽습니다.

## 해결책

이 도구는 자동으로 다음을 수행합니다:

1. `AGENTS.md`가 존재할 때 `@AGENTS.md` 참조와 함께 `CLAUDE.md` **생성**
2. 기존 `CLAUDE.md`의 맨 위에 참조 **추가**
3. `AGENTS.md`가 삭제되면 참조 **정리**
4. `CLAUDE.md`의 기존 콘텐츠 **보존**

**pre-commit hook** 또는 독립형 CLI로 작동합니다.

## 설치

### npm을 통해 (Node.js 프로젝트)

```bash
npm install --save-dev sync-claude-md
npx sync-claude-md --help
```

### GitHub Releases를 통해

[Releases](https://github.com/lohn/sync-claude-md/releases)에서 플랫폼용 바이너리를 다운로드하세요.

### Go를 통해

```bash
go install github.com/lohn/sync-claude-md/cmd/sync-claude-md@latest
```

## 사용법

### CLI

```bash
# 스테이징된 AGENTS.md만 처리 (기본값)
sync-claude-md

# 전체 저장소 스캔
sync-claude-md --all

# 드라이런: 변경 없이 확인
sync-claude-md --check

# CLAUDE.md와 함께 GEMINI.md(@./AGENTS.md)도 동기화
sync-claude-md --gemini

# GEMINI.md만 동기화
sync-claude-md --gemini --no-claude

# 특정 파일 처리
sync-claude-md path/to/AGENTS.md another/AGENTS.md

# pre-commit 모드: 스테이징된 AGENTS.md를 동기화하고 git 인덱스와 대조
sync-claude-md pre-commit

# pre-commit 모드: 동기화된 파일을 자동으로 스테이징 (한 번에 성공)
sync-claude-md pre-commit --stage
```

**대상 플래그:**

- `CLAUDE.md`는 기본적으로 동기화됩니다
- `--gemini` — 각 디렉터리에 `GEMINI.md`(`@./AGENTS.md`)도 동기화
- `--no-claude` — `CLAUDE.md`를 건너뜀 (`--gemini`와 함께 사용하면 `GEMINI.md`만 동기화)

**종료 코드:**

- `0` — 모든 것이 최신 상태
- `1` — 변경이 수행됨 (또는 `--check` 모드에서 변경 필요)

### `pre-commit` 하위 명령

`sync-claude-md pre-commit`은 스테이징된 `AGENTS.md`를 동기화한 다음, 결과를
작업 트리가 아닌 **git 인덱스**와 대조합니다. 두 가지를 보장합니다:

- **동기화** — `@AGENTS.md` 참조가 **스테이징되어 있어야** 합니다. 그래야 동기화가
  실제로 커밋에 포함됩니다. 스테이징되지 않은 경우(새로 생성되었지만 추적되지 않은
  `CLAUDE.md` 포함) 커밋이 종료 코드 `1`로 중단되고 `git add`를 요청합니다.
  `--stage`를 전달하면 동기화된 파일을 자동으로 스테이징하여 한 번에 성공합니다
  (종료 코드 `0`).
- **손상 방지** — 스테이징되지 않은 변경이 있는 대상 파일을 덮어써 작업 중인 변경을
  잃게 만드는 것을 거부하고, 아무것도 쓰지 않은 채 `1`로 종료합니다. `--force`
  (`-f`)로 재정의할 수 있습니다. (pre-commit/prek로 실행하면 프레임워크가
  스테이징되지 않은 변경을 먼저 stash하므로 거의 발생하지 않으며, 주로 수동 실행을
  보호합니다.)

| 플래그                     | 효과                                                  |
| -------------------------- | ----------------------------------------------------- |
| `--stage`, `-S`            | 동기화된 대상 파일을 `git add`. 한 번에 종료 코드 `0` |
| `--force`, `-f`            | 스테이징되지 않은 변경이 있어도 대상을 덮어씀         |
| `--gemini` / `--no-claude` | 최상위 명령과 동일한 대상 선택                        |

> **참고**: `--stage`는 대상 파일 전체를 add하므로 부분 스테이징(`git add -p`)과는
> 잘 맞지 않습니다. 부분 스테이징 커밋에 의존한다면 `--stage`를 생략하고 수동으로
> 스테이징하세요.

### Pre-commit / [prek](https://github.com/pre-commit/prek)

`.pre-commit-config.yaml`에 추가:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

이 훅은 `sync-claude-md pre-commit`을 실행하며, 기본적으로 동기화된 파일이
스테이징되지 않았을 때 커밋을 실패시켜 다시 스테이징하고 커밋하도록 합니다.
동기화된 파일을 자동으로 스테이징하려면 `args: ['--stage']`를 추가하세요:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
        args: ["--stage"]
```

또는 사전 설치된 바이너리를 `repo: local`로 사용:

```yaml
repos:
  - repo: local
    hooks:
      - id: sync-claude-md
        name: Sync CLAUDE.md
        entry: sync-claude-md pre-commit
        language: system
        always_run: true
        pass_filenames: false
```

### [Husky](https://typicode.github.io/husky/)

자세한 설정 방법은 [docs/husky.md](docs/husky.md)를 참조하세요.

`.husky/pre-commit`의 간단한 예:

```bash
STAGED_AGENTS=$(git diff --cached --name-only --diff-filter=ACMR | grep -E 'AGENTS\.md$' || true)
if [ -n "$STAGED_AGENTS" ]; then
  echo "$STAGED_AGENTS" | xargs sync-claude-md
fi
```

## 작동 원리

### 각 디렉터리 방식

`AGENTS.md`가 발견된 각 디렉터리에 **동일 계층**의 `CLAUDE.md`를 생성합니다:

```markdown
@AGENTS.md
```

`@path/to/file` 구문은 `CLAUDE.md` 파일 자체의 위치에서 상대적으로 해석됩니다 (CWD가 아님). 따라서 `@AGENTS.md`는 항상 올바른 파일을 가리킵니다.

`--gemini`를 지정하면 Gemini의 임포트 구문 `@./AGENTS.md`를 사용하여 동일한 방식으로 `GEMINI.md`를 생성합니다.

### 멱등성과 안전성

- 파일 어디에도 참조가 없을 때만 (맨 위에) 추가
- 기존 콘텐츠를 모두 보존
- `AGENTS.md` 삭제 시 자동으로 참조 제거
- 정리 후 빈 지시 파일 삭제

## 라이선스

MIT © [lohn](https://github.com/lohn)
