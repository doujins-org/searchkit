package search

import (
	"context"
	"fmt"
	"strings"

	querynorm "github.com/doujins-org/searchkit/internal/normalize"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FTSHit struct {
	EntityType string
	EntityID   string
	Language   string
	Score      float32
}

type FTSOptions struct {
	Schema      string
	Language    string
	EntityTypes []string
	Limit       int

	// FilterSQL is an optional additional WHERE fragment appended to the query as:
	//   ... AND (<FilterSQL>)
	//
	// It is intended for host-owned constraints that must be enforced inside the
	// retrieval query.
	//
	// IMPORTANT: this is trusted SQL provided by the host app. Do not insert
	// user input into it unsafely.
	FilterSQL string
	// FilterArgs are named args referenced by FilterSQL using pgx '@name'
	// placeholders (e.g. "... language = @lang").
	FilterArgs map[string]any
}

// NormalizeFTSScore maps Postgres `ts_rank_cd` scores into a bounded [0..1] range.
//
// `ts_rank_cd` does not have a fixed upper bound and can vary by document length.
// This normalization is intentionally simple and monotonic:
//
//	normalized = raw / (raw + 1)
func NormalizeFTSScore(raw float32) float32 {
	if raw <= 0 {
		return 0
	}
	return raw / (raw + 1)
}

// FTSSearchNormalized runs FTSSearch and normalizes the returned score into [0..1].
func FTSSearchNormalized(ctx context.Context, pool *pgxpool.Pool, query string, opts FTSOptions) ([]FTSHit, error) {
	hits, err := FTSSearch(ctx, pool, query, opts)
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].Score = NormalizeFTSScore(hits[i].Score)
	}
	return hits, nil
}

// FTSSearch runs a Postgres full-text search (BM25-family) query against
// `<schema>.search_documents.tsv`.
//
// Notes:
//   - This is language-aware via `searchkit_regconfig_for_language(language)`.
//   - The stored `tsv` is derived from `raw_document`, while trigram/typeahead
//     uses the heavy-normalized `document`.
func FTSSearch(ctx context.Context, pool *pgxpool.Pool, query string, opts FTSOptions) ([]FTSHit, error) {
	if strings.TrimSpace(opts.Schema) == "" {
		return nil, fmt.Errorf("schema is required")
	}
	if strings.TrimSpace(opts.Language) == "" {
		return nil, fmt.Errorf("language is required")
	}
	if opts.Limit <= 0 {
		return []FTSHit{}, nil
	}
	if pool == nil {
		return nil, fmt.Errorf("pool is required")
	}

	q := querynorm.QueryForFTS(query)
	if q == "" {
		return []FTSHit{}, nil
	}

	quotedSchema, err := quoteIdent(opts.Schema)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}
	table := quotedSchema + ".search_documents"

	where := "WHERE sd.language = @language AND sd.tsv IS NOT NULL"
	args := pgx.NamedArgs{
		"language": opts.Language,
		"q":        q,
		"limit":    opts.Limit,
	}
	if len(opts.EntityTypes) > 0 {
		where += " AND sd.entity_type = ANY(@entity_types::text[])"
		args["entity_types"] = opts.EntityTypes
	}
	if strings.TrimSpace(opts.FilterSQL) != "" {
		where += " AND (" + opts.FilterSQL + ")"
		if err := mergeNamedArgs(args, opts.FilterArgs); err != nil {
			return nil, err
		}
	}

	// Prefer websearch_to_tsquery (supports multi-word and quotes).
	// If the query is not parseable, fall back to plainto_tsquery.
	run := func(fn string) ([]FTSHit, error) {
		sql := fmt.Sprintf(`
			WITH q AS (
				SELECT %s(%s.searchkit_regconfig_for_language(@language), @q) AS tsq
			)
			SELECT
				sd.entity_type,
				sd.entity_id,
				sd.language,
				ts_rank_cd(sd.tsv, q.tsq)::float4 AS score
			FROM q, %s sd
			%s
			  AND q.tsq IS NOT NULL
			  AND sd.tsv @@ q.tsq
			ORDER BY score DESC, sd.entity_type ASC, sd.entity_id ASC
			LIMIT @limit
		`, fn, quotedSchema, table, where)

		rows, err := pool.Query(ctx, sql, args)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []FTSHit
		for rows.Next() {
			var h FTSHit
			if err := rows.Scan(&h.EntityType, &h.EntityID, &h.Language, &h.Score); err != nil {
				return nil, err
			}
			out = append(out, h)
		}
		return out, rows.Err()
	}

	out, err := run("websearch_to_tsquery")
	if err == nil {
		return out, nil
	}
	return run("plainto_tsquery")
}
