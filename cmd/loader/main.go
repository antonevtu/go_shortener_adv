//main used for test loading shortener module.
//Loading using numberGoroutines clients and each client makes numberRequests requests
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type shortPathsT struct {
	paths []string
	s     sync.Mutex
}

type requestURL struct {
	URL string `json:"url"`
}
type responseURL struct {
	Result string `json:"result"`
}

var shortPaths shortPathsT

const numberGoroutines = 300
const numberRequests = 100

func main() {
	shortPaths.paths = make([]string, 0, 1000)
	var n sync.WaitGroup
	tic := time.Now()

	for g := 0; g < numberGoroutines; g++ {
		n.Add(1)
		go shorten(&n)
	}

	for g := 0; g < numberGoroutines; g++ {
		n.Add(1)
		go expand(&n)
	}

	n.Wait()
	log.Println("Done in", time.Since(tic))
}

func shorten(n *sync.WaitGroup) {
	httpClient := &http.Client{}
	for i := 0; i < numberRequests; i++ {

		// подготовка запроса
		longURL := "https://yandex.ru/" + uuid.NewString()
		reqAPI, err := json.Marshal(requestURL{URL: longURL})
		if err != nil {
			panic(err)
		}
		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/api/shorten", bytes.NewBuffer(reqAPI))
		if err != nil {
			panic(err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			panic(err)
		}

		// сохраняем сокращенный путь
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		_ = resp.Body.Close()
		res := responseURL{}
		err = json.Unmarshal(body, &res)
		if err != nil {
			panic(err)
		}
		u, err := url.Parse(res.Result)
		if err != nil {
			panic(err)
		}
		shortPaths.s.Lock()
		shortPaths.paths = append(shortPaths.paths, u.Path)
		shortPaths.s.Unlock()

		log.Println("Выполнен запрос на сокращение")
	}

	n.Done()
}

func expand(n *sync.WaitGroup) {
	counter := 0
	httpClient := &http.Client{}
	for counter < numberRequests {

		// выборка короткого пути
		shortPaths.s.Lock()
		var shortPath string
		if len(shortPaths.paths) > 10 {
			n1 := rand.Intn(len(shortPaths.paths))
			shortPath = shortPaths.paths[n1]
			shortPaths.s.Unlock()
		} else {
			shortPaths.s.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// подготовка запроса
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+shortPath, bytes.NewBufferString(""))
		if err != nil {
			panic(err)
		}
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}

		// запрос
		resp, err := httpClient.Do(req)
		if err != nil {
			panic(err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusTemporaryRedirect {
			panic(errors.New("invalid status temporary redirect"))
		}

		counter++
		log.Println("Выполнен запрос на расширение")
	}

	n.Done()
}
