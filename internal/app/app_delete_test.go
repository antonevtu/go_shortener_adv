package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/antonevtu/go_shortener_adv/internal/cfg"
	"github.com/antonevtu/go_shortener_adv/internal/db"
	"github.com/antonevtu/go_shortener_adv/internal/handlers"
	"github.com/antonevtu/go_shortener_adv/internal/pool"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type shortIDList []string
type ToDeleteItem struct {
	UserID  string
	ShortID string
}

// проверка
func TestDBDeleteBatch(t *testing.T) {
	cfgApp := cfg.Config{
		ServerAddress:   *ServerAddress,
		BaseURL:         *BaseURL,
		FileStoragePath: *FileStoragePath,
		DatabaseDSN:     *DatabaseDSN,
		CtxTimeout:      *CtxTimeout,
	}

	// локальная БД
	//dbPool, err := db.New(context.Background(), *DatabaseDSN)
	//assert.Equal(t, err, nil)

	// БД в контейнере
	var dbPool db.T
	ctx := context.Background()
	container, db1, err := createTestContainer(ctx, "pg")
	require.NoError(t, err)
	defer db1.Close()
	defer container.Terminate(ctx)
	dbPool.Pool = db1

	// пул горутин на удаление записей
	deleterPool := pool.New(context.Background(), &dbPool)
	//defer deleterPool.Close()
	cfgApp.DeleterChan = deleterPool.Input
	r := handlers.NewRouter(&dbPool, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// создание таблицы
	sql1 := "create table if not exists urls (" +
		"id serial primary key, " +
		"deleted boolean not null," +
		"user_id varchar(512) not null, " +
		"short_id varchar(512) not null unique, " +
		"long_url varchar(1024) not null unique)"
	_, err = dbPool.Exec(ctx, sql1)
	require.NoError(t, err)

	// запись в БД
	batch := make(batchInput, 3)
	batch[0] = batchInputItem{CorrelationID: "0", OriginalURL: "https://yandex.ru/" + uuid.NewString()}
	batch[1] = batchInputItem{CorrelationID: "1", OriginalURL: "https://yandex.ru/" + uuid.NewString()}
	batch[2] = batchInputItem{CorrelationID: "2", OriginalURL: "https://yandex.ru/" + uuid.NewString()}
	jsonBatch, err := json.Marshal(batch)
	require.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/shorten/batch", bytes.NewBuffer(jsonBatch))
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusCreated)
	cookies := resp.Cookies()
	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// извлечение shortIDs
	var batchOut batchOutput
	err = json.Unmarshal(respBody, &batchOut)
	require.NoError(t, err)
	shortIDs := make(shortIDList, 0, len(batch))
	for _, shortURL := range batchOut {
		u, err := url.Parse(shortURL.ShortURL)
		require.NoError(t, err)
		shortIDs = append(shortIDs, u.Path[1:])
	}

	// Запрос на удаление
	resp1, _ := testGZipRequestCookie(t, ts.URL+"/api/user/urls", http.MethodDelete, testEncodeJSONDeleteList(shortIDs), cookies)
	err = resp1.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp1.StatusCode)
	time.Sleep(time.Second)
}

func testEncodeJSONDeleteList(s shortIDList) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false) // без этой опции символ '&' будет заменён на "\u0026"
	_ = encoder.Encode(s)
	return buf
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
