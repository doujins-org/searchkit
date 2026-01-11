package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool   *pgxpool.Pool
	schema string
}

func NewRepo(pool *pgxpool.Pool, schema string) *Repo {
	if schema == "" {
		schema = "embeddingkit"
	}
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

// FetchReady returns up to limit tasks ready to run now, and bumps next_run_at
// forward by lockAhead to reduce duplicate work across workers.
func (r *Repo) FetchReady(ctx context.Context, limit int, lockAhead time.Duration) ([]Task, error) {
	if limit <= 0 {
		return nil, nil
	}
	if lockAhead <= 0 {
		lockAhead = 30 * time.Second
	}

	now := time.Now().UTC()
	next := now.Add(lockAhead)

	q := fmt.Sprintf(`
		WITH picked AS (
			SELECT id
			FROM %s.embedding_tasks
			WHERE next_run_at <= $1
			ORDER BY next_run_at ASC, id ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE %s.embedding_tasks t
		SET next_run_at = $3, updated_at = $1
		WHERE t.id IN (SELECT id FROM picked)
		RETURNING
			t.id, t.entity_type, t.entity_id, t.model, t.reason, t.attempts, t.next_run_at, t.created_at, t.updated_at
	`, r.schema, r.schema)

	rows, err := r.pool.Query(ctx, q, now, limit, next)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(
			&t.ID,
			&t.EntityType,
			&t.EntityID,
			&t.Model,
			&t.Reason,
			&t.Attempts,
			&t.NextRunAt,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) Complete(ctx context.Context, id int64) error {
	if id <= 0 {
		return nil
	}
	q := fmt.Sprintf("DELETE FROM %s.embedding_tasks WHERE id = $1", r.schema)
	_, err := r.pool.Exec(ctx, q, id)
	return err
}

func (r *Repo) Fail(ctx context.Context, id int64, backoff time.Duration) error {
	if id <= 0 {
		return nil
	}
	if backoff <= 0 {
		backoff = 30 * time.Second
	}
	secs := int64(backoff / time.Second)
	if secs < 1 {
		secs = 1
	}
	q := fmt.Sprintf(`
		UPDATE %s.embedding_tasks
		SET attempts = attempts + 1,
		    next_run_at = now() + make_interval(secs => $1),
		    updated_at = now()
		WHERE id = $2
	`, r.schema)
	_, err := r.pool.Exec(ctx, q, secs, id)
	return err
}
