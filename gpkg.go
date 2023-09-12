package gpkg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/cheggaaa/pb/v3"

	cp "github.com/otiai10/copy"
)

type ReconcileError struct {
	Spec PackageSpec
	Err  error
}

func Reconcile(dir string, specs []PackageSpec) []*ReconcileError {
	bars := make([]*ProgressBar, len(specs))
	pool := pb.NewPool()
	for i := range specs {
		bars[i] = NewProgressBar(fmt.Sprintf("%d", i))
		pool.Add(bars[i].Bar)
	}

	var es []*ReconcileError

	wg := &sync.WaitGroup{}
	pool.Start()
	for i, spec := range specs {
		wg.Add(1)
		go func(spec PackageSpec, bar *ProgressBar, wg *sync.WaitGroup) {
			defer wg.Done()

			err := func() error {
				tmpDir, err := os.MkdirTemp("", "gpkg-*")
				if err != nil {
					return err
				}
				defer os.RemoveAll(tmpDir)

				src, err := getSource(spec)
				if err != nil {
					return err
				}
				dl, err := src.GetDownloader()
				if err != nil {
					return err
				}
				defer dl.Close()
				bar.SetTotal(dl.GetContentLength())

				r := io.TeeReader(dl, bar)

				if err = extract(r, tmpDir, dl.GetAssetName()); err != nil {
					return err
				}

				pkgCachePath := filepath.Join(dir, spec.GetDirName())
				if err = cp.Copy(tmpDir, pkgCachePath); err != nil {
					return err
				}

				if spec.Common().Pick != "" {
					if err := NewPicker(spec.Common().Pick).Do(pkgCachePath); err != nil {
						return err
					}
				}

				return nil
			}()

			if err != nil {
				es = append(es, &ReconcileError{
					Spec: spec,
					Err:  err,
				})
			}

		}(spec, bars[i], wg)
	}
	wg.Wait()
	pool.Stop()

	return es
}

func getSource(s PackageSpec) (Source, error) {
	switch r := s.(type) {
	case *GitHubReleaseSpec:
		return NewGitHubRelease(r.Name, r.Ref)
	default:
		return nil, fmt.Errorf("Unknown spec detected. type=%T", r)
	}
}
