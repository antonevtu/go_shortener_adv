package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
)

func Example() {
	// подготовка запроса
	longURL := "https://yandex.ru/" + uuid.NewString()
	reqAPI, err := json.Marshal(requestURL{URL: longURL})
	if err != nil {
		panic(err)
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/api/shorten", bytes.NewBuffer(reqAPI))
	if err != nil {
		panic(err)
	}

	// выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	// извлекаем сокращенный путь
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

	// запрашиваем исходный URL
	// подготовка запроса
	req, err = http.NewRequest(http.MethodGet, "http://localhost:8080"+u.Path, bytes.NewBufferString(""))
	if err != nil {
		panic(err)
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		panic(errors.New("invalid status temporary redirect"))
	}
}
