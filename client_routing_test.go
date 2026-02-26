package searchkit

import "testing"

func TestLexicalRouting_Search(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		language string
		query    string
		want     lexicalRoute
	}{
		{
			name:     "non-CJK uses FTS",
			language: "en",
			query:    "tokyo school",
			want:     lexicalRoute{useFTS: true},
		},
		{
			name:     "CJK native script uses PGroonga",
			language: "ja",
			query:    "鬼滅",
			want:     lexicalRoute{usePGroonga: true},
		},
		{
			name:     "CJK ASCII-only uses trigram",
			language: "ja",
			query:    "tokyo",
			want:     lexicalRoute{useTrigram: true},
		},
		{
			name:     "CJK mixed script uses both",
			language: "ja",
			query:    "鬼滅 tokyo",
			want: lexicalRoute{
				useTrigram:  true,
				usePGroonga: true,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := lexicalRouting(tc.language, tc.query, false)
			if got != tc.want {
				t.Fatalf("lexicalRouting(%q, %q, false) = %+v; want %+v", tc.language, tc.query, got, tc.want)
			}
		})
	}
}

func TestLexicalRouting_Typeahead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		language string
		query    string
		want     lexicalRoute
	}{
		{
			name:     "non-CJK uses trigram",
			language: "en",
			query:    "tok",
			want:     lexicalRoute{useTrigram: true},
		},
		{
			name:     "CJK native script uses PGroonga",
			language: "zh",
			query:    "東京",
			want:     lexicalRoute{usePGroonga: true},
		},
		{
			name:     "CJK mixed uses both",
			language: "ko",
			query:    "東京 tok",
			want: lexicalRoute{
				useTrigram:  true,
				usePGroonga: true,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := lexicalRouting(tc.language, tc.query, true)
			if got != tc.want {
				t.Fatalf("lexicalRouting(%q, %q, true) = %+v; want %+v", tc.language, tc.query, got, tc.want)
			}
		})
	}
}
