package vl

import "context"

// AssetKind indicates what kind of asset is being embedded.
type AssetKind string

const (
	AssetKindImage AssetKind = "image"
	AssetKindFrame AssetKind = "frame" // video frame
)

// AssetRef is an opaque handle returned by the host app's AssetLister.
// It typically points to an object in S3/MinIO (bucket/key) or a DB row.
type AssetRef struct {
	Kind     AssetKind
	Key      string
	FrameIdx *int // optional: for video-derived frames
}

// AssetLister returns the assets that should be embedded for an entity (gallery/video).
type AssetLister func(ctx context.Context, entityType string, entityID int64) ([]AssetRef, error)

type AssetContent struct {
	// URL is preferred when using hosted providers that can fetch the asset directly
	// (e.g. presigned S3/MinIO URL).
	URL string

	// Bytes is the fallback when the provider can't fetch URLs and needs upload.
	ContentType string
	Bytes       []byte
}

// AssetFetcher resolves an AssetRef into either a URL (preferred) or bytes.
// (Implementations may stream in practice; keep it simple for now.)
type AssetFetcher func(ctx context.Context, ref AssetRef) (AssetContent, error)

// Embedder generates vision-language embeddings for text+assets.
//
// NOTE: Provider wiring for Qwen3-VL-Embedding is intentionally out of scope
// here for now; embeddingkit defines the interface so apps can implement it.
type Embedder interface {
	Model() string
	Dimensions() int
	EmbedTextAndImages(ctx context.Context, text string, images []Image) ([]float32, error)
}

// URLEmbedder is an optional interface for providers that accept image URLs
// directly (preferred, to avoid proxying bytes through the app).
type URLEmbedder interface {
	EmbedTextAndImageURLs(ctx context.Context, text string, urls []string) ([]float32, error)
}

type Image struct {
	ContentType string
	Bytes       []byte
}
