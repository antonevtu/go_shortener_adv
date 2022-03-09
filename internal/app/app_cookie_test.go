package app

import (
	"bytes"
	"compress/gzip"
	"github.com/antonevtu/go_shortener_adv/internal/cfg"
	"github.com/antonevtu/go_shortener_adv/internal/handlers"
	"github.com/antonevtu/go_shortener_adv/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCookie(t *testing.T) {
	//_ = os.Remove(cfgApp.FileStoragePath)
	cfgApp := cfg.Config{
		ServerAddress:   *ServerAddress,
		BaseURL:         *BaseURL,
		FileStoragePath: *FileStoragePath,
		DatabaseDSN:     *DatabaseDSN,
		CtxTimeout:      *CtxTimeout,
	}

	repo, err := repository.New(*FileStoragePath)
	assert.Equal(t, err, nil)

	r := handlers.NewRouter(repo, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create ID1
	longURL := "https://yandex.ru/maps/geo/sochi/53166566/?ll=39.580041%2C43.713351&z=9.98"
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
	longURL = "https://habr.com/ru/all/"
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

func testGZipRequestCookie(t *testing.T, url, method string, body io.Reader, cookie []*http.Cookie) (*http.Response, string) {
	client := &http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	buf := bytes.Buffer{}
	gz := gzip.NewWriter(&buf)
	defer gz.Close()
	body_, err := io.ReadAll(body)
	require.NoError(t, err)
	_, err = gz.Write(body_)
	require.NoError(t, err)
	err = gz.Close()
	require.NoError(t, err)

	req, err := http.NewRequest(method, url, &buf)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	if cookie != nil {
		req.AddCookie(cookie[0])
		require.NoError(t, err)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)

	dec, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	defer dec.Close()

	respBody, err := ioutil.ReadAll(dec)
	require.NoError(t, err)
	return resp, string(respBody)
}

func testGZipRequestCookie204(t *testing.T, url, method string, body io.Reader, cookie []*http.Cookie) *http.Response {
	client := &http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	buf := bytes.Buffer{}
	gz := gzip.NewWriter(&buf)
	defer gz.Close()
	body_, err := io.ReadAll(body)
	require.NoError(t, err)
	_, err = gz.Write(body_)
	require.NoError(t, err)
	err = gz.Close()
	require.NoError(t, err)

	req, err := http.NewRequest(method, url, &buf)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req.AddCookie(cookie[0])
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}
