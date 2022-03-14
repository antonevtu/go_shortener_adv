package main

import (
	"github.com/antonevtu/go_shortener_adv/internal/app"
	_ "net/http/pprof"
)

func main() {
	app.Run()
	//http.ListenAndServe(":8081", nil) // запускаем сервер
}
