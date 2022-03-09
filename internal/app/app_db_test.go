package app

type batchInput []batchInputItem
type batchInputItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}
type batchOutput []batchOutputItem
type batchOutputItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

/*
func TestDBBatch(t *testing.T) {
	cfgApp := cfg.Config{
		ServerAddress:   *ServerAddress,
		BaseURL:         *BaseURL,
		FileStoragePath: *FileStoragePath,
		DatabaseDSN:     *DatabaseDSN,
		CtxTimeout:      *CtxTimeout,
	}

	dbPool, err := db.New(context.Background(), *DatabaseDSN)
	assert.Equal(t, err, nil)
	r := handlers.NewRouter(&dbPool, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

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
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = respBody
}

func TestDBPing(t *testing.T) {
	cfgApp := cfg.Config{
		ServerAddress:   *ServerAddress,
		BaseURL:         *BaseURL,
		FileStoragePath: *FileStoragePath,
		DatabaseDSN:     *DatabaseDSN,
		CtxTimeout:      *CtxTimeout,
	}

	dbPool, err := db.New(context.Background(), *DatabaseDSN)
	assert.Equal(t, err, nil)
	r := handlers.NewRouter(&dbPool, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)
	defer resp.Body.Close()

}

func TestDBJSONAPIConflict(t *testing.T) {
	cfgApp := cfg.Config{
		ServerAddress:   *ServerAddress,
		BaseURL:         *BaseURL,
		FileStoragePath: *FileStoragePath,
		DatabaseDSN:     *DatabaseDSN,
		CtxTimeout:      *CtxTimeout,
	}

	dbPool, err := db.New(context.Background(), *DatabaseDSN)
	assert.Equal(t, err, nil)

	r := handlers.NewRouter(&dbPool, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create ID1
	longURL := "https://yandex.ru/" + uuid.NewString()
	buf := testEncodeJSONLongURL(longURL)
	resp, shortURLInJSON := testRequest(t, ts.URL+"/api/shorten", "POST", buf)
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse shortURL
	shortURL1 := testDecodeJSONShortURL(t, shortURLInJSON)
	_, err = url.Parse(shortURL1)
	require.NoError(t, err)

	// Create ID2
	//longURL = "https://habr.com/ru/all/"
	buf = testEncodeJSONLongURL(longURL)
	resp, shortURLInJSON = testRequest(t, ts.URL+"/api/shorten", "POST", buf)
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	// Parse shortURL
	shortURL2 := testDecodeJSONShortURL(t, shortURLInJSON)
	u, err := url.Parse(shortURL2)
	require.NoError(t, err)

	// check Short URLs are equal
	require.Equal(t, shortURL1, shortURL2)

	// Check redirection by existing ID
	resp, _ = testRequest(t, ts.URL+u.Path, "GET", nil)
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	longURLRecovered := resp.Header.Get("Location")
	assert.Equal(t, longURL, longURLRecovered)

	// Check StatusBadRequest for incorrect JSON key in request
	badJSON := `{"urlBad":"abc"}`
	resp, _ = testRequest(t, ts.URL+"/api/shorten", "POST", bytes.NewBufferString(badJSON))
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

}

func TestDBCookie(t *testing.T) {
	cfgApp := cfg.Config{
		ServerAddress:   *ServerAddress,
		BaseURL:         *BaseURL,
		FileStoragePath: *FileStoragePath,
		DatabaseDSN:     *DatabaseDSN,
		CtxTimeout:      *CtxTimeout,
	}

	dbPool, err := db.New(context.Background(), *DatabaseDSN)
	assert.Equal(t, err, nil)

	r := handlers.NewRouter(&dbPool, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create ID1
	longURL := "https://yandex.ru/" + uuid.NewString()
	buf := testEncodeJSONLongURL(longURL)
	resp, shortURLInJSON := testGZipRequest(t, ts.URL+"/api/shorten", "POST", buf)
	cookies := resp.Cookies()
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Check response header
	val := resp.Header.Get("Content-Encoding")
	assert.Equal(t, val, "gzip")

	// Parse shortURL
	shortURL := testDecodeJSONShortURL(t, shortURLInJSON)
	_, err = url.Parse(shortURL)
	require.NoError(t, err)

	// Create ID2
	longURL = "https://yandex.ru/" + uuid.NewString()
	buf = testEncodeJSONLongURL(longURL)
	resp, shortURLInJSON = testGZipRequestCookie(t, ts.URL+"/api/shorten", "POST", buf, cookies)
	//cookies = resp.Cookies()
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Check response header
	val = resp.Header.Get("Content-Encoding")
	assert.Equal(t, val, "gzip")

	// Parse shortURL
	shortURL = testDecodeJSONShortURL(t, shortURLInJSON)
	_, err = url.Parse(shortURL)
	require.NoError(t, err)

	// Test User history with true cookie
	resp, urls := testGZipRequestCookie(t, ts.URL+"/user/urls", "GET", buf, cookies)
	err = resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, len(urls) > 2, true)

	// Test User history with false cookie
	cookies[0].Value = "4a03f9cb72d311ec9a6930c9abdb0255e0706164f46eddd837acfabfdc37ffb65275b7ca3f1e7cb9e113f4c79c33e910"
	resp = testGZipRequestCookie204(t, ts.URL+"/user/urls", "GET", buf, cookies)
	err = resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

//*/
