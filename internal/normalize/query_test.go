package normalize

import "testing"

func TestQueryForEmbedding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "  two   factor  ", want: "two factor"},
		{in: "two-factor", want: "two factor"},
		{in: "-factor", want: "factor"},
		{in: "--factor", want: "factor"},
		{in: "two not factor", want: "two not factor"},
		{in: "not two-factor", want: "not two factor"},
		{in: "R-18", want: "R 18"},
	}

	for _, tc := range cases {
		if got := QueryForEmbedding(tc.in); got != tc.want {
			t.Fatalf("QueryForEmbedding(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestQueryForFTS(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "two factor", want: "two factor"},
		{in: "two-factor", want: "two factor"},
		{in: "-factor", want: "factor"},
		{in: "not factor", want: "-factor"},
		{in: "two not factor", want: "two -factor"},
		{in: "not two-factor", want: "-two factor"},
		{in: "X NOT Y", want: "X -Y"},
	}

	for _, tc := range cases {
		if got := QueryForFTS(tc.in); got != tc.want {
			t.Fatalf("QueryForFTS(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
