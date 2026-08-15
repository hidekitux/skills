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

## Branch and Pull Request flow

変更ごとに先に Issue を作成し、`issue/<番号>` ブランチを作ります。許可する PR の向きは `.github/branch-policy.toml` の `[[routes]]` で定義します。既定では `issue/<番号> -> develop -> release/vX.Y.Z -> main` ですが、release ブランチが不要なプロジェクトでは `develop -> main` の経路だけにできます。`main`、`develop`、`release/*` への直接 push、force-push、削除は禁止です。Issue ブランチの PR 本文には対応する `Closes #<番号>` を単独行で記載してください。
