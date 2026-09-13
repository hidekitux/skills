package check

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/support"
)

func writeWritingFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func isolatedWritingGit(root string, args ...string) *exec.Cmd {
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	command.Env = support.GitEnv()
	return command
}

func writingGitRepo(t *testing.T, files map[string]string, tracked ...string) string {
	t.Helper()
	root := t.TempDir()
	if err := isolatedWritingGit(root, "init", "-q").Run(); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		writeWritingFile(t, root, name, content)
	}
	args := append([]string{"add"}, tracked...)
	if err := isolatedWritingGit(root, args...).Run(); err != nil {
		t.Fatal(err)
	}
	config := [][]string{{"config", "user.email", "test.invalid"}, {"config", "user.name", "Test"}}
	for _, values := range config {
		if err := isolatedWritingGit(root, values...).Run(); err != nil {
			t.Fatal(err)
		}
	}
	if err := isolatedWritingGit(root, "commit", "-qm", "fixture").Run(); err != nil {
		t.Fatal(err)
	}
	return root
}

func runWriting(t *testing.T, root string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckWritingQuality(root, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestWritingQualityUsesTrackedMarkdownOnly(t *testing.T) {
	long := "This untracked sentence contains enough ordinary words to exceed the writing threshold without being a genuine enumeration or a deliberate example."
	root := writingGitRepo(t, map[string]string{
		"README.md": "A short tracked sentence.\n",
		"notes.md":  long + "\n",
	}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 0 {
		t.Fatalf("expected untracked Markdown to be ignored, got %d: %s", code, errOut)
	}
}

func TestWritingQualityExcludesMarkdownStructureAndExamples(t *testing.T) {
	long := "This sentence is intentionally very long and contains many words that would exceed the threshold if the parser treated this excluded structure as running prose."
	content := "---\ndescription: " + long + "\n---\n" +
		"# " + long + "\n\n" +
		"- " + long + "\n\n" +
		"| " + long + " |\n| --- |\n\n" +
		"```text\n" + long + "\nRun `check:all`.\n```\n\n" +
		"> " + long + "\n\n" +
		"- Before: " + long + "\n- After: " + long + "\n"
	root := writingGitRepo(t, map[string]string{"README.md": content}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 0 {
		t.Fatalf("expected structural exclusions to pass, got %d: %s", code, errOut)
	}
}

func TestWritingQualityTreatsInlineCodeAsOneWord(t *testing.T) {
	content := "Run `git worktree add -b issue/<number> <path> origin/main` before you inspect the branch.\n"
	root := writingGitRepo(t, map[string]string{"README.md": content}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 0 {
		t.Fatalf("expected inline code to count as one word, got %d: %s", code, errOut)
	}
}

func TestWritingQualityTreatsJapaneseInlineCodeAsOneCharacter(t *testing.T) {
	within := strings.Repeat("あ", 68) + "`check:all`。\n"
	root := writingGitRepo(t, map[string]string{"README.md": within}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 0 {
		t.Fatalf("expected Japanese inline code to count as one character, got %d: %s", code, errOut)
	}

	over := strings.Repeat("あ", 69) + "`check:all`。\n"
	root = writingGitRepo(t, map[string]string{"README.md": over}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 1 || !strings.Contains(errOut, "Japanese sentence length") {
		t.Fatalf("expected Japanese length boundary failure, got %d: %s", code, errOut)
	}
}

func TestWritingQualityRejectsMeasuredViolations(t *testing.T) {
	longEnglish := "This sentence has enough ordinary words to exceed the forty five word limit while avoiding commas and list structure so the checker must report it as an unambiguous sentence length violation instead of treating the passage as a genuine enumeration for review and human classification by a reader who checks the exact measured threshold."
	longJapanese := "この文章は列挙ではなく一つの説明を長く続けるために書いてあり読者が一文の中で多くの情報を保持する必要がある状態を明確に再現しさらに説明を続けて文章の長さが上限を超えることを確認します。"
	particles := "検証が失敗すると原因を直すので再実行しますが、その結果を確認してから記録します。"
	emDash := strings.Repeat("A short sentence — with an aside. ", 4)
	flat := strings.Repeat("Short words form one line. ", 10)
	connector := "Furthermore, this paragraph opens with a formal connector.\n\nMoreover, this paragraph opens with another formal connector.\n\nA third paragraph states a plain result.\n\nA fourth paragraph states another result.\n"
	japaneseConnector := "また、この段落は接続詞で始まります。\n\nさらに、この段落も接続詞で始まります。\n\nまた、三つ目の段落です。\n\nさらに、四つ目の段落です。\n"
	cases := []struct {
		name, content, rule string
	}{
		{"English sentence length", longEnglish, "English sentence length"},
		{"Japanese sentence length", longJapanese, "Japanese sentence length"},
		{"Japanese connective particles", particles, "Japanese connective particles"},
		{"English em-dash density", emDash, "English em-dash density"},
		{"English sentence variance", flat, "English sentence-length variance"},
		{"paragraph connector rate", connector, "paragraph-opening connector rate"},
		{"Japanese paragraph connector rate", japaneseConnector, "paragraph-opening connector rate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writingGitRepo(t, map[string]string{"README.md": tc.content}, "README.md")
			if code, _, errOut := runWriting(t, root); code != 1 || !strings.Contains(errOut, tc.rule) {
				t.Fatalf("expected %s failure, got %d: %s", tc.rule, code, errOut)
			}
		})
	}
}

func TestWritingQualityDoesNotTreatJapaneseConjunctionAsConnector(t *testing.T) {
	content := "または、別の方法を使います。\n\nまたは、別の方法を使います。\n\nまたは、別の方法を使います。\n\nまたは、別の方法を使います。\n"
	root := writingGitRepo(t, map[string]string{"README.md": content}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 0 || errOut != "" {
		t.Fatalf("expected Japanese conjunctions to pass, got %d: %s", code, errOut)
	}
}

func TestWritingQualityCountsUnorderedListConnectors(t *testing.T) {
	content := "- Furthermore, this item states a result.\n- Moreover, this item states a result.\n- Additionally, this item states a result.\n- Furthermore, this item states a result.\n"
	root := writingGitRepo(t, map[string]string{"README.md": content}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 1 || !strings.Contains(errOut, "paragraph-opening connector rate") {
		t.Fatalf("expected unordered list connectors to fail, got %d: %s", code, errOut)
	}
}

func TestWritingQualityExcludesNumberedProcedureConnectors(t *testing.T) {
	content := "1. Furthermore, this procedure step states a result.\n2. Moreover, this procedure step states a result.\n3. Additionally, this procedure step states a result.\n4. Furthermore, this procedure step states a result.\n"
	root := writingGitRepo(t, map[string]string{"README.md": content}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 0 || errOut != "" {
		t.Fatalf("expected numbered procedure connectors to pass, got %d: %s", code, errOut)
	}
}

func TestWritingQualityCountsListItemsForEmDashDensity(t *testing.T) {
	var content strings.Builder
	for i := 0; i < 10; i++ {
		content.WriteString("- One two three four five — six seven eight nine ten.\n")
	}
	root := writingGitRepo(t, map[string]string{"README.md": content.String()}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 1 || !strings.Contains(errOut, "English em-dash density") {
		t.Fatalf("expected list em dashes to fail, got %d: %s", code, errOut)
	}
}

func TestWritingQualityExcludesReferenceListSeparatorsFromEmDashDensity(t *testing.T) {
	var content strings.Builder
	for i := 0; i < 10; i++ {
		content.WriteString("- [Reference](https://example) — describes a source entry.\n")
	}
	root := writingGitRepo(t, map[string]string{"README.md": content.String()}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 0 || errOut != "" {
		t.Fatalf("expected reference separators to be excluded, got %d: %s", code, errOut)
	}
}

func TestWritingQualityCountsRhetoricalEmDashesInReferenceEntries(t *testing.T) {
	var content strings.Builder
	for i := 0; i < 10; i++ {
		content.WriteString("- [Reference](https://example) — describes a source — with a caveat.\n")
	}
	root := writingGitRepo(t, map[string]string{"README.md": content.String()}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 1 || !strings.Contains(errOut, "English em-dash density") {
		t.Fatalf("expected rhetorical em dashes to fail, got %d: %s", code, errOut)
	}
}

func TestWritingQualityReportsGenuineEnumerationCandidates(t *testing.T) {
	english := "This sentence names the tracked file, the command, the measured value, the exclusion, the human review boundary, the failure behavior, and the candidate rule, while retaining the complete enumeration because each item carries a distinct fact for the reader and removing one would hide a required decision."
	japanese := "この文は、入力、出力、失敗条件、例外、確認方法、対象範囲、判定結果、除外条件をすべて示す必要があるため、短く分割すると情報が失われる本当の列挙です。"
	root := writingGitRepo(t, map[string]string{"README.md": english + "\n" + japanese + "\n"}, "README.md")
	if code, out, errOut := runWriting(t, root); code != 0 || errOut != "" {
		t.Fatalf("expected candidates to pass, got %d: %s", code, errOut)
	} else if !strings.Contains(out, "review candidate") || !strings.Contains(out, "sentence length") {
		t.Fatalf("expected candidate diagnostics, got: %s", out)
	}
}

func TestWritingQualityDistinguishesExecutableCommandsFromTaskNames(t *testing.T) {
	good := "The `check:all` task is the repository aggregate. Run `mise run check:all` after editing.\n\n```sh\nRun `check:all`.\n```\n"
	root := writingGitRepo(t, map[string]string{"README.md": good}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 0 {
		t.Fatalf("expected declarations and canonical commands to pass, got %d: %s", code, errOut)
	}
	bad := fmt.Sprintf("Run `%s`.\n", "check:all")
	root = writingGitRepo(t, map[string]string{"README.md": bad}, "README.md")
	if code, _, errOut := runWriting(t, root); code != 1 || !strings.Contains(errOut, "canonical mise task command") {
		t.Fatalf("expected bare executable task to fail, got %d: %s", code, errOut)
	}
}

func TestWritingQualityLeavesSemanticJudgmentToReview(t *testing.T) {
	content := "Delve into the multifaceted realm. The important conclusion appears after several details.\n"
	root := writingGitRepo(t, map[string]string{"README.md": content}, "README.md")
	if code, out, errOut := runWriting(t, root); code != 0 || errOut != "" {
		t.Fatalf("expected semantic prose choices to remain human-only, got %d: %s", code, errOut)
	} else if !strings.Contains(out, "human-only review") {
		t.Fatalf("expected human-review boundary in success output: %s", out)
	}
}
