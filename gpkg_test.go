package gpkg

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPackage(t *testing.T) {
	tests := []struct {
		testName      string
		spec          *PackageSpec
		expectedError bool
		expectedType  interface{}
	}{
		{
			"ghr",
			&PackageSpec{
				From: "ghr",
				Name: "foo/bar",
			},
			false,
			&GHReleasePackage{},
		},
		{
			"empty spec",
			&PackageSpec{},
			true,
			nil,
		},
		{
			"invalid spec",
			&PackageSpec{
				From: "invalid",
			},
			true,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			pkg, err := newPackage(tt.spec)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.IsType(t, tt.expectedType, pkg)
			}
		})
	}
}

func mkdirTestPkg(paths []string) (string, error) {
	tmpPath, err := os.MkdirTemp("", "gpkg-test-*")
	if err != nil {
		return "", err
	}
	for _, p := range paths {
		p0 := filepath.Join(tmpPath, p)
		dir := filepath.Dir(p0)
		// Create a parent directory if it doesn't exist
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return "", err
		}
		// Create an empty file
		_, err = os.Create(p0)
		if err != nil {
			return "", err
		}
	}
	return tmpPath, nil
}

func TestNewPicker(t *testing.T) {
	for i, tt := range []struct {
		input    string
		expected []string
	}{
		{
			"foo",
			[]string{"foo", ""},
		},
		{
			"foo -> bar",
			[]string{"foo", "bar"},
		},
	} {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			p := NewPicker(tt.input)
			assert.EqualValues(t, p.lhs, tt.expected[0])
			assert.EqualValues(t, p.rhs, tt.expected[1])
		})
	}
}

func TestPickerDo(t *testing.T) {
	files := []string{
		"foo-v0.0.1-x86_64/bar",
		"foo-v0.0.1-x86_64/scripts/boo.sh",
		"foo-v0.0.1-x86_64/scripts/boo.bash",
		"foo-v0.0.1-x86_64/scripts/boo.zsh",
		"baz",
		"qux-v0.0.1-x86_64-linux",
	}

	for _, tt := range []struct {
		name      string
		input     string
		expected  []string
		recvError bool
	}{
		{
			"single",
			"foo-.*/bar",
			[]string{
				"bar",
			},
			false,
		},
		{
			"multiple",
			`foo-.*/boo\.(sh|bash|zsh)`,
			[]string{
				"boo.sh",
				"boo.bash",
				"boo.zsh",
			},
			false,
		},
		{
			"directly under",
			"baz",
			[]string{
				"baz",
			},
			false,
		},
		{
			"directory should be skipped",
			`foo-.*`,
			[]string{},
			false,
		},
		{
			"wrong pattern",
			`invalid-matcher`,
			[]string{},
			true,
		},
		{
			"rename",
			`qux-.* -> qux`,
			[]string{
				"qux",
			},
			false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root, err := mkdirTestPkg(files)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(root)

			err = NewPicker(tt.input).Do(root)
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
