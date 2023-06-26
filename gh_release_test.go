package gpkg

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type DummyRelease struct {
	assets map[string]io.Reader
	server *httptest.Server
}

func NewDummyRelease(tag string, assets map[string]io.Reader) *DummyRelease {
	mux := http.NewServeMux()
	for fileName, reader := range assets {
		mux.HandleFunc("/"+fileName, func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(reader)
			w.WriteHeader(200)
			w.Write(body)
		})
	}

	return &DummyRelease{
		server: httptest.NewServer(mux),
		assets: assets,
	}
}

func (r *DummyRelease) GetRelease(ctx context.Context, ref string) (*Release, error) {
	rel := &Release{}
	rel.ref = ref
	rel.assets = make(map[string]string, len(r.assets))
	for k, _ := range r.assets {
		rel.assets[k] = r.server.URL + "/" + k
	}
	return rel, nil
}

func TestGHReleasePackage_Download(t *testing.T) {
	tests := []struct {
		name          string
		assets        map[string]io.Reader
		expectedError bool
		expectedFiles []string
	}{
		{
			"supports targz",
			map[string]io.Reader{
				"foo-0.0.1-linux_amd64.tar.gz": makeTarGz(t, []*tar.Header{
					{Name: "foo", Typeflag: tar.TypeReg},
				}),
			},
			false,
			[]string{"", "foo"},
		},
		{
			"supports direct binary",
			map[string]io.Reader{
				"foo-0.0.1-linux_amd64": bytes.NewBuffer([]byte("")),
			},
			false,
			[]string{"", "foo-0.0.1-linux_amd64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			defer os.RemoveAll(tmpDir)

			p := &GHReleasePackage{
				Spec: &PackageSpec{
					Name: "foo/bar",
				},
				owner:    "foo",
				repo:     "bar",
				releases: NewDummyRelease("v0.0.1", tt.assets),
			}

			err := p.Download(tmpDir)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assertDirectoryContents(t, tmpDir, tt.expectedFiles)
			}
		})
	}
}
