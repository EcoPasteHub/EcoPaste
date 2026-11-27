//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestHighlightTokensDedupesAndSplits(t *testing.T) {
	tokens := highlightTokens("  Hello   hello WORLD  ")
	if len(tokens) != 2 {
		t.Fatalf("tokens = %#v, want 2 unique tokens", tokens)
	}
	if tokens[0] != "Hello" || tokens[1] != "WORLD" {
		t.Fatalf("tokens = %#v", tokens)
	}
	if got := highlightTokens("   "); got != nil {
		t.Fatalf("blank query tokens = %#v, want nil", got)
	}
}

func TestHighlightRanges(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		tokens []string
		want   []NSRange
	}{
		{
			name:   "case insensitive",
			text:   "Hello World",
			tokens: []string{"WORLD"},
			want:   []NSRange{{Location: 6, Length: 5}},
		},
		{
			name:   "all occurrences",
			text:   "ab AB ab",
			tokens: []string{"ab"},
			want:   []NSRange{{Location: 0, Length: 2}, {Location: 3, Length: 2}, {Location: 6, Length: 2}},
		},
		{
			name:   "multiple tokens merge into one run",
			text:   "abc",
			tokens: []string{"ab", "bc"},
			want:   []NSRange{{Location: 0, Length: 3}},
		},
		{
			name:   "cjk uses utf-16 offsets",
			text:   "你好世界",
			tokens: []string{"世界"},
			want:   []NSRange{{Location: 2, Length: 2}},
		},
		{
			name:   "emoji counts as two utf-16 units",
			text:   "a😀b",
			tokens: []string{"b"},
			want:   []NSRange{{Location: 3, Length: 1}},
		},
		{
			name:   "no match",
			text:   "hello",
			tokens: []string{"zzz"},
			want:   nil,
		},
		{
			name:   "token longer than text",
			text:   "hi",
			tokens: []string{"hello"},
			want:   nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := highlightRanges(tc.text, tc.tokens)
			if len(got) != len(tc.want) {
				t.Fatalf("ranges = %#v, want %#v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("range[%d] = %#v, want %#v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestHighlightedRowStringAppliesBackgroundToMatches(t *testing.T) {
	font := objc.ID(classNSFont).Send(sel("systemFontOfSize:"), 12.0)
	color := objc.ID(classNSColor).Send(sel("labelColor"))
	attr := highlightedRowString("Hello World", []string{"World"}, font, color, objc.ID(0))
	if attr == 0 {
		t.Fatal("highlightedRowString returned nil")
	}
	if n := objc.Send[uint](attr, sel("length")); n != 11 {
		t.Fatalf("length = %d, want 11", n)
	}

	var highlighted []NSRange
	block := objc.NewBlock(func(_ objc.Block, value objc.ID, r NSRange, _ *bool) {
		if value != 0 {
			highlighted = append(highlighted, r)
		}
	})
	attr.Send(sel("enumerateAttribute:inRange:options:usingBlock:"),
		nsString("NSBackgroundColor"), NSRange{Length: 11}, uint64(0), block)
	block.Release()

	if len(highlighted) != 1 {
		t.Fatalf("highlighted ranges = %#v, want exactly the match", highlighted)
	}
	if highlighted[0] != (NSRange{Location: 6, Length: 5}) {
		t.Fatalf("highlighted range = %#v, want {6 5}", highlighted[0])
	}
}

func TestHighlightedRowStringReturnsNilWithoutMatch(t *testing.T) {
	font := objc.ID(classNSFont).Send(sel("systemFontOfSize:"), 12.0)
	color := objc.ID(classNSColor).Send(sel("labelColor"))
	if attr := highlightedRowString("Hello", []string{"zzz"}, font, color, objc.ID(0)); attr != 0 {
		t.Fatalf("attr = %v, want nil for no match", attr)
	}
}
