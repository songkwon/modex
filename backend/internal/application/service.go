package application

import (
	"context"
	"os"
	"strconv"

	"modex/backend/internal/auth"
	"modex/backend/internal/embedding"
	"modex/backend/internal/search"
	"modex/backend/internal/store"
)

// Repository is the persistence boundary used by the application layer.
// Implementations can be backed by pgx, an ORM, or a test double without
// leaking database details into HTTP controllers.
type Repository interface {
	Save(ctx context.Context, st *store.Store) error
	Close()
}

// Service owns the business-facing dependencies used by controllers.
type Service struct {
	store      *store.Store
	auth       *auth.Service
	search     search.Service
	repository Repository
}

func New(st *store.Store, vectors search.VectorStore, repository Repository) *Service {
	provider := embedding.SettingsProvider{Load: func() embedding.Settings {
		ai := st.Settings().AI
		return embedding.Settings{
			BaseURL: ai.EmbeddingBaseURL,
			Model:   ai.EmbeddingModel,
			APIKey:  ai.EmbeddingAPIKey,
			Dim:     ai.EmbeddingDim,
		}
	}}
	return &Service{
		store:      st,
		auth:       auth.NewService(auth.FromEnv()),
		repository: repository,
		search: search.Service{
			Store:          st,
			Embedder:       provider,
			Vectors:        vectors,
			KeywordWeight:  envFloat("HYBRID_KEYWORD_WEIGHT", 0.6),
			SemanticWeight: envFloat("HYBRID_SEMANTIC_WEIGHT", 0.4),
		},
	}
}

func (s *Service) Store() *store.Store {
	return s.store
}

func (s *Service) Auth() *auth.Service {
	return s.auth
}

func (s *Service) Search() *search.Service {
	return &s.search
}

func (s *Service) HasRepository() bool {
	return s.repository != nil
}

func (s *Service) Save(ctx context.Context) error {
	if s.repository == nil {
		return nil
	}
	return s.repository.Save(ctx, s.store)
}

func (s *Service) Close() {
	if s.repository != nil {
		s.repository.Close()
	}
}

func envFloat(key string, fallback float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil || v == 0 {
		return fallback
	}
	return v
}
