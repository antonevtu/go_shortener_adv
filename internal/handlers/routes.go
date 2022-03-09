package handlers

import (
	"context"
	"github.com/antonevtu/go_shortener_adv/internal/cfg"
	"github.com/antonevtu/go_shortener_adv/internal/db"
	"github.com/antonevtu/go_shortener_adv/internal/pool"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Repositorier interface {
	AddEntity(ctx context.Context, entity db.Entity) error
	SelectByLongURL(ctx context.Context, longURL string) (db.Entity, error)
	SelectByShortID(ctx context.Context, shortURL string) (db.Entity, error)
	SelectByUser(ctx context.Context, userID string) ([]db.Entity, error)
	AddEntityBatch(ctx context.Context, userID string, input db.BatchInput) error
	Ping(ctx context.Context) error
	SetDeletedBatch(ctx context.Context, userID string, shortIDs []string) error
	SetDeleted(ctx context.Context, item pool.ToDeleteItem) error
}

func NewRouter(repo Repositorier, cfgApp cfg.Config) chi.Router {
	// Определяем роутер chi
	r := chi.NewRouter()

	// зададим встроенные middleware, чтобы улучшить стабильность приложения
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// архивирование запроса/ответа gzip
	r.Use(gzipResponseHandle)
	r.Use(gzipRequestHandle)

	// создадим суброутер
	r.Route("/", func(r chi.Router) {
		r.Post("/", handlerShortenURL(repo, cfgApp))
		r.Post("/api/shorten", handlerShortenURLJSONAPI(repo, cfgApp))
		r.Get("/{id}", handlerExpandURL(repo, cfgApp))
		r.Get("/api/user/urls", handlerUserHistory(repo, cfgApp))
		r.Get("/ping", handlerPingDB(repo))
		r.Post("/api/shorten/batch", handlerShortenURLAPIBatch(repo, cfgApp))
		r.Delete("/api/user/urls", handlerDelete(repo, cfgApp))
	})
	return r
}
