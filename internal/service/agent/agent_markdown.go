package agent

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	mdFenceRe      = regexp.MustCompile("(?s)```.*?```")
	mdLinkRe       = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	mdLinePrefixRe = regexp.MustCompile(`(?m)^[#>*+\-]\s*`)
	mdFenceLineRe  = regexp.MustCompile("(?m)^(\\s*)(`{3,})(.*)$")
	mdStrayFenceRe = regexp.MustCompile("`{3,}[a-zA-Z0-9_+-]*")
	// mdItemDismissRe marks a sentence that dismisses the video it mentions
	// (e.g. "搜到了《溯》，但和编程无关"). Only items cited inside such a
	// sentence are dropped; other recommendations stay.
	mdItemDismissRe = regexp.MustCompile(
		"无关|不相关|没关系|八竿子打不着|没搜到|没有搜到|没找到|没有找到" +
			"|暂时没有|没有相关|暂无|只有(动画|音乐|视频)",
	)
	// displayMarkerRe captures the model-declared display list
	// (【展示】search_videos#23,get_video_detail#24); displayMarkerLineRe
	// removes the whole marker line before the reply is persisted.
	displayMarkerRe     = regexp.MustCompile(`【展示】\s*([^\n【】]*)`)
	displayMarkerLineRe = regexp.MustCompile(`(?m)^[^\n]*【展示】[^\n]*$`)
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

// stitchContinuation merges the stopped partial with the model's continuation
// using ONLY exact, deterministic rules (no fuzzy overlap guessing that could
// split code blocks):
//  1. the whole partial repeated verbatim at the head of the continuation is
//     dropped;
//  2. a verbatim duplicate of the partial's last line is dropped;
//  3. an incomplete partial tail line that the model restarted from its
//     beginning is replaced by the continuation's complete line;
//  4. a re-emitted code-fence opener right after an unclosed fence is dropped.
//
// It returns the stitched full reply and the clean tail to stream to the
// client (the client never sees the unstitched seam).
func stitchContinuation(partial string, continuation string) (full string, tail string) {
	p := strings.TrimSpace(partial)
	c := strings.TrimSpace(continuation)
	if p == "" {
		return c, c
	}
	if c == "" {
		return p, ""
	}
	// Rule 1: the model re-emitted the entire partial.
	if strings.HasPrefix(c, p) {
		rest := strings.TrimPrefix(c, p)
		// Only treat it as a whole-partial repeat when the partial ends at a
		// line boundary in the continuation; otherwise the continuation is
		// merely restarting the same line, which rule 3 handles below.
		if rest == "" || strings.HasPrefix(rest, "\n") {
			return c, strings.TrimSpace(rest)
		}
	}
	pl := strings.Split(p, "\n")
	cl := strings.Split(c, "\n")
	// Rule 4: a re-emitted opener right after an unclosed fence.
	if partialEndsInsideCodeFence(p) && len(cl) > 0 && strings.HasPrefix(strings.TrimSpace(cl[0]), "```") {
		cl = cl[1:]
	}
	for len(cl) > 0 && strings.TrimSpace(cl[0]) == "" {
		cl = cl[1:]
	}
	changed := false
	restartPrefix := ""
	for len(cl) > 0 && len(pl) > 0 {
		first := strings.TrimSpace(cl[0])
		last := strings.TrimSpace(pl[len(pl)-1])
		if first == "" {
			cl = cl[1:]
			changed = true
			continue
		}
		if first == last {
			// Rule 2: verbatim line-level repeat.
			cl = cl[1:]
			pl = pl[:len(pl)-1]
			changed = true
			continue
		}
		if last != "" && utf8.RuneCountInString(last) >= 4 && strings.HasPrefix(first, last) {
			// Rule 3: the model restarted the whole line from its beginning.
			restartPrefix = last
			pl = pl[:len(pl)-1]
			changed = true
			break
		}
		break
	}
	body := strings.TrimSpace(strings.Join(cl, "\n"))
	if body == "" {
		return strings.TrimSpace(p), ""
	}
	if changed {
		full = strings.TrimSpace(strings.Join(append(pl, cl...), "\n"))
	} else {
		full = strings.TrimSpace(p + "\n" + body)
	}
	full = dedupeConsecutiveLines(full)
	full = normalizeMarkdownFences(full)
	// The tail must only contain text the user has NOT seen in the draft yet:
	// seam-consumed lines are visible already, and a rule-3 restarted line only
	// contributes its new suffix.
	tailLines := cl
	if restartPrefix != "" && len(tailLines) > 0 {
		trimmed := strings.TrimSpace(tailLines[0])
		if strings.HasPrefix(trimmed, restartPrefix) {
			rest := strings.TrimPrefix(trimmed, restartPrefix)
			if rest == "" {
				tailLines = tailLines[1:]
			} else {
				tailLines = append([]string{rest}, tailLines[1:]...)
			}
		}
	}
	tail = strings.TrimSpace(strings.Join(tailLines, "\n"))
	return full, tail
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
