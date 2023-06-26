package gpkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

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
