# sync-claude-md

[![CI](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml/badge.svg)](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lohn/sync-claude-md)](https://goreportcard.com/report/github.com/lohn/sync-claude-md)

> [English](README.md) | **日本語** | [한국어](README.ko.md)

マルチエージェント開発ワークフローにおいて、`CLAUDE.md` と `AGENTS.md` を自動的に同期します。

## 課題

異なる AI コーディングエージェントは異なる指示ファイルを使用します：

- **Claude Code** は `CLAUDE.md` を読みます
- **その他のエージェント**（GitHub Copilot、Cursor など）は `AGENTS.md` を読みます

複数の開発者がいる環境では、両方のファイルを手動で管理するのは面倒で、ミスが起きやすいです。

## 解決策

このツールは自動的に以下を行います：

1. `AGENTS.md` が存在する場合、`CLAUDE.md` を `@AGENTS.md` 参照付きで**作成**
2. 既存の `CLAUDE.md` の先頭に参照を**追加**
3. `AGENTS.md` が削除された場合、参照を**クリーンアップ**
4. `CLAUDE.md` の既存コンテンツを**保持**

**pre-commit hook** またはスタンドアロン CLI として動作します。

## インストール

### npm を使用（Node.js プロジェクト）

```bash
npm install --save-dev sync-claude-md
npx sync-claude-md --help
```

### GitHub Releases から

[Releases](https://github.com/lohn/sync-claude-md/releases) からお使いのプラットフォーム用のバイナリをダウンロードしてください。

### Go を使用

```bash
go install github.com/lohn/sync-claude-md/cmd/sync-claude-md@latest
```

## 使い方

### CLI

```bash
# ステージングされた AGENTS.md のみ処理（デフォルト）
sync-claude-md

# リポジトリ全体をスキャン
sync-claude-md --all

# ドライラン：変更を加えずに確認
sync-claude-md --check

# CLAUDE.md に加えて GEMINI.md（@./AGENTS.md）も同期
sync-claude-md --gemini

# GEMINI.md のみを同期
sync-claude-md --gemini --no-claude

# 特定のファイルを処理
sync-claude-md path/to/AGENTS.md another/AGENTS.md
```

**対象フラグ：**

- `CLAUDE.md` はデフォルトで同期されます
- `--gemini` — 各ディレクトリに `GEMINI.md`（`@./AGENTS.md`）も同期
- `--no-claude` — `CLAUDE.md` をスキップ（`--gemini` と併用すると `GEMINI.md` のみ同期）

**終了コード：**

- `0` — すべて最新
- `1` — 変更が行われた（または `--check` モードで変更が必要）

> **部分失敗について**: 複数ファイルの処理中にエラーが発生した場合、
> エラーが発生する前に処理されたファイルは変更されたまま残ることがあります。
> 本ツールは変更をロールバックしません。エラー報告時は作業ディレクトリを確認してください。

### Pre-commit / [prek](https://github.com/pre-commit/prek)

`.pre-commit-config.yaml` に追加：

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

または、事前にインストールしたバイナリを `repo: local` で使用：

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

詳細な設定手順は [docs/husky.md](docs/husky.md) を参照してください。

`.husky/pre-commit` の簡単な例：

```bash
STAGED_AGENTS=$(git diff --cached --name-only --diff-filter=ACMR | grep -E 'AGENTS\.md$' || true)
if [ -n "$STAGED_AGENTS" ]; then
  echo "$STAGED_AGENTS" | xargs sync-claude-md
fi
```

## 仕組み

### 各ディレクトリ方式

`AGENTS.md` が見つかった各ディレクトリに、**同階層**の `CLAUDE.md` を作成します：

```markdown
@AGENTS.md
```

`@path/to/file` 構文は、`CLAUDE.md` ファイル自身の場所から相対的に解決されるため（CWD ではなく）、`@AGENTS.md` は常に正しいファイルを指します。

`--gemini` を指定すると、Gemini のインポート構文 `@./AGENTS.md` を使って同様に `GEMINI.md` を作成します。

### 冪等性と安全性

- 参照が先頭にない場合のみ追加
- 既存のコンテンツをすべて保持
- `AGENTS.md` 削除時に自動で参照を除去
- クリーンアップ後に空になった指示ファイルを削除

## ライセンス

MIT © [lohn](https://github.com/lohn)
