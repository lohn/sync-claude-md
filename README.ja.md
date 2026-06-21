# sync-claude-md

[![CI](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml/badge.svg)](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lohn/sync-claude-md)](https://goreportcard.com/report/github.com/lohn/sync-claude-md)
[![npm](https://img.shields.io/npm/v/sync-claude-md.svg)](https://www.npmjs.com/package/sync-claude-md)
[![PyPI](https://img.shields.io/pypi/v/sync-claude-md.svg)](https://pypi.org/project/sync-claude-md/)

> [English](README.md) | **日本語** | [한국어](README.ko.md)

マルチエージェント開発ワークフローにおいて、`CLAUDE.md` と `AGENTS.md` を自動的に同期します。

## なぜ必要か

AI コーディングエージェントは指示ファイルの読み込み先がそれぞれ異なります。Claude
Code は `CLAUDE.md` を、その他の多くのエージェント（GitHub Copilot、Cursor など）は
`AGENTS.md` を読みます。両方を手動で管理するのは面倒で、ミスが起きやすいです。

`sync-claude-md` はこれらを自動的に同期します。`AGENTS.md` が見つかった各ディレクトリに、
`@AGENTS.md` 参照を含む同階層の `CLAUDE.md` を用意します（新規作成、または既存ファイルの
内容を保持したまま参照を追加）。`AGENTS.md` が削除されたら参照を除去し、空になった場合は
ファイルごと削除します。`--gemini` を付けると `GEMINI.md`（`@./AGENTS.md`）にも同様の処理を行います。

**pre-commit hook** またはスタンドアロン CLI として動作します。

## インストール

### npm を使用（Node.js プロジェクト）

```bash
npm install --save-dev sync-claude-md
npx sync-claude-md --help
```

### PyPI を使用（Python プロジェクト）

```bash
pip install sync-claude-md
sync-claude-md --help
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
sync-claude-md sync           # ステージング済み AGENTS.md のみ処理（デフォルト）、git インデックスと照合
sync-claude-md sync --all     # リポジトリ全体をスキャン
sync-claude-md sync --stage   # 同期したファイルも自動でステージ（1 パスで成功）
sync-claude-md check --all    # ドライラン：書き込まずにドリフトを確認
sync-claude-md sync --gemini  # GEMINI.md（@./AGENTS.md）も同期
```

コマンドなしで `sync-claude-md` を実行するとヘルプが表示されます。ファイル引数を
指定しない場合、ステージ済みの `AGENTS.md` のみが処理されます（git フックでの利用を
想定したデフォルト動作）。git リポジトリ外では「ステージ済み」という概念がないため、
デフォルトでも全体スキャンにフォールバックします。

**フラグ：**

| フラグ             | 効果                                                                   |
| ------------------ | ---------------------------------------------------------------------- |
| `--all`            | ステージ済みファイルのみではなく、リポジトリ全体をスキャン             |
| `--stage`, `-S`    | 同期した対象ファイルを `git add`（git リポジトリ内でのみ）             |
| `--force`, `-f`    | 未ステージの変更がある対象、または git リポジトリ外でも書き込む        |
| `--gemini`         | 各ディレクトリに `GEMINI.md`（`@./AGENTS.md`）も同期                   |
| `--no-claude`      | `CLAUDE.md` をスキップ（`--gemini` と併用すると `GEMINI.md` のみ同期） |
| `--no-ignore`      | git で ignore された対象ファイルも処理（デフォルトではスキップ）       |
| `--fail-on-change` | 書き込みが発生した場合、ステージ成功後でも終了コード `1` で終了        |

`--all` やステージ検出に頼らず、特定のファイルを直接渡すこともできます。例：
`sync-claude-md sync path/to/AGENTS.md another/AGENTS.md`

**`sync` は 3 つの安全保証を行います：**

- **破壊防止** — 未ステージの変更がある対象ファイルを上書きして作業中の変更を
  失わせることを拒否し、書き込みをせずに終了コード `1` で終了します。`--force`
  を付けると上書きできます。
- **git リポジトリ外では書き込まない** — 復元元となる git の履歴が存在しないため、
  新規作成も含めて一切書き込みを拒否し、終了コード `1` で終了します。`--force`
  を付けると書き込めます。
- **同期**（git リポジトリ内でのみ） — `@AGENTS.md` 参照が**ステージされている**
  必要があります。これにより同期が実際に次のコミットに含まれます。ステージされて
  いない場合（新規作成されたが未追跡の `CLAUDE.md` を含む）、終了コード `1` で
  終了し、`git add` を促します。`--stage` を付けると同期したファイルを自動で
  ステージし、1 パスで成功します。

> **注意**: `--stage` は対象ファイル全体を add するため、部分ステージ
> （`git add -p`）とは相性がよくありません。部分ステージのコミットに依存する場合は
> `--stage` を付けず手動でステージしてください。

**終了コード：** やるべきことが何もない場合（すべて最新で、git リポジトリ内では
ステージ済み）は `0`。上記の保証に違反した場合、`check` でドリフトを検出した場合、
または `--fail-on-change` 指定時に書き込みが発生した場合は `1`。

### Pre-commit / [prek](https://github.com/pre-commit/prek)

`.pre-commit-config.yaml` に追加：

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

このフックは `sync-claude-md sync` を実行し、デフォルトでは同期したファイルが
ステージされていないときにコミットを失敗させ、再ステージとコミットを促します。
同期したファイルを自動でステージするには `args: ['--stage']` を追加します：

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
        args: ["--stage"]
```

または、事前にインストールしたバイナリを `repo: local` で使用：

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

詳細な設定手順は [docs/husky.md](docs/husky.md) を参照してください。

`.husky/pre-commit` の簡単な例：

```bash
sync-claude-md sync --stage
```

## 仕組み

`AGENTS.md` が見つかった各ディレクトリに、**同階層**の `CLAUDE.md` を作成します（内容は以下のみ）：

```markdown
@AGENTS.md
```

`@path/to/file` 構文は `CLAUDE.md` ファイル自身の場所から相対的に解決されるため
（CWD ではなく）、`@AGENTS.md` は常に正しいファイルを指します。`--gemini` を指定すると、
Gemini のインポート構文 `@./AGENTS.md` を使って同様に `GEMINI.md` を作成します。

冪等性と安全性：

- ファイル内のどこにも参照がない場合のみ（先頭に）追加し、既存のコンテンツをすべて保持
- `AGENTS.md` 削除時に自動で参照を除去し、結果として空になった場合はファイルも削除
- 対象ファイルが 10 MiB を超える場合は読み込みを拒否し、一度にメモリへ読み込む量を
  頭打ちします。通常サイズのファイルには影響しません

## ライセンス

MIT © [lohn](https://github.com/lohn)
