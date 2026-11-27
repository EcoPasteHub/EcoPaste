package clipboard

import (
	"strings"
	"testing"
)

func TestSummarizeTruncatesAtCharacterBoundary(t *testing.T) {
	prefix := strings.Repeat("字", SummaryMaxChars-2)
	for _, tc := range []struct{ input, want string }{
		{"  \n", ""},
		{" " + prefix + "🙂a\n ", prefix + "🙂a"},
		{prefix + "🙂ab", prefix + "🙂a"},
		{prefix + "\xff\xfe", prefix + "\xff\xfe"},
		{prefix + "\xff\xfeextra", prefix + "\ufffd\ufffd"},
	} {
		got := summarize(tc.input)
		if tc.want == "" && got == nil {
			continue
		}
		if got == nil || *got != tc.want {
			t.Fatalf("summarize did not preserve the expected prefix")
		}
	}
}

func BenchmarkSummarizeLargeText(b *testing.B) {
	plain := strings.Repeat("字", 256*1024)
	b.ReportAllocs()
	for b.Loop() {
		_ = summarize(plain)
	}
}
