package agent

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	mdFenceRe      = regexp.MustCompile("(?s)```.*?```")
	mdLinkRe       = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	mdLinePrefixRe = regexp.MustCompile("(?m)^[#>*+\\-]\\s*")
	mdFenceLineRe  = regexp.MustCompile("(?m)^(\\s*)(`{3,})(.*)$")
	mdStrayFenceRe = regexp.MustCompile("`{3,}[a-zA-Z0-9_+-]*")
	mdDismissRe    = regexp.MustCompile(
		"没有(相关|什么)?(的)?(教程|内容|视频|投稿)" +
			"|确实没有|没有找到|没找到|暂无|暂时没有|无关|不相关|没关系" +
			"|八竿子打不着|没搜到|没有搜到|只有(动画|音乐|视频)",
	)
)

// plainTextPreview strips common Markdown syntax so the conversation-list
// preview never exposes raw formatting (bold markers, list bullets, code
// fences, links) to the user.
func plainTextPreview(content string) string {
	text := content
	text = mdFenceRe.ReplaceAllString(text, " ")
	text = mdLinkRe.ReplaceAllString(text, "$1")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "`", "")
	text = mdLinePrefixRe.ReplaceAllString(text, "")
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

func mergeContinuation(partial string, continuation string) string {
	p := strings.TrimSpace(partial)
	c := strings.TrimSpace(continuation)
	if p == "" {
		return c
	}
	if c == "" {
		return p
	}
	np := strings.Join(strings.Fields(p), " ")
	nc := strings.Join(strings.Fields(c), " ")
	nr := []rune(np)
	cr := []rune(nc)
	maxOverlap := len(nr)
	if l := len(cr); l < maxOverlap {
		maxOverlap = l
	}
	best := 0
	for n := maxOverlap; n >= 1; n-- {
		if strings.HasSuffix(string(nr), string(cr[:n])) {
			best = n
			break
		}
	}
	if best == 0 {
		c = dropSeamDuplicateLines(p, c)
		return p + "\n" + c
	}
	return p + string([]rune(c)[normOverlapCut(c, best):])
}

// dropSeamDuplicateLines merges the seam when the continuation re-emits the
// partial's trailing line: either verbatim (drop the duplicate) or by
// restarting the whole line from its beginning (drop the partial's incomplete
// tail line and keep the continuation's full line).
func dropSeamDuplicateLines(p string, c string) string {
	pl := strings.Split(p, "\n")
	cl := strings.Split(c, "\n")
	for len(cl) > 0 && len(pl) > 0 {
		first := strings.TrimSpace(cl[0])
		last := strings.TrimSpace(pl[len(pl)-1])
		if first == "" {
			cl = cl[1:]
			continue
		}
		if first == last {
			cl = cl[1:]
			pl = pl[:len(pl)-1]
			continue
		}
		if last != "" && utf8.RuneCountInString(last) >= 4 && strings.HasPrefix(first, last) {
			// The model restarted the whole line from its beginning: keep the
			// continuation's complete line and drop the partial's half line.
			pl = pl[:len(pl)-1]
			return strings.Join(append(pl, cl...), "\n")
		}
		break
	}
	return strings.Join(cl, "\n")
}

// normOverlapCut maps a whitespace-normalized overlap length back to a rune
// offset in the original string.
func normOverlapCut(s string, target int) int {
	runes := []rune(s)
	acc := 0
	prevSpace := false
	for i, r := range runes {
		sp := unicode.IsSpace(r)
		if sp {
			if !prevSpace {
				acc++
			}
			prevSpace = true
		} else {
			acc++
			prevSpace = false
		}
		if acc >= target {
			return i + 1
		}
	}
	return len(runes)
}

// dedupeConsecutiveLines removes exact consecutive duplicate lines (the model
// sometimes re-emits a block when continuing).
func dedupeConsecutiveLines(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for i, ln := range lines {
		if i > 0 && ln == lines[i-1] {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// generateSuggestions asks the model for 3 short follow-up questions based on
// the reply, so the UI can render contextual suggestion chips. Fail-soft:

func partialEndsInsideCodeFence(partial string) bool {
	fences := 0
	for _, ln := range strings.Split(partial, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") {
			fences++
		}
	}
	return fences%2 == 1
}

// normalizeMarkdownFences balances fenced code blocks: every fence line is
// normalized to exactly three backticks, and an unclosed fence is closed at
// the end so the rendered reply never breaks the chat layout.
func normalizeMarkdownFences(text string) string {
	if !strings.Contains(text, "`") {
		return text
	}
	lines := strings.Split(text, "\n")
	open := false
	for i, ln := range lines {
		if !mdFenceLineRe.MatchString(ln) {
			// Stray fence markers mid-line (e.g. the model re-emitted a fence
			// at a continuation seam) are not valid markdown; strip them.
			if strings.Contains(ln, "```") {
				lines[i] = mdStrayFenceRe.ReplaceAllString(ln, "")
			}
			continue
		}
		m := mdFenceLineRe.FindStringSubmatch(ln)
		if open && strings.TrimSpace(m[3]) != "" {
			// A language-tagged fence while already inside a code block is a
			// continuation-seam artifact (the model re-emitted the opener).
			// Drop the stray line; only a bare ``` legitimately closes.
			lines[i] = ""
			continue
		}
		lines[i] = m[1] + "```" + m[3]
		open = !open
	}
	if open {
		lines = append(lines, "```")
	}
	return strings.Join(lines, "\n")
}
