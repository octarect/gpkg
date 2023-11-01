package gpkg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateConfigFile(t *testing.T) {
	expectedFileName := "config.toml"
	tests := []struct {
		name  string
		setup func(t *testing.T) (cfgPath string)
	}{
		{
			"already exists",
			func(t *testing.T) string {
				d := t.TempDir()
				if _, err := os.Create(filepath.Join(d, expectedFileName)); err != nil {
					t.Fatal(err)
				}
				return d
			},
		},
		{
			"a config file does not exist yet",
			func(t *testing.T) string {
				return t.TempDir()
			},
		},
		{
			"create a directry if not exists",
			func(t *testing.T) string {
				d := t.TempDir()
				return filepath.Join(d, "autocreated")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.setup(t)
			cfgPath := filepath.Join(d, expectedFileName)

			err := CreateConfigFile(cfgPath)
			require.NoError(t, err)

			got, err := os.ReadFile(cfgPath)
			expected := bytes.NewBuffer([]byte{})
			tmpl, _ := template.ParseFS(tmplFS, "templates/new_config.toml.tmpl")
			if err = tmpl.Execute(expected, "dummy"); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, expected.String(), string(got))
		})
	}
}

func TestSpecEqual(t *testing.T) {
	tests := []struct {
		a        PackageSpec
		b        PackageSpec
		expected bool
	}{
		{NewNopSpec("foo"), NewNopSpec("foo"), true},
		{NewNopSpec("foo"), NewNopSpec("bar"), false},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("test%02d", i), func(t *testing.T) {
			if got := SpecEqual(tt.a, tt.b); got != tt.expected {
				t.Errorf("Unexpected value returned. expected=%v, got=%v", tt.expected, got)
			}
		})
	}
}
