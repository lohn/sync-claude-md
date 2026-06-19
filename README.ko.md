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
```

**대상 플래그:**

- `CLAUDE.md`는 기본적으로 동기화됩니다
- `--gemini` — 각 디렉터리에 `GEMINI.md`(`@./AGENTS.md`)도 동기화
- `--no-claude` — `CLAUDE.md`를 건너뜀 (`--gemini`와 함께 사용하면 `GEMINI.md`만 동기화)

**종료 코드:**

- `0` — 모든 것이 최신 상태
- `1` — 변경이 수행됨 (또는 `--check` 모드에서 변경 필요)

> **부분 실패에 대해**: 여러 파일 처리 중 오류가 발생하면,
> 오류 발생 전에 처리된 파일은 변경된 상태로 남을 수 있습니다.
> 이 도구는 변경 사항을 롤백하지 않습니다. 오류 보고 시 작업 디렉터리를 확인하세요.

### Pre-commit / [prek](https://github.com/pre-commit/prek)

`.pre-commit-config.yaml`에 추가:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

또는 사전 설치된 바이너리를 `repo: local`로 사용:

```yaml
repos:
  - repo: local
    hooks:
      - id: sync-claude-md
        name: Sync CLAUDE.md
        entry: sync-claude-md
        language: system
        files: AGENTS\.md$
        pass_filenames: true
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
