package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/antonevtu/go_shortener_adv/internal/cfg"
	"github.com/antonevtu/go_shortener_adv/internal/db"
	"github.com/antonevtu/go_shortener_adv/internal/handlers"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type requestURL struct {
	URL string `json:"url"`
}
type responseURL struct {
	Result string `json:"result"`
}

var (
	ServerAddress   *string
	BaseURL         *string
	FileStoragePath *string
	DatabaseDSN     *string
	CtxTimeout      *int64
)

func init() {
	ServerAddress = flag.String("a", ":8080", "server address for shorten")
	BaseURL = flag.String("b", "http://localhost:8080", "base url for expand")
	FileStoragePath = flag.String("f", "./storage.txt", "path to storage file")
	DatabaseDSN = flag.String("d", "", "postgres url")
	CtxTimeout = flag.Int64("t", 500, "context timeout")
}

func BenchmarkOne(b *testing.B) {
	cfgApp := newConfig()
	container, database := createDBConnection()
	defer container.Terminate(context.Background())
	defer database.Close()
	migrations(database)
	r := handlers.NewRouter(&database, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()
	//time.Sleep(3 * time.Second)

	b.ResetTimer() // сбрасываем все счётчики
	shortPaths := make([]string, 0, 10000)

	b.Run("shorten", func(b *testing.B) {
		for i := 0; i < b.N; i++ {

			// подготовка запроса
			longURL := "https://yandex.ru/" + uuid.NewString()
			reqAPI, err := json.Marshal(requestURL{URL: longURL})
			if err != nil {
				panic(err)
			}
			client := &http.Client{}
			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/shorten", bytes.NewBuffer(reqAPI))
			if err != nil {
				panic(err)
			}

			b.StartTimer() // возобновляем таймер
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			b.StopTimer() // останавливаем таймер

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
			shortPaths = append(shortPaths, u.Path)
		}
	})

	b.Run("expand", func(b *testing.B) {
		for i := 0; i < b.N; i++ {

			// подготовка запроса
			n := rand.Intn(len(shortPaths))
			client := &http.Client{}
			req, err := http.NewRequest(http.MethodGet, ts.URL+shortPaths[n], bytes.NewBufferString(""))
			if err != nil {
				panic(err)
			}
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}

			b.StartTimer() // возобновляем таймер
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			_ = resp.Body.Close()
			b.StopTimer() // останавливаем таймер

			if resp.StatusCode != http.StatusTemporaryRedirect {
				panic(errors.New("invalid status temporary redirect"))
			}
		}
	})
}

func newConfig() cfg.Config {
	cfgApp := cfg.Config{
		ServerAddress:   *ServerAddress,
		BaseURL:         *BaseURL,
		FileStoragePath: *FileStoragePath,
		DatabaseDSN:     *DatabaseDSN,
		CtxTimeout:      *CtxTimeout,
	}
	return cfgApp
}

func createDBConnection() (testcontainers.Container, db.T) {
	var dbPool db.T
	ctx := context.Background()
	container, db1, err := createTestContainer(ctx, "pg")
	if err != nil {
		panic(err)
	}
	//defer db1.Close()
	//defer container.Terminate(ctx)
	dbPool.Pool = db1
	return container, dbPool
}

func createTestContainer(ctx context.Context, dbname string) (testcontainers.Container, *pgxpool.Pool, error) {
	var env = map[string]string{
		"POSTGRES_PASSWORD": "password",
		"POSTGRES_USER":     "postgres",
		"POSTGRES_DB":       dbname,
	}
	var port = "5432/tcp"
	dbURL := func(port nat.Port) string {
		return fmt.Sprintf("postgres://postgres:password@localhost:%s/%s?sslmode=disable", port.Port(), dbname)
	}

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:latest",
			ExposedPorts: []string{port},
			Cmd:          []string{"postgres", "-c", "fsync=off"},
			Env:          env,
			WaitingFor:   wait.ForSQL(nat.Port(port), "postgres", dbURL).Timeout(time.Second * 100),
		},
		Started: true,
	}
	container, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		return container, nil, fmt.Errorf("failed to start container: %s", err)
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(port))
	if err != nil {
		return container, nil, fmt.Errorf("failed to get container external port: %s", err)
	}

	log.Println("postgres container ready and running at port: ", mappedPort)

	urlS := fmt.Sprintf("postgres://postgres:password@localhost:%s/%s?sslmode=disable", mappedPort.Port(), dbname)

	db1, err := pgxpool.Connect(ctx, urlS)
	//db, err := sql.Open("postgres", url)
	if err != nil {
		return container, db1, fmt.Errorf("failed to establish database connection: %s", err)
	}

	return container, db1, nil
}

func migrations(dbPool db.T) {
	// создание таблицы
	sql1 := "create table if not exists urls (" +
		"id serial primary key, " +
		"deleted boolean not null," +
		"user_id varchar(512) not null, " +
		"short_id varchar(512) not null unique, " +
		"long_url varchar(1024) not null unique)"
	_, err := dbPool.Exec(context.Background(), sql1)
	if err != nil {
		panic(err)
	}
}
