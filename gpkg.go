package gpkg

import (
	"fmt"
	"io"
	"os"

	cp "github.com/otiai10/copy"
)

func ReconcilePackage(packagesDir string, states *StateData, spec PackageSpec, ch chan<- *Event, w io.Writer) error {
	ev := newEventBuilder(spec)
	ch <- ev.started()
	tmpDir, err := os.MkdirTemp("", "gpkg-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	src, err := getSource(spec)
	if err != nil {
		return err
	}

	var currentRef string
	if _, state, _ := states.FindState(spec); state != nil {
		currentRef = state.Ref
	}
	yes, nextRef, err := src.ShouldUpdate(currentRef)
	if !yes {
		ch <- ev.skipped(currentRef)
		return nil
	}

	dl, err := src.GetDownloader()
	if err != nil {
		return err
	}
	defer dl.Close()

	r := io.TeeReader(dl, w)

	ch <- ev.downloadStarted(dl, currentRef, nextRef)
	if err = extract(r, tmpDir, dl.GetAssetName()); err != nil {
		return err
	}
	ch <- ev.downloadCompleted()

	if err = cp.Copy(tmpDir, spec.PackagePath()); err != nil {
		return err
	}

	ch <- ev.pickStarted()
	if spec.Common().Pick != "" {
		if err := Pick(spec.PackagePath(), spec.Common().Pick); err != nil {
			return err
		}
	}

	states.Upsert(spec, nextRef)

	ch <- ev.completed()

	return nil
}

func getSource(s PackageSpec) (Source, error) {
	switch r := s.(type) {
	case *GitHubReleaseSpec:
		return NewGitHubRelease(r.Repo, r.Ref, nil)
	default:
		return nil, fmt.Errorf("Unknown spec detected. type=%T", r)
	}
}
