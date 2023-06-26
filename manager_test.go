package gpkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	ss := []*PackageSpec{
		{
			From: "ghr",
			Name: "octarect/gpkg",
		},
	}
	m, err := NewManager(tmpDir, ss)
	require.NoError(t, err)

	assert.EqualValues(t, ss[0], m.Packages[0].GetSpec())

	pkgsPath := filepath.Join(tmpDir, "packages")
	assert.DirExists(t, pkgsPath)
}

type DummyPackage struct {
	spec      *PackageSpec
	LatestRef string
}

func (p *DummyPackage) Download(path string) error {
	return nil
}

func (p *DummyPackage) FetchLatestRef() (string, error) {
	return p.LatestRef, nil
}

func (p *DummyPackage) GetDirName() (string, error) {
	return p.spec.Name, nil
}

func (p *DummyPackage) GetSpec() *PackageSpec {
	return p.spec
}

func TestManager_GenerateScript(t *testing.T) {
	cachePath := "/cache"

	tests := []struct {
		name          string
		packages      []Package
		expectedLines []string
	}{
		{
			"empty",
			[]Package{},
			[]string{
				`export PATH="$PATH"`,
			},
		},
		{
			"one package",
			[]Package{
				&DummyPackage{
					spec: &PackageSpec{
						Name: "test00",
					},
				},
			},
			[]string{
				`export PATH="/cache/packages/test00:$PATH"`,
			},
		},
		{
			"multiple packages",
			[]Package{
				&DummyPackage{
					spec: &PackageSpec{
						Name: "test00",
					},
				},
				&DummyPackage{
					spec: &PackageSpec{
						Name: "test01",
					},
				},
			},
			[]string{
				`export PATH="/cache/packages/test00:/cache/packages/test01:$PATH"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				CachePath: cachePath,
				Packages:  tt.packages,
			}
			bs, err := m.GenerateScript()
			require.NoError(t, err)

			script := string(bs)
			for _, line := range tt.expectedLines {
				assert.Contains(t, script, line)
			}
		})
	}
}

func TestManager_ShouldUpdate(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	m := Manager{}
	m.CachePath = tmpDir

	tests := []struct {
		name       string
		currentRef string
		nextRef    string
		latestRef  string
		expected   bool
	}{
		{
			"new package",
			"",
			"v0.0.0",
			"",
			true,
		},
		{
			"new package latest",
			"",
			"latest",
			"v0.0.0",
			true,
		},
		{
			"specific version with no change",
			"v0.0.0",
			"v0.0.0",
			"",
			false,
		},
		{
			"specific version",
			"v0.0.0",
			"v0.0.1",
			"",
			true,
		},
		{
			"latest with no change",
			"v0.0.0",
			"latest",
			"v0.0.0",
			false,
		},
		{
			"latest",
			"v0.0.0",
			"latest",
			"v0.0.1",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &DummyPackage{
				spec: &PackageSpec{
					Name: tt.name,
				},
			}

			if tt.currentRef != "" {
				pkgDirName, err := pkg.GetDirName()
				require.NoError(t, err)

				refFile := filepath.Join(m.getPackagesDir(), pkgDirName, ".pkgref")

				err = os.MkdirAll(filepath.Dir(refFile), 0777)
				require.NoError(t, err)

				err = os.WriteFile(refFile, []byte(tt.currentRef), 0666)
				require.NoError(t, err)
			}

			pkg.spec.Ref = tt.nextRef
			if tt.nextRef == "latest" {
				pkg.LatestRef = tt.latestRef
			}

			result, err := m.shouldUpdate(pkg)
			require.NoError(t, err)
			assert.Equal(t, result, tt.expected)
		})
	}
}

func TestGetRef(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		latestRef string
	}{
		{
			"specific version",
			"v0.0.1",
			"",
		},
		{
			"latest",
			"latest",
			"v0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &DummyPackage{
				spec: &PackageSpec{
					Ref: tt.ref,
				},
				LatestRef: tt.latestRef,
			}
			ref, err := getRef(pkg)
			require.NoError(t, err)

			if tt.ref == "latest" {
				assert.EqualValues(t, ref, tt.latestRef)
			} else {
				assert.EqualValues(t, ref, tt.ref)
			}
		})
	}
}
