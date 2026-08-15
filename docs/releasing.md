# リリース手順

このリポジトリは、1つの GitHub Release を1つのセマンティックバージョンとして扱います。`CATALOG.yml` に記録された全スキルの `version` は、同じリリースタグのバージョン部分と一致させます。

## リリース前

1. リリース対象の `vX.Y.Z` に、`CATALOG.yml` の全スキルの `version` を揃える。
2. Todo List、検証結果、変更内容をレビューする。
3. リポジトリルートで `mise run validate` を実行する。Codex で
   `skill-creator` が利用可能な場合は、追加で `mise run validate-local` を実行する。
4. `specs/*.fsl` を変更した場合は `mise run mutate-fsl` を実行し、survivor をレビューする。
5. 変更をコミットする。
6. `mise run verify-release -- vX.Y.Z` で、タグ形式・カタログのバージョン・コミット済み状態を確認する。

`verify-release` はタグを作成せず、既存のローカルタグも再利用しません。未コミット変更や未追跡ファイルがある場合も失敗します。

## 公開

検証済みのコミットから、次を実行します。

```bash
gh skill publish --tag vX.Y.Z
```

`gh skill publish` は `skills/*/SKILL.md` などの規約でスキルを検出し、指定タグの GitHub Release を作成します。公開後は対象タグと Release の内容を確認し、利用側では `skill-name@vX.Y.Z` のように固定して導入します。

## バージョン規則

- 互換性を壊さない機能追加は minor、修正は patch、互換性を壊す変更は major とする。
- リポジトリ内の全スキルを同じタグで公開するため、1つだけ変更した場合でも `CATALOG.yml` の全エントリをタグのバージョンに揃える。
- 公開済みタグは上書きせず、修正リリースを新しい patch バージョンで作成する。
