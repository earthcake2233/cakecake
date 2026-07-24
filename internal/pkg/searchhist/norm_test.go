package searchhist

import (
	"testing"
)

func TestNorm_MixedCase(t *testing.T) {
	if got := Norm("HelloWorld"); got != "helloworld" {
		t.Fatalf("Norm(\"HelloWorld\") = %q, want \"helloworld\"", got)
	}
}

func TestNorm_WithSpaces(t *testing.T) {
	if got := Norm("hello world"); got != "helloworld" {
		t.Fatalf("Norm(\"hello world\") = %q, want \"helloworld\"", got)
	}
}

func TestNorm_LeadingTrailingSpaces(t *testing.T) {
	if got := Norm("  hello  "); got != "hello" {
		t.Fatalf("Norm(\"  hello  \") = %q, want \"hello\"", got)
	}
}

func TestNorm_ChineseCharacters(t *testing.T) {
	if got := Norm("B站 视频"); got != "b站视频" {
		t.Fatalf("Norm(\"B站 视频\") = %q, want \"b站视频\"", got)
	}
}

func TestNorm_EmptyString(t *testing.T) {
	if got := Norm(""); got != "" {
		t.Fatalf("Norm(\"\") = %q, want empty", got)
	}
}

func TestNorm_OnlySpaces(t *testing.T) {
	if got := Norm("   "); got != "" {
		t.Fatalf("Norm(\"   \") = %q, want empty", got)
	}
}

func TestNorm_MultipleInternalSpaces(t *testing.T) {
	if got := Norm("a   b   c"); got != "abc" {
		t.Fatalf("Norm(\"a   b   c\") = %q, want \"abc\"", got)
	}
}
