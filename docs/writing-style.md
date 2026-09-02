# Writing style

This document is the detail behind the `Writing quality` section of `AGENTS.md`. It applies to every piece of prose an agent writes for this repository: Issue and Pull Request bodies, `docs/`, `SKILL.md`, commit message bodies, and Japanese conversational reports. Each of those artifacts is written in English, and the conversation carrying the reports is held in Japanese. Code, identifiers, commands, and quoted output are exempt.

## How to read this document

The rules sit in three parts. `Rules for both languages` binds every draft. `Rules for English` and `Rules for Japanese` each bind one language and nothing else.

Read the shared part, then the part for the language you are writing in, and stop. A Japanese writer never needs the English part, and the reverse holds too. Each part carries its own patterns, rules, thresholds, and checklist, so one pass through two parts reaches every rule that binds you.

Within a part, check a draft against the machine-typical patterns, then against the rules, then against the thresholds. Then apply the convergence rule before calling anything a defect.

One concept is often named twice, once in an English artifact and once in a Japanese report. [term-glossary.md](term-glossary.md) fixes those pairs, and the Japanese part states the test for a term the glossary does not list.

## Rules for both languages

### Machine-typical patterns

- **Anthropomorphism.** A document, a command, or a data set does not decide or speak, so giving it intent hides who or what actually acts.
  - Before: The catalog tells us which skills are published.
  - After: The `skills:` list in `CATALOG.yml` is the published inventory.
- **Uniform sentence length.** When every sentence runs to the same length the prose loses stress, and the reader cannot tell which sentence carries the load.
  - Before: The task runs the checks. The checks report the result. The result blocks the push.
  - After: The task runs the checks. A failure blocks the push, and the diagnostic names the check that failed.
- **Rule-of-three padding.** A polished triplet reads as complete whether or not the third item is real, so the reader accepts a filler item as a fact.
  - Before: The check is fast, reliable, and comprehensive.
  - After: The check runs in under a second and reads only tracked files.
- **Transition stacking.** `Furthermore`, `Moreover`, and `Additionally` announce a connection the sentences already have, and one connector per paragraph opening is already too many when the ideas are ordered.
  - Before: Furthermore, the hook reruns setup. Moreover, it records the result.
  - After: The hook reruns setup and records the result.
- **One repeated paragraph shape.** A document where every paragraph runs topic sentence, evidence, restatement makes the last sentence of each paragraph predictable and skippable.
  - Before: three paragraphs that each open with a claim, cite one command, and close by restating the claim.
  - After: one paragraph states the rule, the next gives the command, and the third states the failure case.
- **Bold-header listicle.** A list where every item is `**Label**: one explanatory sentence` flattens items of different weight into one shape, and hides which item the reader must act on.
  - Before: **Speed**: The check is fast. **Scope**: The check reads tracked files.
  - After: The check reads only tracked files, which is why it finishes in under a second.
- **Equal-length list items.** Items padded or trimmed to match each other lose the size difference that tells the reader which item is larger.
  - Before: four items of eleven words each.
  - After: one item of three words, one of twenty, ordered by what the reader does first.
- **Heading and bold sprinkling.** A heading over two sentences, or a bold phrase in every paragraph, spends the reader's attention on structure that carries no information.
  - Before: a `###` heading above each of five two-sentence paragraphs.
  - After: one paragraph of five sentences, with no heading and no bold text.
- **Meta-commentary.** `In this document, we will explore` and `In conclusion` describe the document instead of stating its content, so the reader pays for a sentence that says nothing.
  - Before: In this section, we will explore how validation works.
  - After: Validation runs in three tiers, and Tier 1 is required on every Pull Request.
- **Hedge sandwich.** `While X is important, it is worth noting Y` states two positions and commits to neither, leaving the reader to decide what the writer thought.
  - Before: While the check is useful, it is worth noting that it can be slow.
  - After: The check takes four minutes, so run it before the push rather than on every save.
- **Refusal to pick a side.** `Both options have merit` is the answer that costs nothing and helps nobody, because the reader asked which one to use.
  - Before: Both a hook and a CI check have merit.
  - After: Use the hook, because it fails before the push instead of after it.
- **Sycophantic opener.** `Great question` and `That is a thoughtful point` delay the answer and tell the reader nothing about the answer.
  - Before: Great question. The task is `check:local`.
  - After: Run `mise run check:local`.
- **Ghost citation.** `Studies show` and `experts agree` claim authority without naming it, so the reader cannot check the claim.
  - Before: Studies show that shorter sentences read faster.
  - After: The Federal Plain Language Guidelines require short sentences, and `docs/writing-style.md` states the threshold.
- **Restatement that adds no fact.** A sentence that repeats the previous sentence in new words doubles the reading cost and adds nothing.
  - Before: The hook blocks the commit. In other words, the commit does not proceed.
  - After: The hook blocks the commit.
- **Over-explanation of the known.** Defining a term the audience already uses signals that the writer did not think about the reader.
  - Before: A commit, which is a recorded change in Git, must follow the header rule.
  - After: Every commit header must follow the `type: summary #<number>` rule.
- **Abstraction where a detail existed.** A general noun in place of the specific one forces the reader to guess which file, command, or value is meant.
  - Before: The configuration declares the relevant fields.
  - After: `.github/issue-project.toml` declares the `Status`, `Priority`, and `Scope` fields.

### Readable-writing rules

Every rule below names its source, so a reader can check it.

- **State the conclusion first.** Source: Japanese technical-writing practice. A reader who gets the conclusion first knows what to do with the detail that follows.
  - Before: Because the hook reruns on checkout and the pinned commitlint is shared, parallel worktrees do not conflict.
  - After: Parallel worktrees do not conflict. The hook reruns on checkout, and the pinned commitlint is shared.
- **Use the active voice.** Source: Federal Plain Language Guidelines, and the Google developer documentation style guide. The active voice names who acts, which a passive sentence can leave out.
  - Before: The branch is created from the upstream default branch.
  - After: `implement-issue` creates the branch from the upstream default branch.
- **Prefer the concrete word to the abstraction.** Source: Federal Plain Language Guidelines. A concrete noun tells the reader which thing is meant; an abstraction makes them guess.
  - Before: The tooling reports the outcome.
  - After: `mise run check:local` prints the failing check and its exit code.
- **Prefer the short everyday word.** Source: Orwell's rules for writing. A short word is read faster and understood by more readers, with no loss of precision.
  - Before: The command necessitates prior authorization.
  - After: The command needs approval first.
- **Use a word people actually use.** Source: this repository's convention, adopted after review rejected `関門` as the Japanese word for `adoption gate`. Read the replacement aloud: a word you would not say to a colleague working on the same thing fails, however exactly a dictionary matches it to the term. The rule binds the replacement and the prose around it in either language, so a draft assembled from rare words fails even when every rare word is correct.
  - Before: 採用の関門を通過したスキルだけを公開します。
  - After: 採用判定を通ったスキルだけを公開します。
- **Cut every word that can be cut.** Source: Orwell's rules for writing. A word that carries no fact costs the reader time.
  - Before: It is important to note that the check is required on every Pull Request.
  - After: The check is required on every Pull Request.
- **Vary sentence length deliberately.** Source: this repository's convention, adopted because uniform length is a machine-typical pattern. A short sentence after two long ones marks the point that matters.
  - Before: The task runs the checks and reports the result, and the result decides whether the push proceeds or stops.
  - After: The task runs the checks and reports the result. A failure stops the push.
- **State a position and give its reason.** Source: this repository's convention. A reader who asked which option to use needs an answer, not a survey.
  - Before: A hook and a CI check are both possible here.
  - After: Use the hook. It fails before the push, so a bad commit never reaches the remote.
- **Name the source of every claim.** Source: this repository's convention. A claim about the repository must cite the file, command, or output it came from, so the reader can verify it.
  - Before: The repository requires ten checks on every Pull Request.
  - After: `CONTRIBUTING.md` lists the ten required Tier 1 checks.
- **Use one term for one concept.** Source: the Google developer documentation style guide, and technical-writing guidance on terminology consistency. Alternating terms makes the reader ask whether two names mean two things.
  - Before: Run the setup task first. The initial task also enables the hooks.
  - After: Run `mise run setup:all` first. `mise run setup:all` also enables the hooks.
- **Write headings in sentence case.** Source: the Google developer documentation style guide. Sentence case keeps a heading readable as a phrase instead of a label.
  - Before: `## Writing Quality And Style Rules`
  - After: `## Writing quality`

### Consistency and substance rules

These four rules catch the defects the pattern catalog cannot see from a single sentence.

- **Name a thing in full on first mention, then reuse that exact term.** Source: the Google developer documentation style guide, and technical-writing guidance on terminology consistency. A shortened or varied later mention makes the reader ask whether it is the same thing. Define a short form explicitly when the full term is long enough to need one, then use only the short form.
  - Before: Run `mise run setup:all` once. `setup:all` is safe to rerun, and the setup task runs again on checkout.
  - After: Run `mise run setup:all` once. `mise run setup:all` is safe to rerun, and it runs again on checkout.
- **Make every sentence add a fact the reader did not have.** Source: this repository's convention. Delete each sentence in turn and ask what the reader loses; a sentence that loses nothing does not belong.
  - Before: Validation matters for this repository. Validation is how the repository stays correct. Tier 1 runs on every Pull Request.
  - After: Tier 1 validation runs on every Pull Request.
- **Write prose by default, and use a list only for items a reader counts.** Source: this repository's convention. A list of two related sentences hides the connection that prose would state.
  - Before: a three-item list whose items are the cause, the effect, and the exception.
  - After: one sentence stating the cause and effect, and a second stating the exception.
- **Cite the file, command, or output behind every claim about the repository.** Source: this repository's convention. A claim the reader cannot trace is a claim they must re-derive.
  - Before: The Project configuration declares the required fields.
  - After: `.github/issue-project.toml` declares the `Status`, `Priority`, and `Scope` fields with their options.

### Thresholds

Every threshold states how to count it and what the count excludes, because a threshold without its counting rule is not usable: two readers would measure the same passage differently. The rows below bind both languages. The English and Japanese parts carry their own.

| Threshold | How to count | Excluded from the count |
| --- | --- | --- |
| At most one paragraph in four opening with a formal connector | Count paragraphs opening with `Furthermore`, `Moreover`, `Additionally`, `また`, or `さらに` | Numbered procedure steps, where the connector marks order |
| At most one polished triplet per 200 words | Count three-item parallel lists that sit inside one sentence | A genuine enumeration of three real items, such as three field names |
| No sentence survives the deletion test | Delete each sentence in turn and name the fact the reader loses | A sentence whose only job is to state the conclusion before the detail |

The one-in-four connector limit is set by this repository rather than taken from the cited research, and is stricter than the more-than-half share the research treats as the tell.

### Counting exceptions

These three exceptions apply to every threshold in this document.

- **A `Before` example is exempt from every threshold.** An example that demonstrates a pattern has to contain the pattern, so measuring it would flag the document for showing the reader what to avoid. Measure the document's own prose, not its quoted material.
  - Before: The count includes the `検証を行うことが必要です。` line and reports this file as breaking the deletion test.
  - After: The count skips every `Before` line and reports only the document's own prose.
- **An inline code span counts as one reading unit.** `git worktree add -b issue/<number> <path> origin/main` is one token to a reader and 53 characters to a naive count, so counting characters would flag a sentence that reads short. Count the span as one unit, not as its length.
  - Before: `mise run setup:all` は `.githooks` を有効にします。 counts as 43 characters and reads as close to the 50-character guide.
  - After: The same sentence counts as 14 characters, because each code span is one unit, and it reads as the short sentence it is.
- **A genuine enumeration is excluded from the sentence-length threshold.** A sentence that lists several real items is long because the list is long, and shortening it would drop an item. This exception does not cover rule-of-three padding, where the third item exists only to complete the pattern; the test is whether removing an item removes a fact.
  - Before: この文書はパターンを、語彙と語法、構造とリズム、実質と姿勢、日本語固有の登録という4つの群に分けて並べます。 is 54 characters with the code-span exception already applied, and dropping 実質と姿勢 alone leaves 48, so reaching the guide costs the reader a group name.
  - After: The sentence keeps all four group names, because removing one removes a fact, and the enumeration is excluded from the count.

### The convergence rule

One marker is not a defect. An em dash, a triplet, or a single formal connector appears in careful human writing, and rewriting a passage over one marker costs more than it returns.

Rewrite when three or more markers converge in one passage, or when a threshold in this document is exceeded. Report a single marker only when the reader would misread the sentence.

### Checklist for both languages

- [ ] Every section states its conclusion before its detail.
- [ ] Every sentence is active and names who acts.
- [ ] Every noun is the specific one, every word is the short everyday one, and every word that can be cut is cut.
- [ ] Nothing explains what the audience already knows.
- [ ] Every word, including one chosen to replace a loanword, is a word people use in speech.
- [ ] Every heading is in sentence case.
- [ ] No anthropomorphism remains.
- [ ] The triplet and formal-connector thresholds hold.
- [ ] No paragraph repeats one shape, and no list forces its items to equal length.
- [ ] Headings and bold text mark structure the reader needs, not every paragraph.
- [ ] No meta-commentary, hedge sandwich, sycophantic opener, or refusal to pick a side remains.
- [ ] Every claim names its file, command, output, or external source.
- [ ] Every sentence adds a fact, verified by the deletion test.
- [ ] Every concept keeps one term from first mention to last.
- [ ] Prose carries the argument, and each list holds items a reader counts.
- [ ] Any remaining marker is a single one, permitted by the convergence rule.

## Rules for English

### Machine-typical patterns

- **Inflated style word.** A style word such as `delve`, `pivotal`, `tapestry`, `realm`, `underscore`, or `multifaceted` replaces a plain verb or noun and carries no extra fact, so the reader decodes register instead of content.
  - Before: The skill delves into the repository to surface pivotal findings.
  - After: The skill reads the repository and reports the findings that block the release.
- **Latinate padding.** `utilize`, `facilitate`, and `commence` are longer than `use`, `help`, and `start` and mean the same thing, so the extra syllables buy nothing.
  - Before: Utilize mise to commence the validation.
  - After: Use mise to start the validation.
- **Copula avoidance.** `serves as`, `boasts`, and `features` stand in for `is` and turn a plain statement of identity into a claim of importance.
  - Before: mise serves as the standard command entry point.
  - After: mise is the standard command entry point.
- **Negative parallelism.** `not just X, but Y` promises a contrast and usually delivers a restatement, so the reader waits for a distinction that never arrives.
  - Before: The check is not just a lint step, but a policy gate.
  - After: The check blocks the push when the branch policy fails.
- **The `from X to Y` frame.** The frame implies a range the sentence does not define, so it reads as coverage without naming the covered set.
  - Before: The skills cover everything from planning to release.
  - After: The skills cover planning, implementation, review, merge, and release.
- **Gerund opener.** A sentence that opens with an `-ing` phrase buries the subject and makes the reader hold the modifier until the verb arrives.
  - Before: Offering a bounded review, review-pr returns findings by severity.
  - After: review-pr returns findings ordered by severity.
- **Em dash pile-up.** An em dash every second sentence turns every clause into an aside, so nothing reads as the main point.
  - Before: The task, run from the root — always from the root — validates the tree.
  - After: Run the task from the repository root. It validates the whole tree.

### Readable-writing rules

- **Use the second person and the present tense.** Source: the Google developer documentation style guide. `You` names the actor, and the present tense keeps the document true whenever it is read.
  - Before: Developers will be required to run the validation before pushing.
  - After: Run the validation before you push.
- **Keep one idea in one sentence.** Source: Japanese technical-writing practice, applied here to English as well. A sentence with two ideas forces the reader to hold the first while parsing the second. The threshold table below sets the point where a sentence has certainly stopped holding one idea, at 45 words.
  - Before: The skill reads the Issue and derives the tasks, and it commits each task separately so the history mirrors the plan.
  - After: The skill reads the Issue and derives the tasks. It commits each task separately, so the history mirrors the plan.

### Thresholds

| Threshold | How to count | Excluded from the count |
| --- | --- | --- |
| At most 45 words per English sentence | Count words between sentence-ending punctuation; an inline code span counts as one word whatever its length | A genuine enumeration; headings, list items, table cells, and code blocks |
| At most 10 em dashes per 1,000 words | Count `—` in sentences, divide by the word count, multiply by 1,000 | Code spans, code blocks, quoted output, and the ` — ` separator of a reference-list entry, which is structural rather than rhetorical |
| Sentence-length variance of at least 0.5 | Divide the standard deviation of words per sentence by the mean, over prose sentences only | Headings, list items, table cells, and code blocks; a sample below ten prose sentences, where one sentence would decide the result |

Both numbers here are set by this repository rather than taken from the cited research. The 45-word limit takes the shape of the Japanese sentence-length rule, a guide carrying the same enumeration exclusion. Eight sentences of 45 words or more sat in tracked Markdown when the limit was written. The longest ran to 64 words. The variance minimum of 0.5 sits between the 0.2 to 0.4 range the research reports for machine prose and the 0.6 to 1.2 range it reports for human prose.

### Checklist for English

- [ ] No inflated style word, Latinate padding, or copula avoidance survives where a plain word fits.
- [ ] Instructions use the second person and the present tense.
- [ ] No negative parallelism, `from X to Y` frame, or gerund opener remains.
- [ ] No sentence runs past 45 words, unless the length is a genuine enumeration.
- [ ] Sentence length varies, and the em dash threshold holds.

## Rules for Japanese

### Machine-typical patterns

- **翻訳調.** `〜することができます` and `〜という事実` are English constructions carried into Japanese, and they add characters without adding meaning.
  - Before: この方法を用いることができます。
  - After: この方法を使えます。
- **冗長な水増し.** `〜を行う` and `〜を実施する` wrap a verb the sentence already has.
  - Before: 検証を行うことが必要です。
  - After: 検証します。
- **不要なカタカナ語と文中英単語.** A loanword where a common Japanese word exists makes the reader translate twice. Keep a term that is established in the domain, such as `ブランチ`, `チェックアウト`, or `ワークツリー`, and keep every identifier, command, and path in its original form.
  - Before: チェックアウトしたスナップショットにスキルをレジストします。
  - After: チェックアウトした時点の内容にスキルを登録します。
- **敬体と常体の混在.** Switching between `です・ます` and `だ・である` inside one document reads as two writers.
  - Before: 検証は3層である。Tier 1 は必須です。
  - After: 検証は3層です。Tier 1 は必須です。
- **体言止めの混在.** A noun-final sentence among predicate-final sentences hides whether the line is a statement or a label.
  - Before: フックが再実行。結果を記録します。
  - After: フックが再実行します。あわせて結果を記録します。
- **曖昧なぼかし.** `一般的に〜とされています` and `〜と言われています` attribute a claim to nobody, so the reader cannot check it. This is the Japanese phrasing of the ghost citation in the shared part.
  - Before: 一般的に短い文が読みやすいとされています。
  - After: 本多勝一は、長い修飾語の境界に読点を打つよう定めています。

### Readable-writing rules

- **一文一義を守り、1文を50字程度にします。** Source: Japanese technical-writing practice. 1文に2つの動作を入れると、読者は前半を保持したまま後半を読む必要があります。
  - Before: `mise run setup:all` は時点の内容にスキルを登録し、固定版の commitlint を再利用します。
  - After: `mise run setup:all` は時点の内容にスキルを登録します。あわせて固定版の commitlint を再利用します。
- **接続助詞は1文に2つまでにします。** Source: Japanese technical-writing practice. 接続助詞が3つ以上あると、文の切れ目が読者に見えなくなります。
  - Before: 検証が失敗するとプッシュが止まるので、原因を直してから再実行しますが、そのときも同じコマンドを使います。
  - After: 検証が失敗するとプッシュは止まります。原因を直し、同じコマンドで再実行します。
- **修飾語は長い順に前へ置きます。** Source: 本多勝一『日本語の作文技術』の修飾の順序4原則。節を句より前に置き、長い修飾語を先に置き、大きな状況を先に置き、親和度の強い語を離します。
  - Before: 白い横線の引かれた厚手の紙
  - After: 横線の引かれた厚手の白い紙
- **読点は長い修飾語の境界に打ちます。** Source: 本多勝一『日本語の作文技術』の読点の原則。読点は必要最小限にとどめ、修飾語の原則に対して語順が逆のときにも打ちます。分かち書きを目的とした読点は打ちません。
  - Before: 診断コマンドが報告する登録済みで非ベアのワークツリーを使います。
  - After: 診断コマンドが報告する、登録済みで非ベアのワークツリーを使います。
- **敬体で統一し、体言止めを混ぜません。** Source: this repository's convention. 文体が混ざると、書き手が複数いるように読めます。
  - Before: 検証は3層である。Tier 1 は必須です。
  - After: 検証は3層です。Tier 1 は必須です。
- **定着した技術用語は残し、置き換えられるカタカナ語は日本語にします。** Source: this repository's convention, and the `JTF日本語標準スタイルガイド`. `ブランチ` や `ワークツリー` のような定着した用語は残します。識別子、コマンド、パスは原形のまま書きます。
  - Before: リリースのフローをドキュメントにデスクライブします。
  - After: リリースの手順を文書に書きます。

### Deciding whether a term stays in English

Japanese prose keeps an English or katakana term only when replacing it would cost the reader. Three tests decide that, applied in order.

1. Is the term an identifier, a command, a path, a file name, or a heading quoted so the reader can find a section? Keep the original form and stop. `mise run check:local` and `CATALOG.yml` never take a Japanese word.
2. Does the reader look the term up under this exact name in an interface? `Issue`, `Pull Request`, and the Project field `Status` stay in English, because a Japanese form sends the reader to a name GitHub does not show.
3. Would a Japanese speaker in this field say the term aloud in a sentence about this work? `ブランチ` and `マージ` pass and stay. `boundary` and `evidence` fail and take 境界 and 根拠.

A term that fails all three takes the Japanese word that carries the same sense with nothing added and nothing lost. 証拠 was rejected for `evidence`, because it carries a forensic sense the English term does not have. Apply the ordinary-word rule to whatever word you choose. When no word passes it, keep the English term and say why.

[docs/term-glossary.md](term-glossary.md) records the decisions already made, including `finding`, `boundary`, `evidence`, and `adoption gate`. Read it before applying the tests. Add a row when a term turns up a second time.

### Thresholds

| Threshold | How to count | Excluded from the count |
| --- | --- | --- |
| About 50 characters per Japanese sentence, and 70 at most | Count characters between `。`; an inline code span counts as one reading unit whatever its length | A genuine enumeration; headings and table cells |
| At most two connective particles per Japanese sentence | Count `して`, `ため`, `ので`, `が、`, `し、`, and `て、` | Nothing |

### Checklist for Japanese

- [ ] Japanese text holds one idea per sentence, stays near 50 characters with the counting exceptions applied, and keeps at most two connective particles.
- [ ] Japanese text places long modifiers first, puts `読点` at modifier boundaries, keeps one register, and replaces only the loanwords that have a common Japanese equivalent.
- [ ] Every English or katakana term left in Japanese text passed the decision test, and a term `docs/term-glossary.md` already decided matches the glossary.

## Sources

- [Principles of plain language](https://digital.gov/guides/plain-language/principles) and the [Federal Plain Language Guidelines](https://www.wordrake.com/resources/federal-plain-language-guidelines) — active voice, short sentences, and the concrete word.
- [Google developer documentation style guide](https://developers.google.com/style/highlights) — second person, present tense, sentence-case headings, and one term per concept.
- [Using large language models in technical writing](https://developers.google.com/tech-writing/two/llms) — terminology consistency and its cost to the reader.
- [本多勝一『日本語の作文技術』のまとめ](https://www.math.nagoya-u.ac.jp/~shinichiroh/2018/02/13/japanese-punctuation.html) — the modifier-order principles and the `読点` rules.
- [テクニカルライティングで伝わる文章を書くコツ](https://tech.trustbank.co.jp/entry/20241210/technical-writing) — `一文一義`, sentence length, connective-particle count, and conclusion first.
- [JTF日本語標準スタイルガイド](https://www.jtf.jp/pdf/jtf_style_guide.pdf) — Japanese orthography and notation.
- [George Orwell's six rules for writing](https://www.openculture.com/2025/12/george-orwells-six-rules-for-writing.html) — the short everyday word and cutting every word that can be cut.
- [Wikipedia:Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing), [Signs of AI writing: 27 red flags](https://vrid.ai/blog/signs-of-ai-writing), and [Signs of AI writing: 12 patterns with reproducible thresholds](https://slopdetector.org/blog/signs-of-ai-writing) — the machine-typical patterns and the countable thresholds.
