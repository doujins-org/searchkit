-- embeddingkit: task table only (embedding storage is app-owned)

CREATE SCHEMA IF NOT EXISTS embeddingkit;

CREATE TABLE IF NOT EXISTS embeddingkit.embedding_tasks (
    id bigserial PRIMARY KEY,
    entity_type text NOT NULL,
    entity_id bigint NOT NULL,
    model text NOT NULL,
    reason text NOT NULL DEFAULT 'unknown',
    attempts integer NOT NULL DEFAULT 0,
    next_run_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_embedding_tasks_entity_model
    ON embeddingkit.embedding_tasks(entity_type, entity_id, model);

CREATE INDEX IF NOT EXISTS idx_embedding_tasks_ready
    ON embeddingkit.embedding_tasks(next_run_at, id);
