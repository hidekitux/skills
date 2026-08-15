# hidekitux/skills

個人・チームで再利用する Agent Skills の正本リポジトリです。各スキルは [Agent Skills](https://agentskills.io/specification) 形式で管理し、`gh skill` で公開・導入します。

## ライセンス

このリポジトリと公開スキルは [Apache License 2.0](LICENSE) です。各
`SKILL.md` の `license: Apache-2.0` とルートの `LICENSE` を正本として扱います。著作権者と対象年は [NOTICE](NOTICE) に記録します。

`LICENSE` は GitHub API から取得した標準本文を保持します。付録の
`[yyyy] [name of copyright owner]` はライセンス本文の記入例であり、暦年ごとに
自動更新しません。新しい著作物を追加・更新した年だけ `NOTICE` の年を更新します。

## 開発環境

[mise](https://mise.jdx.dev/) をこのリポジトリの標準コマンド入口として使います。初回は設定を信頼し、利用可能なタスクを確認してください。

```bash
mise trust
mise tasks ls
```

このリポジトリは `skill-creator` の検証に Python と uv を使うため、`mise.toml` でそれらを固定しています。ほかのツールは、リポジトリが実際に必要になった時点でだけ追加します。

`mise run validate` は、FSL検証を含むため Linux x64 と macOS Apple Silicon をサポート対象とします。ほかのプラットフォームでは、対応する `fslc` が導入されるまで完全な検証は実行できません。

## ディレクトリ

```text
skills/<skill-name>/SKILL.md
skills/<namespace>/<skill-name>/SKILL.md
```

各スキルには `SKILL.md` が必須です。`name` は親ディレクトリ名と一致させ、英小文字・数字・ハイフンだけを使用します。スクリプト、詳細な資料、テンプレートは必要になったスキルだけに `scripts/`、`references/`、`assets/` として追加してください。

## Todo List

すべての公開スキルは、実行開始時に Todo List を作成して進捗を維持します。項目には必要に応じて、調査・スコープ確認・実装・検証・引き渡しを含めます。ネイティブの Todo List 機能があるホストではそれを使用し、ない場合は会話内の Markdown チェックリストを同じ契約として使います。完了は根拠が得られた項目だけに付け、未完了項目は引き渡し時に明示します。

## 開発手順

1. `skills/<skill-name>/SKILL.md` を追加する。
2. `CATALOG.yml` に用途・所有者・対応エージェントを記録する。
3. 公開前に検証する。

```bash
mise run validate
```

`validate` には、`CATALOG.yml`、Apache-2.0 表記、ホストアダプター、Todo List
契約を確認するリポジトリ整合性検査も含まれます。単独で確認する場合は
`mise run validate-repository` を使います。

Codex でスキルを作成・更新した場合は、`skill-creator` の検証も加えます。

```bash
mise run validate-local
```

4. レビュー後、[リリース手順](docs/releasing.md)に従ってカタログとセマンティックバージョンのタグを検証してから公開する。

```bash
mise run verify-release -- vX.Y.Z
gh skill publish --tag vX.Y.Z
```

新規作成・大幅更新では、必ず `skill-creator` を使います。依頼内容を [スキル作成ブリーフ](docs/skill-brief-template.md) の項目で渡すと、発動条件、出力、検証方法を不足なく設計できます。

## 利用

Codex のユーザー領域へ全スキルを導入する例です。

```bash
gh skill install hidekitux/skills --all --agent codex --scope user
```

プロジェクト単位で導入する場合は、対象プロジェクトで `--scope project` を指定します。リリースを固定したい場合は `skill-name@vX.Y.Z` を指定してください。

```bash
gh skill install hidekitux/skills <skill-name>@vX.Y.Z --agent codex --scope project
```

## エージェント互換性

スキルの正本は `skills/` に一つだけ置きます。Codex と Claude Code などの差異は、共通の `SKILL.md` を複製せず、必要な場合だけスキル内の `references/hosts/<host>.md` に記録します。各ノートには、利用可能な機能、代替手段、結果の検証方法を記載してください。

```text
skills/pr-review/
├── SKILL.md
├── agents/openai.yaml              # Codex UI向け。必要な場合だけ
└── references/hosts/
    ├── codex.md                    # 実行差分がある場合だけ
    └── claude-code.md

hosts/
├── codex/                          # リポジトリ全体のCodex設定例
└── claude-code/                    # リポジトリ全体のClaude Code設定例
```

`hosts/` は設定例・導入補助のための管理対象であり、公開スキルではありません。`.codex/`、`.claude/`、`.agents/` は利用先で生成・導入されるローカル状態としてGit管理しません。

## 安全性

スキルに秘密情報を含めないでください。`allowed-tools` に shell や bash を指定するのは、スクリプトと参照先をレビュー済みの場合だけにします。

## FSL による仕様検証

FSL は `SKILL.md` の書式を検査するものではなく、スキルが規定する状態遷移や公開条件を形式仕様として検証するために使います。たとえば、レビュー前に公開できないこと、非推奨化したスキルを新規導入しないこと、更新時に検証結果を失わないことを対象にします。

FSL 仕様は `specs/*.fsl` に置きます。自然言語の運用ルールから仕様を作る前に、状態・アクション・不変条件・未確定事項をまとめた formalization memo をレビューしてください。

```bash
# すべての FSL 仕様を構文・型検査してから有限深さで検証する
mise run verify-fsl
```

仕様を追加したら、通常のスキル検証に加え、`fslc mutate` で性質が実際に検出力を持つことも確認します。詳細は [docs/fsl.md](docs/fsl.md) を参照してください。

公開フローの順序とタグ再利用禁止は [specs/release-gate.fsl](specs/release-gate.fsl) で検証しています。
