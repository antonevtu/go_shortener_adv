// Package handlers processes http requests for URL shortening, extension, and deletion
package handlers

import (
	"context"
	"github.com/antonevtu/go_shortener_adv/internal/cfg"
	"github.com/antonevtu/go_shortener_adv/internal/db"
	"github.com/antonevtu/go_shortener_adv/internal/pool"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http/pprof"
)

type Repositorier interface {

	//AddEntity adds new row Entity in DB. If long URL already exists, returns ErrUniqueViolation
	AddEntity(ctx context.Context, entity db.Entity) error

	//SelectByLongURL returns row Entity for known long URL
	SelectByLongURL(ctx context.Context, longURL string) (db.Entity, error)

	//SelectByShortID returns row Entity for known short ID
	SelectByShortID(ctx context.Context, shortURL string) (db.Entity, error)

	//SelectByUser returns all Entity rows for given userID
	SelectByUser(ctx context.Context, userID string) ([]db.Entity, error)

	//AddEntityBatch fast adds BatchInput in transaction mode
	AddEntityBatch(ctx context.Context, userID string, input db.BatchInput) error

	//Ping checks DB connection is alive
	Ping(ctx context.Context) error

	//SetDeletedBatch fast delete several Entities in transaction mode
	//Doesn't remove rows, only sets deleted flags = true
	SetDeletedBatch(ctx context.Context, userID string, shortIDs []string) error

	//SetDeleted delete one row Entity.
	//Doesn't remove row, only sets deleted flag = true
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
		r.Delete("/api/user/urls", handlerDelete(cfgApp))

		//r.Get("/pprof/profile", pprof.Profile)
		//r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		//	http.Redirect(w, r, r.RequestURI+"/pprof/", http.StatusMovedPermanently)
		//})
		//r.HandleFunc("/pprof", func(w http.ResponseWriter, r *http.Request) {
		//	http.Redirect(w, r, r.RequestURI+"/", http.StatusMovedPermanently)
		//})

		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)

		//r.HandleFunc("/pprof/*", pprof.Index)
		//r.HandleFunc("/pprof/cmdline", pprof.Cmdline)
		//r.HandleFunc("/pprof/profile", pprof.Profile)
		//r.HandleFunc("/pprof/symbol", pprof.Symbol)
		//r.HandleFunc("/pprof/trace", pprof.Trace)
		//r.HandleFunc("/vars", expVars)

		//r.Handle("/pprof/goroutine", pprof.Handler("goroutine"))
		//r.Handle("/pprof/threadcreate", pprof.Handler("threadcreate"))
		//r.Handle("/pprof/mutex", pprof.Handler("mutex"))
		r.Handle("/pprof/heap", pprof.Handler("heap"))
		//r.Handle("/pprof/block", pprof.Handler("block"))
		//r.Handle("/pprof/allocs", pprof.Handler("allocs"))
	})
	return r
}

//// Replicated from expvar.go as not public.
//func expVars(w http.ResponseWriter, r *http.Request) {
//	first := true
//	w.Header().Set("Content-Type", "application/json")
//	fmt.Fprintf(w, "{\n")
//	expvar.Do(func(kv expvar.KeyValue) {
//		if !first {
//			fmt.Fprintf(w, ",\n")
//		}
//		first = false
//		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
//	})
//	fmt.Fprintf(w, "\n}\n")
//}
