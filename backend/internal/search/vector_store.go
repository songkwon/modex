package search

import "context"

// VectorStore persists document embeddings outside the process. Implementations
// should return independent slices that callers may safely discard or modify.
type VectorStore interface {
	Existing(ctx context.Context, docIDs []string) (map[string]bool, error)
	Similarities(ctx context.Context, query []float32, docIDs []string, limit int) (map[string]float64, error)
	UpsertChunk(ctx context.Context, docID, chunkID, content string, vector []float32) error
	Clear(ctx context.Context) error
	DeletePrefix(ctx context.Context, docIDPrefix string) error
	Count(ctx context.Context) (int, error)
}
