package main

import (
	"strings"
	"testing"
)

func TestCleanReleaseNotes(t *testing.T) {
	in := "- 新增:\n<img src=\"https://example.com/a.png\" alt=\"图\" style=\"width: 100%\" />\n![shot](https://example.com/b.png)\n正文\n\n\n结束"
	out := cleanReleaseNotes(in)
	if strings.Contains(out, "<img") || strings.Contains(out, "![") {
		t.Fatalf("HTML/image left in notes: %q", out)
	}
	if !strings.Contains(out, "正文") || !strings.Contains(out, "结束") {
		t.Fatalf("text lost: %q", out)
	}
	if strings.Contains(out, "\n\n\n") {
		t.Fatalf("blank lines not collapsed: %q", out)
	}
}
