package check

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/hidekitux/skills/internal/support"
)

const writingCoveragePath = "docs/validation-tiers.md"

var (
	writingTaskRE      = regexp.MustCompile(`(?i)\b(?:run|runs|execute|executes|executed|invoke|invokes|invoked)\s+` + "`" + `([a-z][a-z0-9-]*:[a-z0-9-]+)` + "`")
	writingWordRE      = regexp.MustCompile(`⟦code⟧|[\p{L}\p{N}]+(?:['’-][\p{L}\p{N}]+)*`)
	writingSentenceRE  = regexp.MustCompile(`(?:[。！？]|[.!?][*_)]*(?:\s|$))`)
	writingConnectorRE = regexp.MustCompile(`(?i)^(?:furthermore|moreover|additionally|また|さらに)\b`)
)

type writingFinding struct {
	file, rule, value string
	line              int
}

type writingCandidate struct {
	file, rule, value string
	line              int
}

type writingSentence struct {
	text string
	line int
}

type writingFileMetrics struct {
	paragraphs, connectorParagraphs int
	englishWords, emDashes          int
	englishLengths                  []int
	english, japanese               []writingSentence
}

func trackedMarkdown(root string) ([]string, error) {
	out, err := support.GitOutputIn(root, "ls-files", "--", "*.md")
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, filepath.ToSlash(line))
		}
	}
	sort.Strings(files)
	return files, nil
}

func maskWritingCode(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); {
		if text[i] != '`' {
			b.WriteByte(text[i])
			i++
			continue
		}
		end := strings.IndexByte(text[i+1:], '`')
		if end < 0 {
			b.WriteString(text[i:])
			break
		}
		b.WriteString(" ⟦code⟧ ")
		i += end + 2
	}
	return b.String()
}

func writingWords(text string) int { return len(writingWordRE.FindAllString(text, -1)) }

func writingJapanese(text string) bool {
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x30ff) || (r >= 0x3400 && r <= 0x9fff) {
			return true
		}
	}
	return false
}

func writingListItem(text string) bool {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "- ") || strings.HasPrefix(text, "* ") || strings.HasPrefix(text, "+ ") {
		return true
	}
	for i, r := range text {
		if !unicode.IsDigit(r) {
			return i > 0 && i+1 < len(text) && (text[i] == '.' || text[i] == ')') && text[i+1] == ' '
		}
	}
	return false
}

func writingExample(text string) bool {
	text = strings.TrimSpace(text)
	return strings.HasPrefix(text, "Before:") || strings.HasPrefix(text, "After:") ||
		strings.HasPrefix(text, "- Before:") || strings.HasPrefix(text, "- After:")
}

func writingSentences(text string, line int) []writingSentence {
	var result []writingSentence
	start := 0
	for _, match := range writingSentenceRE.FindAllStringIndex(text, -1) {
		end := match[1]
		sentence := strings.TrimSpace(text[start:end])
		if sentence != "" {
			result = append(result, writingSentence{text: sentence, line: line})
		}
		start = end
	}
	if tail := strings.TrimSpace(text[start:]); tail != "" {
		result = append(result, writingSentence{text: tail, line: line})
	}
	return result
}

func genuineWritingEnumeration(text string) bool {
	return strings.Count(text, ",") >= 2 || strings.Count(text, "、") >= 2 ||
		(strings.Contains(text, ", and ") && strings.Contains(text, " and "))
}

func writingParticles(text string) int {
	markers := []string{"して", "ため", "ので", "が、", "し、", "て、"}
	count := 0
	for _, marker := range markers {
		count += strings.Count(text, marker)
	}
	return count
}

func addWritingSentence(m *writingFileMetrics, sentence writingSentence, list bool, findings *[]writingFinding, candidates *[]writingCandidate, file string) {
	masked := maskWritingCode(sentence.text)
	words := writingWords(masked)
	if words == 0 {
		return
	}
	if writingJapanese(masked) && strings.ContainsAny(masked, "。！？") {
		m.japanese = append(m.japanese, sentence)
		chars := len([]rune(masked))
		if chars > 70 {
			if genuineWritingEnumeration(masked) {
				*candidates = append(*candidates, writingCandidate{file, "Japanese sentence length", fmt.Sprintf("%d characters", chars), sentence.line})
			} else {
				*findings = append(*findings, writingFinding{file, "Japanese sentence length", fmt.Sprintf("%d characters", chars), sentence.line})
			}
		}
		if particles := writingParticles(masked); particles > 2 {
			*findings = append(*findings, writingFinding{file, "Japanese connective particles", fmt.Sprintf("%d markers", particles), sentence.line})
		}
		return
	}
	m.english = append(m.english, sentence)
	if list {
		return
	}
	m.englishWords += words
	m.emDashes += strings.Count(masked, "—")
	m.englishLengths = append(m.englishLengths, words)
	if words > 45 {
		if genuineWritingEnumeration(masked) {
			*candidates = append(*candidates, writingCandidate{file, "English sentence length", fmt.Sprintf("%d words", words), sentence.line})
		} else {
			*findings = append(*findings, writingFinding{file, "English sentence length", fmt.Sprintf("%d words", words), sentence.line})
		}
	}
}

func scanWritingFile(root, file string) ([]writingFinding, []writingCandidate, []int, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return nil, nil, nil, err
	}
	lines := strings.Split(string(content), "\n")
	var findings []writingFinding
	var candidates []writingCandidate
	var metrics writingFileMetrics
	inFence, inFrontmatter := false, len(lines) > 0 && strings.TrimSpace(lines[0]) == "---"
	var paragraph []writingSentence
	var paragraphList bool
	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		metrics.paragraphs++
		if !paragraphList && writingConnectorRE.MatchString(strings.ToLower(strings.TrimSpace(paragraph[0].text))) {
			metrics.connectorParagraphs++
		}
		paragraph, paragraphList = nil, false
	}
	for number, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if inFrontmatter {
			if number > 0 && trimmed == "---" {
				inFrontmatter = false
			}
			flushParagraph()
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			flushParagraph()
			continue
		}
		if inFence || trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ">") ||
			strings.HasPrefix(trimmed, "|") || writingExample(trimmed) {
			flushParagraph()
			continue
		}
		list := writingListItem(trimmed)
		masked := maskWritingCode(trimmed)
		for _, match := range writingTaskRE.FindAllStringSubmatchIndex(trimmed, -1) {
			if !strings.Contains(trimmed[match[0]:match[1]], "mise run") {
				findings = append(findings, writingFinding{file, "canonical mise task command", "bare executable task reference", number + 1})
			}
		}
		sentences := writingSentences(masked, number+1)
		if !list {
			if len(paragraph) == 0 || number+1 != paragraph[len(paragraph)-1].line+1 {
				flushParagraph()
			}
			paragraph = append(paragraph, sentences...)
			paragraphList = false
		}
		for _, sentence := range sentences {
			addWritingSentence(&metrics, sentence, list, &findings, &candidates, file)
		}
	}
	flushParagraph()
	if metrics.paragraphs > 0 && metrics.connectorParagraphs*4 > metrics.paragraphs {
		findings = append(findings, writingFinding{file, "paragraph-opening connector rate", fmt.Sprintf("%d/%d paragraphs", metrics.connectorParagraphs, metrics.paragraphs), 1})
	}
	if metrics.englishWords > 0 && metrics.emDashes*1000 > metrics.englishWords*10 {
		findings = append(findings, writingFinding{file, "English em-dash density", fmt.Sprintf("%d per 1,000 words", metrics.emDashes*1000/metrics.englishWords), 1})
	}
	return findings, candidates, metrics.englishLengths, nil
}

func CheckWritingQuality(root string, out, errOut io.Writer) int {
	files, err := trackedMarkdown(root)
	if err != nil {
		fmt.Fprintf(errOut, "Writing-quality check failed: %v\n", err)
		return 1
	}
	var findings []writingFinding
	var candidates []writingCandidate
	var englishLengths []int
	for _, file := range files {
		fileFindings, fileCandidates, fileLengths, scanErr := scanWritingFile(root, file)
		if scanErr != nil {
			findings = append(findings, writingFinding{file, "readability", scanErr.Error(), 1})
			continue
		}
		findings = append(findings, fileFindings...)
		candidates = append(candidates, fileCandidates...)
		englishLengths = append(englishLengths, fileLengths...)
	}
	if len(englishLengths) >= 10 {
		mean := 0.0
		for _, length := range englishLengths {
			mean += float64(length)
		}
		mean /= float64(len(englishLengths))
		variance := 0.0
		for _, length := range englishLengths {
			variance += math.Pow(float64(length)-mean, 2)
		}
		coefficient := math.Sqrt(variance/float64(len(englishLengths))) / mean
		if coefficient < 0.5 {
			findings = append(findings, writingFinding{"tracked Markdown", "English sentence-length variance", fmt.Sprintf("%.2f", coefficient), 1})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].file+fmt.Sprint(findings[i].line) < findings[j].file+fmt.Sprint(findings[j].line)
	})
	if len(findings) > 0 {
		fmt.Fprintf(errOut, "Writing-quality check failed; see %s for the coverage boundary.\n", writingCoveragePath)
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s:%d: %s (%s)\n", finding.file, finding.line, finding.rule, finding.value)
		}
		return 1
	}
	fmt.Fprintf(out, "Writing-quality check passed for %d tracked Markdown file(s); see %s for measured rules and human-only review.\n", len(files), writingCoveragePath)
	for _, candidate := range candidates {
		fmt.Fprintf(out, "- review candidate %s:%d: %s (%s)\n", candidate.file, candidate.line, candidate.rule, candidate.value)
	}
	return 0
}
