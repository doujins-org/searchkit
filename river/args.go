package river

// EmbeddingTaskBatchArgs runs a bounded batch against embeddingkit's task table.
// It's intended to be scheduled periodically by the host app (e.g. every minute),
// or enqueued on-demand when you need backfill.
type EmbeddingTaskBatchArgs struct {
	Limit int `json:"limit"`
}

func (EmbeddingTaskBatchArgs) Kind() string { return "embeddingkit_embedding_tasks_batch" }

