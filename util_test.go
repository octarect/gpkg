package gpkg

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractTarGz(t *testing.T) {
	t.Run("empty dst", func(t *testing.T) {
		archive := makeTarGz(t, nil)
		err := extractTarGz(archive, "")
		require.Error(t, err)
	})

	t.Run("huge tar", func(t *testing.T) {
		files := make([]*tar.Header, 0, 10000)
		for i := 0; i < 10000; i++ {
			files = append(files, &tar.Header{
				Name:     fmt.Sprintf("%d.txt", i),
				Typeflag: tar.TypeReg,
			})
		}

		dst, err := os.MkdirTemp("", "TestExtractTarGz")
		require.NoError(t, err)
		defer os.RemoveAll(dst)

		archive := makeTarGz(t, files)
		err = extractTarGz(archive, dst)
		require.NoError(t, err)
	})

	tests := []struct {
		files         []*tar.Header
		expectedError bool
		expectedFiles []string
	}{
		{
			[]*tar.Header{{Name: "../test/path", Typeflag: tar.TypeDir}},
			true,
			nil,
		},
		{
			[]*tar.Header{{Name: "../../test/path", Typeflag: tar.TypeDir}},
			true,
			nil,
		},
		{
			[]*tar.Header{{Name: "../../test/../path", Typeflag: tar.TypeDir}},
			true,
			nil,
		},
		{
			[]*tar.Header{{Name: "test/../../path", Typeflag: tar.TypeDir}},
			true,
			nil,
		},
		{
			[]*tar.Header{{Name: "test/path/../..", Typeflag: tar.TypeDir}},
			false,
			[]string{""},
		},
		{
			[]*tar.Header{{Name: "test", Typeflag: tar.TypeDir}},
			false,
			[]string{"", "test"},
		},
		{
			[]*tar.Header{
				{Name: "test", Typeflag: tar.TypeDir},
				{Name: "test/path", Typeflag: tar.TypeDir},
			},
			false,
			[]string{"", "test", "test/path"},
		},
		{
			[]*tar.Header{
				{Name: "test", Typeflag: tar.TypeDir},
				{Name: "test/path/", Typeflag: tar.TypeDir},
			},
			false,
			[]string{"", "test", "test/path"},
		},
		{
			[]*tar.Header{
				{Name: "test", Typeflag: tar.TypeDir},
				{Name: "test/path", Typeflag: tar.TypeDir},
				{Name: "test/path/file.ext", Typeflag: tar.TypeReg},
			},
			false,
			[]string{"", "test", "test/path", "test/path/file.ext"},
		},
		{
			[]*tar.Header{
				{Name: "/../../file.ext", Typeflag: tar.TypeReg},
			},
			true,
			nil,
		},
		{
			[]*tar.Header{
				{Name: "/../../link", Typeflag: tar.TypeLink},
			},
			false,
			[]string{""},
		},
		{
			[]*tar.Header{
				{Name: "..file", Typeflag: tar.TypeReg},
			},
			false,
			[]string{"", "..file"},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			dst, err := os.MkdirTemp("", "TestExtractTarGz")
			require.NoError(t, err)
			defer os.RemoveAll(dst)

			archive := makeTarGz(t, tt.files)
			err = extractTarGz(archive, dst)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assertDirectoryContents(t, dst, tt.expectedFiles)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	br := bytes.NewBuffer([]byte("foo"))

	// Under a non-existent directory
	err := copyFile(br, "/does/not/exist", 0644)
	require.Error(t, err)

	io.NopCloser(br)

	// Valid destination
	err = copyFile(br, filepath.Join(t.TempDir(), "bar"), 0644)
	require.NoError(t, err)
}
