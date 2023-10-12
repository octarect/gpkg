package gpkg

import (
	"fmt"
	"io"
	"os"

	cp "github.com/otiai10/copy"
)

func ReconcilePackage(packagesDir string, states *StateData, spec PackageSpec, ch chan<- *Event, w io.Writer) error {
	builder := newEventBuilder(spec)
	ch <- builder.started()
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

	r := io.TeeReader(dl, w)

	ch <- builder.downloadStarted(dl)
	if err = extract(r, tmpDir, dl.GetAssetName()); err != nil {
		return err
	}
	ch <- builder.downloadCompleted()

	if err = cp.Copy(tmpDir, spec.PackagePath()); err != nil {
		return err
	}

	ch <- builder.pickStarted()
	if spec.Common().Pick != "" {
		if err := Pick(spec.PackagePath(), spec.Common().Pick); err != nil {
			return err
		}
	}

	states.Upsert(spec, spec.Common().Ref)

	ch <- builder.completed()

	return nil
}

func getSource(s PackageSpec) (Source, error) {
	switch r := s.(type) {
	case *GitHubReleaseSpec:
		return NewGitHubRelease(r.Repo, r.Ref)
	default:
		return nil, fmt.Errorf("Unknown spec detected. type=%T", r)
	}
}
