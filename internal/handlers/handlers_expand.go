package handlers

import (
	"context"
	"encoding/json"
	"github.com/antonevtu/go_shortener_adv/internal/cfg"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

type responseUserHistory []item
type item struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func handlerExpandURL(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfgApp.CtxTimeout)*time.Second)
		defer cancel()
		entity, err := repo.SelectByShortID(ctx, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if entity.Deleted {
			w.WriteHeader(http.StatusGone)
			return
		} else {
			w.Header().Set("Location", entity.LongURL)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}
}

func handlerUserHistory(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfgApp.CtxTimeout)*time.Second)
		defer cancel()
		selection, err := repo.SelectByUser(ctx, userID.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		setCookie(w, userID)
		w.Header().Set("Content-Type", "application/json")

		if len(selection) > 0 {
			history := make(responseUserHistory, len(selection))
			for i, v := range selection {
				history[i] = item{
					ShortURL:    cfgApp.BaseURL + "/" + v.ShortID,
					OriginalURL: v.LongURL,
				}
			}
			js, err := json.Marshal(history)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			w.WriteHeader(http.StatusOK)
			_, err = w.Write(js)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
