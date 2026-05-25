package markdown

import (
	"strings"
	"testing"
)

func TestRenderHeadingIDsMatchToc(t *testing.T) {
	body := "# 一级\n\n## 二级标题\n\n### 小节\n\n## 另一个"
	html, toc, err := Render(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(toc) != 4 {
		t.Fatalf("toc len=%d", len(toc))
	}
	for _, e := range toc {
		if !strings.Contains(html, `id="`+e.ID+`"`) {
			t.Fatalf("html missing id %q\nhtml=%s", e.ID, html)
		}
	}
}
