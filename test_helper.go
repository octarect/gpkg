package gpkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "gpkg-test-*")
	require.NoError(t, err)
	return tmpDir
}

func assertDirectoryContents(t *testing.T, dir string, expectedFiles []string) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		file := strings.TrimPrefix(path, dir)
		file = strings.TrimPrefix(file, "/")
		files = append(files, file)
		return nil
	})
	require.NoError(t, err)

	sort.Strings(files)
	sort.Strings(expectedFiles)
	assert.Equal(t, expectedFiles, files)
}

// Build an in memory archive with the specified files, writing the path as each
// file's contents when applicable.
func makeTarGz(t *testing.T, files []*tar.Header) *bytes.Buffer {
	var archive bytes.Buffer
	gzw := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gzw)

	for _, f := range files {
		if f.Typeflag == tar.TypeReg {
			contents := []byte(f.Name)
			f.Size = int64(len(contents))
			err := tw.WriteHeader(f)
			require.NoError(t, err)

			var written int
			written, err = tw.Write(contents)
			require.NoError(t, err)
			require.EqualValues(t, len(contents), written)
		} else {
			err := tw.WriteHeader(f)
			require.NoError(t, err)
		}
	}

	err := tw.Close()
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)

	return &archive
}

func checkDiff(t *testing.T, valueType, expected, got interface{}, ignoreFields ...string) {
	o := []cmp.Option{}
	for _, f := range ignoreFields {
		o = append(o, cmpopts.IgnoreFields(valueType, f))
	}
	o = append(o, cmp.AllowUnexported(valueType))
	if diff := cmp.Diff(expected, got, o...); diff != "" {
		t.Errorf("diff: -expected, +got:\n%s", diff)
	}
}

// NopSpec implements PackageSpec interface and its methods do nothing.
// You can use this struct in test code.
type NopSpec struct {
	*CommonSpec
}

var _ PackageSpec = &NopSpec{}

func NewNopSpec(id string) *NopSpec {
	return &NopSpec{
		CommonSpec: &CommonSpec{
			From: "nop",
			ID:   id,
			config: &Config{
				CachePath: "/tmp",
			},
		},
	}
}

func (s *NopSpec) PackagePath() string {
	return filepath.Join(s.config.GetPackagesPath(), s.ID)
}

func (s *NopSpec) Unique() string {
	return s.ID
}

type muxHandler func(http.ResponseWriter, *http.Request)

func newTestServer(path string, code int, payload string) *httptest.Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Write([]byte(payload))
	})
	return server
}
