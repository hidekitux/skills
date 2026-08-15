# FSL の運用

## 目的

FSL は、スキルの本文や Markdown の品質を直接判定するものではありません。このリポジトリでは、スキルのライフサイクルや高リスクなスキルが定める業務フローを、状態機械として検証するために使います。

適した対象は次のとおりです。

- 作成、レビュー、検証、公開、更新、非推奨化の遷移
- 「検証に成功した版だけ公開できる」のような不変条件
- 承認、ロールバック、再試行、権限、二重実行を含むスキルのワークフロー

単なる文体、ファイル名、frontmatter の検査には `gh skill publish --dry-run` を使います。

## 配置と検証

仕様の正本は所有者とともに置き、コピーは作りません。

- 公開スキル自身のワークフローを表す仕様は `skills/<skill-name>/specs/<topic>.fsl` に置きます。リポジトリからも検証・参照するため、`specs/<skill-name>/<topic>.fsl` にその正本への**相対シンボリックリンク**を作ります。
- リリースゲート、横断的なブランチ方針など、特定の公開スキルに属さない仕様は `specs/<topic>.fsl` に通常ファイルとして置きます。

この packaging memo は #29 で確認済みです。`issue-creation.fsl` は `create-issue`、`pull-request-creation.fsl` は `create-pr` が所有します。`branch-flow.fsl` と `release-gate.fsl` はリポジトリ固有の正本として残します。仕様を追加したら、リポジトリルートで次を実行します。

```text
mise run verify-fsl
```

このタスクは、リポジトリ直下の通常ファイルと各 `skills/**/specs/*.fsl` の正本を発見し、公式リリースの `fslc` v4.2.0 を SHA-256 で検証して、CI では `RUNNER_TEMP`、ローカルでは `TMPDIR` 配下の一時キャッシュに導入してから、各仕様に対して `fslc check` と `fslc verify --depth 8` を実行します。シンボリックリンクはリポジトリ検証で整合性を確認し、二重に実行しません。深さは必要に応じて `FSL_DEPTH=12 mise run verify-fsl` のように上書きできます。対応プラットフォームは GitHub Actions の Linux x64 と開発環境の macOS Apple Silicon です。

## 作成時の約束

FSL 仕様を書く前に、チャット上で formalization memo を確認します。memo には状態、アクション、禁止状態、境界条件、表現上の仮定、未決事項を含めます。動作に影響する未決事項を推測で補完しません。

仕様が検証を通った後は `mise run mutate-fsl` で重要な性質の検出力を確認し、弱めた仕様や壊した遷移を検出できることを確認します。mutation の survivors はレビュー対象であり、単独では自動失敗ではありません。FSL の検証結果は「仕様が整合している」ことを示すものであり、実装やスキル本文が仕様どおりであることまで自動的に保証するものではありません。

## 公開ゲート仕様

`specs/release-gate.fsl` は、カタログ更新、`mise run validate`、コミット、
`mise run verify-release -- vX.Y.Z`、公開の順序をモデル化します。公開は検証済みの
クリーンなコミットからだけ可能で、既存タグの再利用は許可しません。これは GitHub
API の公開結果を証明するものではなく、公開コマンドを実行してよい前提条件を検証する
仕様です。
