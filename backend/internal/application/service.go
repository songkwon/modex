package application

import (
	"modex/backend/internal/auth"
	"modex/backend/internal/config"
	"modex/backend/internal/embedding"
	"modex/backend/internal/search"
	"modex/backend/internal/store"
)

// Repository is the persistence boundary used by the application layer.
// Implementations can be backed by pgx, an ORM, or a test double without
// leaking database details into HTTP controllers.
type Repository interface {
	Close()
}

// Service owns the business-facing dependencies used by controllers.
type Service struct {
	store      store.DataStore
	auth       *auth.Service
	search     search.Service
	repository Repository
}

func New(st store.DataStore, vectors search.VectorStore, repository Repository) *Service {
	return newService(st, vectors, repository, auth.NewService(auth.FromEnv()))
}

func NewConfigured(st store.DataStore, vectors search.VectorStore, repository Repository) (*Service, error) {
	authService, err := auth.NewConfiguredService(auth.FromEnv())
	if err != nil {
		return nil, err
	}
	return newService(st, vectors, repository, authService), nil
}

func newService(st store.DataStore, vectors search.VectorStore, repository Repository, authService *auth.Service) *Service {
	contentStore := search.ContentStore(st)
	scoring := config.LoadSearchScoring()
	provider := embedding.SettingsProvider{Load: func() embedding.Settings {
		ai := contentStore.Settings().AI
		return embedding.Settings{
			BaseURL: ai.EmbeddingBaseURL,
			Model:   ai.EmbeddingModel,
			APIKey:  ai.EmbeddingAPIKey,
			Dim:     ai.EmbeddingDim,
		}
	}}
	return &Service{
		store:      st,
		auth:       authService,
		repository: repository,
		search: search.Service{
			Store:          contentStore,
			Embedder:       provider,
			Vectors:        vectors,
			KeywordWeight:  positiveFloat(scoring.KeywordWeight, 0.6),
			SemanticWeight: positiveFloat(scoring.SemanticWeight, 0.4),
		},
	}
}

func (s *Service) Store() store.DataStore {
	return s.store
}

func (s *Service) Auth() *auth.Service {
	return s.auth
}

func (s *Service) Search() *search.Service {
	return &s.search
}

func (s *Service) Close() {
	_ = s.auth.Close()
	if s.repository != nil {
		s.repository.Close()
	}
}

func positiveFloat(v, fallback float64) float64 {
	if v <= 0 {
		return fallback
	}
	return v
}
