//main realises service for shorten long url and returns.
//Long and short URLs are stored in Postgres or in file-memory storage.
//When short url requested, it returns redirection to original long url.
package main

import (
	"github.com/antonevtu/go_shortener_adv/internal/app"
)

func main() {
	app.Run()
}
