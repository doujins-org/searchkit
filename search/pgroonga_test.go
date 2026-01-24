package search

import "testing"

func TestBuildPGroongaTypeaheadQuery(t *testing.T) {
	if got := buildPGroongaTypeaheadQuery(""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := buildPGroongaTypeaheadQuery("  a  "); got != "a*" {
		t.Fatalf("expected %q, got %q", "a*", got)
	}
	if got := buildPGroongaTypeaheadQuery("blue archive"); got != "blue* archive*" {
		t.Fatalf("expected %q, got %q", "blue* archive*", got)
	}
	if got := buildPGroongaTypeaheadQuery("tok*"); got != "tok*" {
		t.Fatalf("expected %q, got %q", "tok*", got)
	}
}

func TestNormalizePGroongaScore(t *testing.T) {
	if got := NormalizePGroongaScore(0, 10); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
	if got := NormalizePGroongaScore(-1, 10); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
	got := NormalizePGroongaScore(10, 10)
	if got < 0.49 || got > 0.51 {
		t.Fatalf("expected ~0.5, got %v", got)
	}
	got = NormalizePGroongaScore(10, 0)
	if got <= 0 || got >= 1 {
		t.Fatalf("expected in (0,1), got %v", got)
	}
}

func TestBuildPGroongaSQL(t *testing.T) {
	sql, args, _, err := buildPGroongaSQL("doujins", "doujins", []string{"gallery"}, "", nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sql == "" {
		t.Fatalf("expected sql")
	}
	if _, ok := args["entity_types"]; !ok {
		t.Fatalf("expected entity_types arg")
	}
	if _, ok := args["language"]; !ok {
		t.Fatalf("expected language arg placeholder")
	}
	if _, ok := args["q"]; !ok {
		t.Fatalf("expected q arg placeholder")
	}
	if _, ok := args["limit"]; !ok {
		t.Fatalf("expected limit arg placeholder")
	}
}
