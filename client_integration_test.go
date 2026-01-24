package searchkit

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

func TestClientSearch_Integration_LexicalAndSemantic(t *testing.T) {
	dsn := os.Getenv("SEARCHKIT_TEST_URL")
	if dsn == "" {
		t.Skip("SEARCHKIT_TEST_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	// Minimal schema setup for FTS + semantic search.
	_, err = pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS s;
		SET search_path = s, public;
		CREATE EXTENSION IF NOT EXISTS pg_trgm;
		CREATE EXTENSION IF NOT EXISTS vector;

		CREATE OR REPLACE FUNCTION searchkit_regconfig_for_language(lang text)
		RETURNS regconfig
		LANGUAGE sql
		IMMUTABLE
		AS $$
			SELECT 'simple'::regconfig
		$$;

		CREATE TABLE IF NOT EXISTS search_documents (
			entity_type text NOT NULL,
			entity_id text NOT NULL,
			language text NOT NULL,
			raw_document text,
			tsv tsvector,
			created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (entity_type, entity_id, language)
		);

		CREATE TABLE IF NOT EXISTS embedding_vectors (
			entity_type text NOT NULL,
			entity_id text NOT NULL,
			model text NOT NULL,
			language text NOT NULL,
			embedding halfvec,
			created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (entity_type, entity_id, model, language)
		);

		TRUNCATE TABLE search_documents;
		TRUNCATE TABLE embedding_vectors;
	`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO s.search_documents(entity_type, entity_id, language, raw_document, tsv)
		VALUES ('gallery', '1', 'en', 'Two factor authentication', to_tsvector(searchkit_regconfig_for_language('en'), 'Two factor authentication'))
	`)
	if err != nil {
		t.Fatalf("insert search_documents: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO s.embedding_vectors(entity_type, entity_id, model, language, embedding)
		VALUES ('gallery', '1', 'm', 'en', $1::halfvec(3))
	`, pgvector.NewHalfVector([]float32{1, 0, 0}))
	if err != nil {
		t.Fatalf("insert embedding_vectors: %v", err)
	}

	emb := &recordingEmbedder{vec: []float32{1, 0, 0}}
	client, err := NewClient(ClientConfig{
		Pool:         pool,
		Schema:       "s",
		Embedder:     emb,
		DefaultModel: "m",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	lexHits, err := client.Search(ctx, "factor", SearchOptions{
		Mode:               SearchModeLexical,
		Language:           "en",
		LexicalEntityTypes: []string{"gallery"},
		Limit:              10,
	})
	if err != nil {
		t.Fatalf("lexical Search: %v", err)
	}
	if len(lexHits) == 0 || lexHits[0].EntityID != "1" {
		t.Fatalf("expected lexical hit entity_id=1, got %+v", lexHits)
	}

	semHits, err := client.Search(ctx, "two-factor", SearchOptions{
		Mode:                SearchModeSemantic,
		Language:            "en",
		SemanticEntityTypes: []string{"gallery"},
		Limit:               10,
	})
	if err != nil {
		t.Fatalf("semantic Search: %v", err)
	}
	if len(semHits) == 0 || semHits[0].EntityID != "1" {
		t.Fatalf("expected semantic hit entity_id=1, got %+v", semHits)
	}
}
