package gpkg

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type muxHandler func(http.ResponseWriter, *http.Request)

func newTestServer(code int, payload string) *httptest.Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Write([]byte(payload))
	})
	return server
}

func defaultTestHTTPDownloader(t *testing.T) (dl *HTTPDownloader, name string, msg string) {
	name = "foo"
	msg = "bar"
	srv := newTestServer(200, msg)
	dl, err := NewHTTPDownloader(name, srv.URL)
	require.NoError(t, err)
	return
}

func TestNewHTTPDownloader(t *testing.T) {
	// Success
	t.Run("success", func(t *testing.T) {
		name := "out"
		msg := "test"
		srv200 := newTestServer(200, msg)
		dl, err := NewHTTPDownloader("out", srv200.URL)
		require.NoError(t, err)
		assert.Equal(t, dl.name, name)
		assert.Equal(t, dl.total, int64(len(msg)))
	})

	// Failure
	t.Run("failure", func(t *testing.T) {
		srv404 := newTestServer(404, "")
		_, err := NewHTTPDownloader("out", srv404.URL)
		require.Error(t, err)
	})
}

func TestHTTPDownloader_Read(t *testing.T) {
	dl, _, msg := defaultTestHTTPDownloader(t)
	b, err := ioutil.ReadAll(dl)
	require.NoError(t, err)
	assert.Equal(t, string(b), msg)
}

func TestHTTPDownloader_Close(t *testing.T) {
	dl, _, _ := defaultTestHTTPDownloader(t)
	require.NoError(t, dl.Close())
}

func TestHTTPDownloader_GetAssetName(t *testing.T) {
	dl, name, _ := defaultTestHTTPDownloader(t)
	assert.Equal(t, dl.GetAssetName(), name)
}

func TestHTTPDownloader_GetContentLength(t *testing.T) {
	dl, _, msg := defaultTestHTTPDownloader(t)
	assert.Equal(t, dl.GetContentLength(), int64(len(msg)))
}
