package searchkit

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type recordingEmbedder struct {
	called bool
	model  string
	text   string
	vec    []float32
	err    error
}

func (r *recordingEmbedder) EmbedQueryText(_ context.Context, model string, text string) ([]float32, error) {
	r.called = true
	r.model = model
	r.text = text
	if r.err != nil {
		return nil, r.err
	}
	return r.vec, nil
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Connection is established lazily; tests that don't hit the DB won't connect.
	// Port 1 should refuse quickly if a query is attempted.
	pool, err := pgxpool.New(context.Background(), "postgres://user:pass@127.0.0.1:1/db?sslmode=disable")
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestClientSearch_SemanticRequiresEmbedder(t *testing.T) {
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
		Mode:                SearchModeSemantic,
		SemanticEntityTypes: []string{"gallery"},
	})
	if err == nil || !strings.Contains(err.Error(), "Embedder is required") {
		t.Fatalf("expected embedder-required error, got: %v", err)
	}
}

func TestClientSearch_SemanticEmbedsNormalizedText(t *testing.T) {
	t.Parallel()

	emb := &recordingEmbedder{vec: []float32{1, 0, 0}}
	client, err := NewClient(ClientConfig{
		Pool:         newTestPool(t),
		Schema:       "test",
		Embedder:     emb,
		DefaultModel: "model",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, _ = client.Search(context.Background(), "two-factor", SearchOptions{
		Mode:                SearchModeSemantic,
		SemanticEntityTypes: []string{"gallery"},
	})
	if !emb.called {
		t.Fatalf("expected embedder to be called")
	}
	if emb.text != "two factor" {
		t.Fatalf("expected normalized query text %q, got %q", "two factor", emb.text)
	}
}

func TestClientSearch_LexicalDoesNotCallEmbedder(t *testing.T) {
	t.Parallel()

	emb := &recordingEmbedder{vec: []float32{1, 0, 0}}
	client, err := NewClient(ClientConfig{
		Pool:         newTestPool(t),
		Schema:       "test",
		Embedder:     emb,
		DefaultModel: "model",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, _ = client.Search(context.Background(), "two-factor", SearchOptions{
		Mode:               SearchModeLexical,
		LexicalEntityTypes: []string{"gallery"},
	})
	if emb.called {
		t.Fatalf("expected embedder not to be called in lexical mode")
	}
}
