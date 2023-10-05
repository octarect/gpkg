package gpkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkdirTestPackage(t *testing.T, files []string) string {
	tmpPath, err := os.MkdirTemp("", "gpkg-test-*")
	require.NoError(t, err)
	for _, f := range files {
		p := filepath.Join(tmpPath, f)
		dir := filepath.Dir(p)
		// Create a parent directory if it doesn't exist
		err = os.MkdirAll(dir, 0777)
		require.NoError(t, err)
		// Create an empty file
		_, err = os.Create(p)
		require.NoError(t, err)
	}
	return tmpPath
}

func TestPickerDo(t *testing.T) {
	ok, ng := false, true
	files := []string{
		"foo-v0.0.1-x86_64/bar",
		"foo-v0.0.1-x86_64/scripts/boo.sh",
		"foo-v0.0.1-x86_64/scripts/boo.bash",
		"foo-v0.0.1-x86_64/scripts/boo.zsh",
		"baz",
		"qux-v0.0.1-x86_64-linux",
	}
	for _, tt := range []struct {
		title     string
		input     string
		expected  []string
		recvError bool
	}{
		{
			"pick a file",
			"foo-.*/bar",
			[]string{
				"bar",
			},
			ok,
		},
		{
			"pick multiple files",
			`foo-.*/boo\.(sh|bash|zsh)`,
			[]string{
				"boo.sh",
				"boo.bash",
				"boo.zsh",
			},
			ok,
		},
		{
			"directly under",
			"baz",
			[]string{
				"baz",
			},
			ok,
		},
		{
			"directory should be skipped",
			`foo-.*`,
			[]string{},
			ok,
		},
		{
			"no file matched with an expression",
			`invalid-expression`,
			nil,
			ng,
		},
		{
			"rename",
			`qux-.* -> qux`,
			[]string{
				"qux",
			},
			ok,
		},
	} {
		t.Run(tt.title, func(t *testing.T) {
			root := mkdirTestPackage(t, files)
			defer os.RemoveAll(root)

			err := Pick(root, tt.input)
			if tt.recvError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				for _, e := range tt.expected {
					assert.FileExists(t, filepath.Join(root, e))
				}
			}
		})
	}
}
