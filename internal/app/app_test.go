package app

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
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
	"os"
	"testing"
)

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

func TestGZipJSONAPI(t *testing.T) {
	cfgApp := cfg.Config{
		ServerAddress:   *ServerAddress,
		BaseURL:         *BaseURL,
		FileStoragePath: *FileStoragePath,
		DatabaseDSN:     *DatabaseDSN,
		CtxTimeout:      *CtxTimeout,
	}

	_ = os.Remove(*FileStoragePath)
	repo, err := repository.New(*FileStoragePath)
	assert.Equal(t, err, nil)

	r := handlers.NewRouter(repo, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create ID1
	longURL := "https://yandex.ru/maps/geo/sochi/53166566/?ll=39.580041%2C43.713351&z=9.98"
	buf := testEncodeJSONLongURL(longURL)
	resp, shortURLInJSON := testGZipRequest(t, ts.URL+"/api/shorten", "POST", buf)
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
}

func TestJSONAPI(t *testing.T) {
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
	resp, shortURLInJSON := testRequest(t, ts.URL+"/api/shorten", "POST", buf)
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse shortURL
	shortURL := testDecodeJSONShortURL(t, shortURLInJSON)
	_, err = url.Parse(shortURL)
	require.NoError(t, err)

	// Create ID2
	longURL = "https://habr.com/ru/all/"
	buf = testEncodeJSONLongURL(longURL)
	resp, shortURLInJSON = testRequest(t, ts.URL+"/api/shorten", "POST", buf)
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse shortURL
	shortURL = testDecodeJSONShortURL(t, shortURLInJSON)
	u, err := url.Parse(shortURL)
	require.NoError(t, err)

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

	// Check server returns correct baseURL
	baseURL := u.Scheme + "://" + u.Host
	assert.Equal(t, baseURL, *BaseURL)

	// Check storage file created
	file, err := os.Open(*FileStoragePath)
	assert.Equal(t, err, nil)
	defer file.Close()
}

func TestTextAPI(t *testing.T) {
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

	// Create ID
	longURL := "https://yandex.ru/maps/geo/sochi/53166566/?ll=39.580041%2C43.713351&z=9.98"
	resp, shortURL_ := testRequest(t, ts.URL, "POST", bytes.NewBufferString(longURL))
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse shortURL
	u, err := url.Parse(shortURL_)
	require.NoError(t, err)

	// Check redirection by existing ID
	resp, _ = testRequest(t, ts.URL+u.Path, "GET", nil)
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	longURLRecovered := resp.Header.Get("Location")
	assert.Equal(t, longURL, longURLRecovered)

	// Check StatusBadRequest for non existed ID
	shortURL1 := ts.URL + u.Path + "xxx"
	resp, _ = testRequest(t, shortURL1, "GET", nil)
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Check not allowed method error
	resp, _ = testRequest(t, ts.URL+u.Path, "PUT", nil)
	err = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Check storage file created
	file, err := os.Open(*FileStoragePath)
	assert.Equal(t, err, nil)
	defer file.Close()
}

func testGZipRequest(t *testing.T, url, method string, body io.Reader) (*http.Response, string) {
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
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	dec, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	defer dec.Close()

	respBody, err := ioutil.ReadAll(dec)
	require.NoError(t, err)

	return resp, string(respBody)
}

func testRequest(t *testing.T, url, method string, body io.Reader) (*http.Response, string) {
	client := &http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	defer resp.Body.Close()

	return resp, string(respBody)
}

func testEncodeJSONLongURL(url string) *bytes.Buffer {
	v := struct {
		URL string `json:"url"`
	}{URL: url}
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false) // без этой опции символ '&' будет заменён на "\u0026"
	_ = encoder.Encode(v)
	return buf
}

func testDecodeJSONShortURL(t *testing.T, js string) string {
	url_ := struct {
		Result string `json:"result"`
	}{}
	err := json.Unmarshal([]byte(js), &url_)
	require.NoError(t, err)
	return url_.Result
}
