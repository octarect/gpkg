package gpkg

import (
	"bytes"
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
				testDir := setupTestDir(t)
				if _, err := os.Create(filepath.Join(testDir, expectedFileName)); err != nil {
					t.Fatal(err)
				}
				return testDir
			},
		},
		{
			"a config file does not exist yet",
			func(t *testing.T) string {
				return setupTestDir(t)
			},
		},
		{
			"create a directry if not exists",
			func(t *testing.T) string {
				testDir := setupTestDir(t)
				return filepath.Join(testDir, "autocreated")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := tt.setup(t)
			defer os.RemoveAll(testDir)
			cfgPath := filepath.Join(testDir, expectedFileName)

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
