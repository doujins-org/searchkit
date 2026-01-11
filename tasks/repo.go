package tasks

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool   *pgxpool.Pool
	schema string
}

func NewRepo(pool *pgxpool.Pool, schema string) *Repo {
	return &Repo{pool: pool, schema: schema}
}

func (r *Repo) Enqueue(ctx context.Context, entityType string, entityID int64, model string, reason string) error {
	if entityType == "" || model == "" {
		return fmt.Errorf("entityType and model are required")
	}
	q := fmt.Sprintf(`
		INSERT INTO %s.embedding_tasks (entity_type, entity_id, model, reason)
		VALUES ($1, $2, $3, COALESCE($4, 'unknown'))
		ON CONFLICT (entity_type, entity_id, model) DO UPDATE SET
			reason = EXCLUDED.reason,
			next_run_at = LEAST(%s.embedding_tasks.next_run_at, now()),
			updated_at = now()
	`, r.schema, r.schema)
	_, err := r.pool.Exec(ctx, q, entityType, entityID, model, reason)
	return err
}

