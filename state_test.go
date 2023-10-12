package gpkg

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeStateData(t *testing.T) {
	ok, ng := false, true
	tests := []struct {
		name      string
		input     string
		expected  *StateData
		recvError bool
	}{
		{
			"accept valid json",
			`
			{
				"states": [
					{
						"spec": {
							"from": "ghr",
							"repo": "foo/bar"
						},
						"path": "/tmp/bin/bar"
					}
				]
			}
			`,
			&StateData{
				States: []State{
					{
						Spec: &GitHubReleaseSpec{
							CommonSpec: &CommonSpec{
								From:   "ghr",
								config: &Config{},
							},
							Repo: "foo/bar",
						},
						Path: "/tmp/bin/bar",
					},
				},
			},
			ok,
		},
		{
			"reject corrupted json",
			"{",
			nil,
			ng,
		},
		{
			"failed when the input is empty",
			"",
			nil,
			ng,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewBufferString(tt.input)
			got, err := DecodeStateData(r)
			if tt.recvError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.EqualValues(t, tt.expected, got)
			}
		})
	}
}

func TestLoadStateDataFromFile(t *testing.T) {
	t.Run("return an empty data when the state file doesn't exist", func(t *testing.T) {
		got, err := LoadStateDataFromFile("/tmp/dummy")
		require.NoError(t, err)
		assert.EqualValues(t, &StateData{}, got)
	})
	t.Run("valid json", func(t *testing.T) {
		testDir := setupTestDir(t)
		defer os.RemoveAll(testDir)
		statePath := filepath.Join(testDir, "states.json")
		json := `
		{
			"states": [
				{
					"spec": {
						"from": "ghr",
						"repo": "foo/bar"
					},
					"path": "/tmp/foo/bar",
					"ref": "v1"
				}
			]
		}
		`
		err := os.WriteFile(statePath, []byte(json), 0666)
		require.NoError(t, err)

		_, err = LoadStateDataFromFile(statePath)
		require.NoError(t, err)
	})
	t.Run("invalid json", func(t *testing.T) {
		testDir := setupTestDir(t)
		defer os.RemoveAll(testDir)
		statePath := filepath.Join(testDir, "states.json")
		json := `xxxx`
		err := os.WriteFile(statePath, []byte(json), 0666)
		require.NoError(t, err)

		_, err = LoadStateDataFromFile(statePath)
		require.Error(t, err)
	})
}

func TestStateData_Save(t *testing.T) {
	sd := &StateData{
		States: []State{
			{
				Spec: NewNopSpec("foo"),
			},
		},
	}
	buf := bytes.NewBuffer([]byte{})
	err := sd.Save(buf)
	require.NoError(t, err)
}

func TestStateData_SaveToFile(t *testing.T) {
	sd := &StateData{
		States: []State{
			{
				Spec: NewNopSpec("foo"),
			},
		},
	}
	t.Run("create a new state file", func(t *testing.T) {
		testDir := setupTestDir(t)
		defer os.RemoveAll(testDir)

		statePath := filepath.Join(testDir, "states.json")
		err := sd.SaveToFile(statePath)
		require.NoError(t, err)
	})
	t.Run("already exists", func(t *testing.T) {
		testDir := setupTestDir(t)
		defer os.RemoveAll(testDir)

		statePath := filepath.Join(testDir, "states.json")
		err := os.WriteFile(statePath, []byte(""), 0666)
		require.NoError(t, err)
		err = sd.SaveToFile(statePath)
		require.NoError(t, err)
	})
}

func TestStateData_FindState(t *testing.T) {
	tests := []struct {
		name          string
		input         PackageSpec
		sd            *StateData
		expectedIndex int
	}{
		{
			"no state exists",
			NewNopSpec("foo"),
			&StateData{
				States: []State{},
			},
			-1,
		},
		{
			"no state found",
			NewNopSpec("foo"),
			&StateData{
				States: []State{
					{
						Spec: NewNopSpec("bar"),
					},
				},
			},
			-1,
		},
		{
			"found",
			NewNopSpec("foo"),
			&StateData{
				States: []State{
					{
						Spec: NewNopSpec("bar"),
					},
					{
						Spec: NewNopSpec("foo"),
					},
				},
			},
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, st, err := tt.sd.FindState(tt.input)
			if tt.expectedIndex == -1 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedIndex, idx)
				assert.EqualValues(t, &tt.sd.States[tt.expectedIndex], st)
			}
		})
	}
}

func TestStateData_Upsert(t *testing.T) {
	foo := NewNopSpec("foo")
	foo.CommonSpec.Ref = "v1"
	fooV2 := NewNopSpec("foo")
	fooV2.CommonSpec.Ref = "v2"
	bar := NewNopSpec("bar")
	tests := []struct {
		name        string
		input       PackageSpec
		initialData StateData
		expected    StateData
	}{
		{
			"add a first state",
			foo,
			StateData{
				States: []State{},
			},
			StateData{
				States: []State{
					{
						Spec: foo,
						Path: foo.PackagePath(),
					},
				},
			},
		},
		{
			"add a state",
			foo,
			StateData{
				States: []State{
					{
						Spec: bar,
						Path: bar.PackagePath(),
					},
				},
			},
			StateData{
				States: []State{
					{
						Spec: bar,
						Path: bar.PackagePath(),
					},
					{
						Spec: foo,
						Path: foo.PackagePath(),
					},
				},
			},
		},
		{
			"update an existing state",
			fooV2,
			StateData{
				States: []State{
					{
						Spec: foo,
						Path: foo.PackagePath(),
					},
				},
			},
			StateData{
				States: []State{
					{
						Spec: fooV2,
						Path: fooV2.PackagePath(),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.initialData
			d.Upsert(tt.input, tt.input.Common().Ref)
			assert.EqualValues(t, tt.expected, d)
		})
	}
}
