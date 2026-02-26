package searchkit

import (
	"context"
	"strings"
	"testing"
)

func TestResolveLanguageModes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		language  string
		mode      LanguageMode
		want      []string
		expectErr bool
	}{
		{
			name:     "default exact mode",
			language: "es",
			mode:     "",
			want:     []string{"es"},
		},
		{
			name:     "exact mode explicit",
			language: "ja",
			mode:     LanguageModeExact,
			want:     []string{"ja"},
		},
		{
			name:     "fallback appends english",
			language: "es",
			mode:     LanguageModeFallbackEnglish,
			want:     []string{"es", "en"},
		},
		{
			name:     "fallback english dedupes",
			language: "en",
			mode:     LanguageModeFallbackEnglish,
			want:     []string{"en"},
		},
		{
			name:      "invalid mode",
			language:  "en",
			mode:      LanguageMode("invalid"),
			expectErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveLanguageModes(tc.language, tc.mode)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for mode %q", tc.mode)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("resolveLanguageModes(...) len=%d; want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("resolveLanguageModes(...) = %v; want %v", got, tc.want)
				}
			}
		})
	}
}

func TestClientSearch_InvalidLanguageMode(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientConfig{
		Pool:         newTestPool(t),
		Schema:       "test",
		DefaultModel: "model",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Search(context.Background(), "two factor", SearchOptions{
		LanguageMode:       LanguageMode("invalid"),
		Mode:               SearchModeLexical,
		LexicalEntityTypes: []string{"gallery"},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid SearchOptions.LanguageMode") {
		t.Fatalf("expected invalid SearchOptions.LanguageMode error, got: %v", err)
	}
}

func TestClientTypeahead_InvalidLanguageMode(t *testing.T) {
	t.Parallel()

	client, err := NewClient(ClientConfig{
		Pool:         newTestPool(t),
		Schema:       "test",
		DefaultModel: "model",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Typeahead(context.Background(), "two", TypeaheadOptions{
		LanguageMode: LanguageMode("invalid"),
		EntityTypes:  []string{"gallery"},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid TypeaheadOptions.LanguageMode") {
		t.Fatalf("expected invalid TypeaheadOptions.LanguageMode error, got: %v", err)
	}
}
