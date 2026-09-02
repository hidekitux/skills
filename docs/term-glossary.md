# Term glossary

This repository writes its artifacts in English and holds conversation and reports in Japanese. `docs/writing-style.md` names each surface. The same concept therefore reaches a reader twice, once in each language. This glossary fixes the pair, so `finding` in an artifact and 指摘 in the report about it are visibly the same thing.

It records decisions, not vocabulary in general. `docs/writing-style.md` states the test that produced them and the test to apply to a term this file does not list.

## What this glossary covers

A term belongs here when it is working vocabulary: a term this repository uses in an English artifact to name a concept a Japanese report would also name. `finding` is working vocabulary. These are not:

- An identifier, a command, a path, or a file name. `mise run check:local`, `CATALOG.yml`, and `issue/<number>` keep their form in both languages.
- A GitHub Project field name or option, such as `Status` or `In progress`.
- A heading quoted so the reader can find a section. `## Findings` names a location; the `finding` inside it names a concept.

A term that names both a concept and an identifier follows the use at hand. `task` in `mise run check:all` is an identifier; a task in an implementation plan is working vocabulary. `policy` in `.github/branch-policy.toml` works the same way.

## Terms that take a Japanese word

The rejected column records a candidate the read-aloud test turned down, so a reader can see what the choice ruled out and argue with it.

| English term | Japanese term | Rejected | Why the rejection |
| --- | --- | --- | --- |
| finding | 指摘 | 発見 | 発見 is something nobody knew before; a review comment is not that |
| boundary | 境界 | 範囲 | 範囲 is the area, not the line around it, and it is already taken by `scope` |
| evidence | 根拠 | 証拠 | 証拠 carries a forensic sense the English term does not have |
| adoption gate | 採用判定 | 関門 | Nobody reaches for 関門 in speech about a release |
| scope | 範囲 | スコープ | A common Japanese word says the same thing |
| plan | 計画 | プラン | A common Japanese word says the same thing |
| report | 報告 | レポート | A common Japanese word says the same thing |
| change | 変更 | 変更点 | 変更点 names the places that differ, not the act |
| validation | 検証 | バリデーション | A common Japanese word says the same thing |
| handoff | 引き継ぎ | ハンドオフ | A common Japanese word says the same thing |
| artifact | 成果物 | アーティファクト | A common Japanese word says the same thing |
| acceptance criteria | 受け入れ条件 | 受入基準 | 受入基準 reads as a measuring standard, not a condition that holds or does not |
| phase | 段階 | フェーズ | A common Japanese word says the same thing |
| severity | 深刻度 | 重要度 | 重要度 is how much something matters, not how badly it breaks |
| defect | 不具合 | 欠陥 | 欠陥 states a flaw in the product itself and is heavier than the term means here |
| root cause | 根本原因 | 真因 | 真因 is trade jargon that a reader outside the trade will stop at |
| behavior | 挙動 | 振る舞い | 振る舞い is what a person does |
| diff | 差分 | ディフ | A common Japanese word says the same thing |
| approval | 承認 | アプルーバル | A common Japanese word says the same thing |
| reproduction | 再現 | リプロダクション | A common Japanese word says the same thing |
| task | 作業 | タスク | A common Japanese word says the same thing |
| policy | 方針 | ポリシー | A common Japanese word says the same thing |

## Terms that keep their katakana form

These are established in the field. A Japanese speaker says them aloud, so replacing them would cost the reader rather than help them.

ブランチ, ワークツリー, チェックアウト, コミット, マージ, リポジトリ, リリース, レビュー, テスト, コマンド, スキル, ワークフロー.

## Terms that stay in English

These name something the reader looks up under this exact name, so a Japanese form would send them to a name that does not exist.

| English term | Why it stays |
| --- | --- |
| Issue | The GitHub object, labeled this way in the interface a reader opens |
| Pull Request | The GitHub object, labeled this way in the interface a reader opens |
| Project | The GitHub Project that tracks Issue status; `.github/issue-project.toml` declares it |
| Todo List | Names the contract stated under `## Todo List contract` in `AGENTS.md` |
| FSL | The specification language; `docs/fsl.md` defines the boundary |

## A term this glossary does not list

Apply the decision test in the `Deciding whether a term stays in English` section of `docs/writing-style.md`. It reaches an answer without a word list. Add the result here when the term turns up a second time; a term used once does not need a row.
