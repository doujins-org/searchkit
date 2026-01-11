package runtime

import (
	"context"
	"fmt"

	"github.com/doujins-org/embeddingkit/embedder"
	"github.com/doujins-org/embeddingkit/tasks"
)

// DocumentBuilder builds a single canonical text document for an entity.
// Embeddings are language-agnostic in storage and search.
type DocumentBuilder func(ctx context.Context, entityType string, entityID int64) (string, error)

// Storage is implemented by the host application and maps embeddingkit's generic
// concepts to the app's concrete Postgres tables/indexes (typically halfvec(K)).
//
// This exists because halfvec requires fixed dimensions, and apps may store
// multiple models with different dims (e.g. 2560 vs 4096) in separate tables.
type Storage interface {
	UpsertTextEmbedding(ctx context.Context, entityType string, entityID int64, model string, dim int, embedding []float32) error
}

type Config struct {
	EnabledModels []ModelSpec
}

type ModelSpec struct {
	Name string
	Dim  int
	Kind string // "text" | "vl" (future)
}

type Runtime struct {
	textEmbedder embedder.Embedder
	taskRepo     *tasks.Repo
	storage      Storage
	builder      DocumentBuilder
	cfg          Config
}

func New(text embedder.Embedder, taskRepo *tasks.Repo, storage Storage, builder DocumentBuilder, cfg Config) (*Runtime, error) {
	if text == nil {
		return nil, fmt.Errorf("text embedder is required")
	}
	if taskRepo == nil {
		return nil, fmt.Errorf("task repo is required")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if builder == nil {
		return nil, fmt.Errorf("document builder is required")
	}
	return &Runtime{
		textEmbedder: text,
		taskRepo:     taskRepo,
		storage:      storage,
		builder:      builder,
		cfg:          cfg,
	}, nil
}

// EnqueueTextEmbedding enqueues a text embedding task for an entity+model.
func (r *Runtime) EnqueueTextEmbedding(ctx context.Context, entityType string, entityID int64, model string, reason string) error {
	return r.taskRepo.Enqueue(ctx, entityType, entityID, model, reason)
}

// GenerateAndStoreTextEmbedding computes and upserts the embedding for an entity.
// Intended to be called from a background worker.
func (r *Runtime) GenerateAndStoreTextEmbedding(ctx context.Context, entityType string, entityID int64, model string) error {
	doc, err := r.builder(ctx, entityType, entityID)
	if err != nil {
		return err
	}
	vec, err := r.textEmbedder.EmbedText(ctx, doc)
	if err != nil {
		return err
	}
	dim := len(vec)
	return r.storage.UpsertTextEmbedding(ctx, entityType, entityID, model, dim, vec)
}

