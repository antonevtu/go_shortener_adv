package main

import (
	"github.com/antonevtu/go_shortener_adv/internal/app"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go app.Run()
	http.ListenAndServe(":8081", nil) // запускаем сервер
}
