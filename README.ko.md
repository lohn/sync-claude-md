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
# 스테이징된 AGENTS.md만 처리 (기본값), git 인덱스와 대조
sync-claude-md sync

# 동기화된 파일을 자동으로 스테이징 (한 번에 성공)
sync-claude-md sync --stage

# 전체 저장소 스캔
sync-claude-md sync --all

# 드라이런: 변경 없이 확인
sync-claude-md check --all

# CLAUDE.md와 함께 GEMINI.md(@./AGENTS.md)도 동기화
sync-claude-md sync --gemini

# GEMINI.md만 동기화
sync-claude-md sync --gemini --no-claude

# 특정 파일 처리
sync-claude-md sync path/to/AGENTS.md another/AGENTS.md

# 스테이징되지 않은 변경이 있어도 대상을 덮어씀
sync-claude-md sync --force

# git에서 무시되는 대상 파일도 처리
sync-claude-md sync --no-ignore

# 쓰기가 발생하면 스테이징 성공 후에도 종료 코드 1로 종료
sync-claude-md sync --fail-on-change
```

명령 없이 `sync-claude-md`를 실행하면 도움말이 표시됩니다.

**대상 플래그:**

- `CLAUDE.md`는 기본적으로 동기화됩니다
- `--gemini` — 각 디렉터리에 `GEMINI.md`(`@./AGENTS.md`)도 동기화
- `--no-claude` — `CLAUDE.md`를 건너뜀 (`--gemini`와 함께 사용하면 `GEMINI.md`만 동기화)
- `--no-ignore` — git에서 무시되는 대상 파일도 처리 (기본적으로는 건너뜀)

**파일 인자를 지정하지 않으면**, 스테이징된 `AGENTS.md`만 처리됩니다 (git 훅
사용을 위한 기본 동작). 저장소 전체를 스캔하려면 `--all`을 전달하세요. git
저장소 밖에서는 "스테이징"이라는 개념이 없으므로 기본 동작도 전체 스캔으로
대체됩니다.

**`sync`는 세 가지를 보장합니다:**

- **손상 방지** — 스테이징되지 않은 변경이 있는 대상 파일을 덮어써 작업 중인 변경을
  잃게 만드는 것을 거부하고, 아무것도 쓰지 않은 채 종료 코드 `1`로 종료합니다.
  `--force`(`-f`)로 덮어쓸 수 있습니다.
- **git 저장소 밖에서는**, 새로 생성하는 경우를 포함해 아무것도 쓰지 않습니다.
  복구할 수 있는 git 히스토리가 없기 때문입니다. 종료 코드 `1`로 종료합니다.
  `--force`(`-f`)로 쓸 수 있습니다.
- **동기화**(git 저장소 내에서만) — `@AGENTS.md` 참조가 **스테이징되어 있어야**
  합니다. 그래야 동기화가 실제로 다음 커밋에 포함됩니다. 스테이징되지 않은
  경우(새로 생성되었지만 추적되지 않은 `CLAUDE.md` 포함) 종료 코드 `1`로
  종료되고 `git add`를 요청합니다. `--stage`(`-S`)를 전달하면 동기화된 파일을
  자동으로 스테이징하여 한 번에 성공합니다.

| 플래그             | 효과                                                           |
| ------------------ | -------------------------------------------------------------- |
| `--all`            | 스테이징된 파일만이 아닌 저장소 전체를 스캔                    |
| `--stage`, `-S`    | 동기화된 대상 파일을 `git add` (git 저장소 내에서만)           |
| `--force`, `-f`    | 스테이징되지 않은 변경이 있는 대상 또는 git 저장소 밖에서도 씀 |
| `--no-ignore`      | git에서 무시되는 대상 파일도 처리                              |
| `--fail-on-change` | 쓰기가 발생하면 스테이징 성공 후에도 종료 코드 `1`로 종료      |

> **참고**: `--stage`는 대상 파일 전체를 add하므로 부분 스테이징(`git add -p`)과는
> 잘 맞지 않습니다. 부분 스테이징 커밋에 의존한다면 `--stage`를 생략하고 수동으로
> 스테이징하세요.

**종료 코드:**

- `0` — 더 이상 할 일이 없는 상태: 모든 것이 최신이고 (git 저장소 내에서는)
  스테이징됨
- `1` — 손상 방지로 인한 차단, git 저장소 밖에서의 쓰기 거부, 스테이징되지 않은
  동기화 위반, 또는 (`check`의 경우) 드리프트가 있는 경우. `--fail-on-change`
  사용 시에는 쓰기가 발생한 것만으로도 종료 코드 `1`이 됩니다

### Pre-commit / [prek](https://github.com/pre-commit/prek)

`.pre-commit-config.yaml`에 추가:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

이 훅은 `sync-claude-md sync`를 실행하며, 기본적으로 동기화된 파일이
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
        entry: sync-claude-md sync
        language: system
        always_run: true
        pass_filenames: false
```

### [Husky](https://typicode.github.io/husky/)

자세한 설정 방법은 [docs/husky.md](docs/husky.md)를 참조하세요.

`.husky/pre-commit`의 간단한 예:

```bash
sync-claude-md sync --stage
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
