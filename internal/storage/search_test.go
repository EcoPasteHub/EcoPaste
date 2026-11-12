package storage

import "testing"

func TestKeywordClauseUsesFTSForLongTokens(t *testing.T) {
	got, ok := keywordClause(`foo "bar`)
	if !ok {
		t.Fatal("expected FTS")
	}
	want := `"foo"* """bar"*`
	if got != want {
		t.Fatalf("keywordClause() = %q, want %q", got, want)
	}
}

func TestKeywordClauseFallsBackForShortToken(t *testing.T) {
	if _, ok := keywordClause("ab long"); ok {
		t.Fatal("short token must use LIKE")
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`a_b%c\d`); got != `a\_b\%c\\d` {
		t.Fatalf("escapeLike() = %q", got)
	}
}
