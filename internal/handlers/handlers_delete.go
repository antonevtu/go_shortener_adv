package handlers

import (
	"encoding/json"
	"github.com/antonevtu/go_shortener_adv/internal/cfg"
	"github.com/antonevtu/go_shortener_adv/internal/pool"
	"io"
	"net/http"
)

type shortIDList []string

func handlerDelete(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDBytes, err := getUserID(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		userID := userIDBytes.String()
		_ = userID

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var shortIDs shortIDList
		err = json.Unmarshal(body, &shortIDs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for _, shortID := range shortIDs {
			cfgApp.DeleterChan <- pool.ToDeleteItem{UserID: userID, ShortID: shortID}
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
