package handlers

import (
	"github.com/antonevtu/go_shortener_adv/internal/cfg"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http/pprof"
)

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

		// профилировщик
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
		r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		//r.Handle("/pprof/block", pprof.Handler("block"))
		//r.Handle("/pprof/allocs", pprof.Handler("allocs"))
	})
	return r
}
