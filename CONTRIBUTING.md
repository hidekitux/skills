# Contributing

## Commit messages

すべてのコミットと Pull Request のタイトルに Conventional Commits を使用します。

```text
type(scope): summary
```

利用できる `type` は次のとおりです。

`build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `test`

`scope` は任意、`summary` は空にせず、ヘッダー全体を100文字以内にします。

例:

```text
feat: add project bootstrap skill
fix(release): reject existing remote tags
docs: explain FSL verification boundary
ci: validate commit messages
```

破壊的変更は `!` または本文・フッターの `BREAKING CHANGE:` で明示します。

初回セットアップ時は次を実行して、mise 管理の Go版 commitlint とローカル hook を有効にします。

```bash
mise run setup-commitlint
```

GitHub Actions がコミットメッセージと Pull Request タイトルを検証します。

## Issue and Pull Request titles

Issue と Pull Request はともに `[Type]: Verb Summary` を使います。Type は `Feature`、`Bug`、`Improvement`、`Documentation`、`Security`、`Maintenance`、`Release` のいずれかです。Summary は大文字で始まる命令形動詞から始めます。動詞の固定リストは設けません。Release のみ `[Release]: vX.Y.Z` を使う例外です。PR は 1 件以上の Issue を扱えるため、Issue の Title との完全一致は要求しません。

Issue は Change または Release テンプレートで作成します。どちらも `Context`、`Goal`、`Scope`、`Acceptance criteria`、`Validation` を含めます。Release Issue には `Added`、`Changed`、`Fixed`、`Removed` の Changelog も必須です。公開版は `vX.Y.Z`、成果物のビルド識別子は `vX.Y.Z+N` とします。

## Branch and Pull Request flow

変更ごとに先に Issue を作成し、`issue/<番号>` ブランチを作ります。許可する PR の向きは `.github/branch-policy.toml` の `[[routes]]` で定義します。既定では `issue/<番号> -> main` です。別の統合・安定化ブランチが必要なプロジェクトだけ、追加の経路を明示的に設定してください。Dependabot による依存関係更新は、作成元 Issue がない自動化例外として `dependabot/* -> main` を許可し、`Closes #<番号>` を要求しません。作業ブランチを最新化するときは上流ブランチへ rebase し、`--force-with-lease` で push します。通常の `--force` は使用しません。`main` と追加した保護ブランチへの直接 push、force-push、削除は禁止です。マージ方式は rebase のみとし、コミットメッセージを PR タイトルによって置き換えません。PR に含まれるすべてのコミットは GitHub で検証済みの署名を持たなければならず、`Validate signed pull-request commits` 必須チェックがこれを強制します。GitHub rebase merge が `main` に生成する置換コミットだけはこの検査の対象外です。Issue ブランチの PR 本文には対応する `Closes #<番号>` を単独行で記載してください。
