package handler

import "testing"

func TestValidateVideoDraftContent(t *testing.T) {
	if !validateVideoDraftContent("标题", "", false) {
		t.Fatal("title only should pass")
	}
	if !validateVideoDraftContent("", "简介", false) {
		t.Fatal("desc only should pass")
	}
	if !validateVideoDraftContent("", "", true) {
		t.Fatal("file only should pass")
	}
	if validateVideoDraftContent("", "", false) {
		t.Fatal("empty should fail")
	}
	if validateVideoDraftContent(string(make([]rune, 81)), "", false) {
		t.Fatal("title too long should fail")
	}
}
