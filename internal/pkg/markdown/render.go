package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// TocEntry is a table-of-contents item for the article reader sidebar.
type TocEntry struct {
	ID       string `json:"id"`
	Level    int    `json:"level"`
	Text     string `json:"text"`
	TextHTML string `json:"text_html,omitempty"`
}

var (
	headingRe = regexp.MustCompile(`(?m)^(#{1,3})\s+(.+)$`)
	// \p{Han} = CJK unified ideographs (Go regexp / RE2, not JS \uXXXX)
	slugSafe  = regexp.MustCompile(`[^a-zA-Z0-9\p{Han}]+`)
	htmlTags  = regexp.MustCompile(`<[^>]*>`)
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	// 标题 id 由 buildToc + injectHeadingIDs 统一注入，勿用 AutoHeadingID（会与目录 id 不一致）
	goldmark.WithRendererOptions(html.WithHardWraps()),
)

var ugcPolicy = bluemonday.UGCPolicy()

// headingInlinePolicy allows safe inline markup inside heading lines (e.g. <font color>).
var headingInlinePolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("font", "b", "strong", "i", "em", "u", "s", "del", "span", "br")
	p.AllowAttrs("color").OnElements("font")
	p.AllowAttrs("class").Globally()
	return p
}()

var headingInnerRe = regexp.MustCompile(`<(h[1-3])(\s[^>]*)?>([^<]*)</(h[1-3])>`)

// Render converts Markdown to sanitized HTML and builds a TOC from h1–h3 headings.
func Render(bodyMD string) (html string, toc []TocEntry, err error) {
	bodyMD = strings.TrimSpace(bodyMD)
	if bodyMD == "" {
		return "", nil, nil
	}
	toc = buildToc(bodyMD)
	var buf bytes.Buffer
	if err := md.Convert([]byte(bodyMD), &buf); err != nil {
		return "", nil, err
	}
	raw := string(ugcPolicy.SanitizeBytes(buf.Bytes()))
	raw = injectHeadingIDs(raw, toc)
	raw = injectHeadingInnerHTML(raw, toc)
	return raw, toc, nil
}

var openHeadingRe = regexp.MustCompile(`<(h[1-3])(\s[^>]*)?>`)

func injectHeadingIDs(html string, toc []TocEntry) string {
	if len(toc) == 0 {
		return html
	}
	idx := 0
	return openHeadingRe.ReplaceAllStringFunc(html, func(s string) string {
		if idx >= len(toc) {
			return s
		}
		m := openHeadingRe.FindStringSubmatch(s)
		if len(m) < 2 {
			return s
		}
		tag := m[1]
		id := toc[idx].ID
		idx++
		return fmt.Sprintf(`<%s id="%s">`, tag, id)
	})
}

func injectHeadingInnerHTML(html string, toc []TocEntry) string {
	if len(toc) == 0 {
		return html
	}
	idx := 0
	return headingInnerRe.ReplaceAllStringFunc(html, func(s string) string {
		if idx >= len(toc) {
			return s
		}
		m := headingInnerRe.FindStringSubmatch(s)
		if len(m) < 5 {
			return s
		}
		tag := m[1]
		attrs := m[2]
		closeTag := m[4]
		inner := strings.TrimSpace(toc[idx].TextHTML)
		idx++
		if inner == "" {
			inner = m[3]
		}
		return fmt.Sprintf("<%s%s>%s</%s>", tag, attrs, inner, closeTag)
	})
}

func buildToc(bodyMD string) []TocEntry {
	var out []TocEntry
	used := map[string]int{}
	for _, m := range headingRe.FindAllStringSubmatch(bodyMD, -1) {
		if len(m) < 3 {
			continue
		}
		level := len(m[1])
		if level < 1 || level > 3 {
			continue
		}
		text := strings.TrimSpace(m[2])
		if text == "" {
			continue
		}
		textHTML := strings.TrimSpace(string(headingInlinePolicy.SanitizeBytes([]byte(text))))
		plain := strings.TrimSpace(htmlTags.ReplaceAllString(text, ""))
		if plain == "" {
			plain = textHTML
		}
		base := slugSafe.ReplaceAllString(plain, "-")
		base = strings.Trim(base, "-")
		if base == "" {
			base = "section"
		}
		id := base
		if n := used[id]; n > 0 {
			id = fmt.Sprintf("%s-%d", base, n+1)
		}
		used[base]++
		out = append(out, TocEntry{ID: id, Level: level, Text: plain, TextHTML: textHTML})
	}
	return out
}

// SlugPreview returns a short slug for display (cv id uses numeric id on API).
func SlugPreview(title string, maxRunes int) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = 32
	}
	if utf8.RuneCountInString(t) > maxRunes {
		t = string([]rune(t)[:maxRunes])
	}
	return slugSafe.ReplaceAllString(t, "-")
}
