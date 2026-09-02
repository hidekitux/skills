# Writing style

This document is the detail behind the `Writing quality` section of `AGENTS.md`. It applies to every piece of prose an agent writes for this repository: Issue and Pull Request bodies, `docs/`, `SKILL.md`, commit message bodies, and Japanese conversational reports. Code, identifiers, commands, and quoted output are not prose and are exempt.

Read it in three passes. Check a draft against the machine-typical patterns, then against the readable-writing rules, then against the thresholds. Apply the convergence rule before calling anything a defect.

## Machine-typical patterns

### Vocabulary and phrasing

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
- **Anthropomorphism.** A document, a command, or a data set does not decide or speak, so giving it intent hides who or what actually acts.
  - Before: The catalog tells us which skills are published.
  - After: The `skills:` list in `CATALOG.yml` is the published inventory.

### Structure and rhythm

- **Uniform sentence length.** When every sentence runs to the same length the prose loses stress, and the reader cannot tell which sentence carries the load.
  - Before: The task runs the checks. The checks report the result. The result blocks the push.
  - After: The task runs the checks. A failure blocks the push, and the diagnostic names the check that failed.
- **Rule-of-three padding.** A polished triplet reads as complete whether or not the third item is real, so the reader accepts a filler item as a fact.
  - Before: The check is fast, reliable, and comprehensive.
  - After: The check runs in under a second and reads only tracked files.
- **Transition stacking.** `Furthermore`, `Moreover`, and `Additionally` announce a connection the sentences already have, and one connector per paragraph opening is already too many when the ideas are ordered.
  - Before: Furthermore, the hook reruns setup. Moreover, it records the result.
  - After: The hook reruns setup and records the result.
- **Em dash pile-up.** An em dash every second sentence turns every clause into an aside, so nothing reads as the main point.
  - Before: The task, run from the root — always from the root — validates the tree.
  - After: Run the task from the repository root. It validates the whole tree.
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

### Substance and stance

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

### Japanese register

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
- **曖昧なぼかし.** `一般的に〜とされています` and `〜と言われています` attribute a claim to nobody, so the reader cannot check it.
  - Before: 一般的に短い文が読みやすいとされています。
  - After: 本多勝一は、長い修飾語の境界に読点を打つよう定めています。

## Readable-writing rules

### Rules for both languages

### English rules

### Japanese rules

## Consistency and substance rules

## Thresholds

## The convergence rule

## Review checklist

## Sources
