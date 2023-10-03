package gpkg

import (
	"fmt"
	"io"
	"net/http"
)

type Source interface {
	GetDownloader() (Downloader, error)
}

type Downloader interface {
	io.ReadCloser
	GetAssetName() string
	GetContentLength() int64
}

type HTTPDownloader struct {
	r     io.ReadCloser
	name  string
	total int64
}

func NewHTTPDownloader(name, url string) (*HTTPDownloader, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("gpkg/%s", Version))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code was returned. expected=200, got=%d, url=%s", resp.StatusCode, url)
	}
	return &HTTPDownloader{
		r:     resp.Body,
		name:  name,
		total: resp.ContentLength,
	}, nil
}

func (dl *HTTPDownloader) Read(p []byte) (int, error) {
	return dl.r.Read(p)
}

func (dl *HTTPDownloader) Close() error {
	return dl.r.Close()
}

func (dl *HTTPDownloader) GetAssetName() string {
	return dl.name
}

func (dl *HTTPDownloader) GetContentLength() int64 {
	return dl.total
}
