package gpkg

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	cp "github.com/otiai10/copy"
)

type Manager struct {
	CachePath string
	Packages  []Package
}

func NewManager(cachePath string, ss []*PackageSpec) (*Manager, error) {
	m := &Manager{}
	m.CachePath = cachePath

	for _, path := range []string{m.CachePath, m.getPackagesDir()} {
		if _, err := os.Stat(path); err != nil {
			if err := os.MkdirAll(path, 0777); err != nil {
				return nil, err
			}
		}
	}

	pkgs := make([]Package, len(ss))
	for i, s := range ss {
		pkg, err := newPackage(s)
		if err != nil {
			return nil, err
		}
		pkgs[i] = pkg
	}
	m.Packages = pkgs

	return m, nil
}

func (m *Manager) getPackagesDir() string {
	return filepath.Join(m.CachePath, "packages")
}

func (m *Manager) GenerateScript() ([]byte, error) {
	paths := make([]string, 0, len(m.Packages)+1)
	for _, pkg := range m.Packages {
		pkgDirName, err := pkg.GetDirName()
		if err != nil {
			return nil, err
		}
		paths = append(paths, filepath.Join(m.getPackagesDir(), pkgDirName))
	}
	paths = append(paths, "$PATH")

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	buf.WriteString(fmt.Sprintf(`export PATH="%s"`, strings.Join(paths, ":")))
	buf.WriteString("\r\n")

	return buf.Bytes(), nil
}

func (m *Manager) UpdatePackages(force bool) error {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	type Result struct {
		spec *PackageSpec
		err  error
	}
	results := make([]*Result, 0, len(m.Packages))
	appendResult := func(pkg Package, err error) {
		mu.Lock()
		results = append(results, &Result{
			spec: pkg.GetSpec(),
			err: err,
		})
		mu.Unlock()
	}

	ui := NewProgressUI()

	for _, pkg := range m.Packages {
		wg.Add(1)

		bar := ui.AddProgressBar(pkg.GetSpec().Name)
		log.SetOutput(bar)

		go func(pkg Package) {
			defer wg.Done()
			tmpDir, err := os.MkdirTemp("", "gpkg-*")
			if err != nil {
				appendResult(pkg, err)
				return
			}
			defer os.RemoveAll(tmpDir)

			yes := true
			if !force {
				yes, err = m.shouldUpdate(pkg)
				if err != nil {
					appendResult(pkg, err)
					return
				}
			}
			if yes {
				if err = pkg.Download(tmpDir, bar); err != nil {
					appendResult(pkg, err)
					return
				}

				ref, err := getRef(pkg)
				if err != nil {
					appendResult(pkg, err)
					return
				}
				refFile := filepath.Join(tmpDir, ".pkgref")
				if err = os.WriteFile(refFile, []byte(ref), 0666); err != nil {
					appendResult(pkg, err)
					return
				}

				pkgDirName, err := pkg.GetDirName()
				if err != nil {
					appendResult(pkg, err)
					return
				}
				pkgCacheDir := filepath.Join(m.getPackagesDir(), pkgDirName)
				if err = cp.Copy(tmpDir, pkgCacheDir); err != nil {
					appendResult(pkg, err)
					return
				}

				if pkg.GetSpec().Pick != "" {
					if err := NewPicker(pkg.GetSpec().Pick).Do(pkgCacheDir); err != nil {
						appendResult(pkg, err)
						return
					}
				}
			}

			appendResult(pkg, nil)
		}(pkg)
	}

	wg.Wait()

	for _, r := range results {
		fmt.Printf("%#v\n", r)
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "Package \"%s\":\n", r.spec.Name)
			fmt.Fprintf(os.Stderr, "\t%s\n", r.err)
		}
	}

	return nil
}

func (m *Manager) shouldUpdate(pkg Package) (update bool, err error) {
	nextRef, err := getRef(pkg)

	pkgDirName, err := pkg.GetDirName()
	if err != nil {
		return
	}
	refFile := filepath.Join(m.getPackagesDir(), pkgDirName, ".pkgref")
	bs, err := os.ReadFile(refFile)
	if err != nil {
		return true, nil
	}
	curRef := string(bs)

	update = (curRef != nextRef)

	return
}

func getRef(pkg Package) (ref string, err error) {
	s := pkg.GetSpec()
	if s.Ref != "" && s.Ref != "latest" {
		ref = s.Ref
	} else {
		ref, err = pkg.FetchLatestRef()
	}
	return
}
