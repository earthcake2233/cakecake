package markdown

import (
	"strings"
	"testing"
)

func TestSanitizeHeadingInlineFont(t *testing.T) {
	raw := `<font color="#3498db">一、引言</font>`
	got := string(headingInlinePolicy.SanitizeBytes([]byte(raw)))
	t.Logf("sanitized=%q", got)
	if !strings.Contains(got, "font") || !strings.Contains(got, "3498db") {
		t.Fatalf("font tag not preserved: %q", got)
	}
}

func TestRenderHeadingWithFontInToc(t *testing.T) {
	body := "## <font color=\"#3498db\">一、引言</font>\n\n正文"
	html, toc, err := Render(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(toc) != 1 {
		t.Fatalf("toc len=%d", len(toc))
	}
	if toc[0].Text != "一、引言" {
		t.Fatalf("toc plain text=%q", toc[0].Text)
	}
	if !strings.Contains(toc[0].TextHTML, "font") {
		t.Fatalf("toc text_html=%q", toc[0].TextHTML)
	}
	if !strings.Contains(html, "3498db") {
		t.Fatalf("html missing color: %s", html)
	}
	if !strings.Contains(html, `id="`+toc[0].ID+`"`) {
		t.Fatalf("html missing id %q", toc[0].ID)
	}
}
