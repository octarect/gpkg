package gpkg

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type PackageSpec struct {
	From string
	Name string
	Pick string
	Ref  string
}

type Picker struct {
	lhs string
	rhs string
}

func NewPicker(s string) *Picker {
	p := &Picker{}
	l := strings.Split(s, "->")
	switch len(l) {
	case 2:
		p.rhs = strings.TrimSpace(l[1])
		fallthrough
	case 1:
		p.lhs = strings.TrimSpace(l[0])
	}
	return p
}

func (p *Picker) Do(root string) error {
	reg, err := regexp.Compile(fmt.Sprintf(`\A%s\z`, p.lhs))
	if err != nil {
		return err
	}

	fileCount := 0

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		// Skip directory
		if entry.IsDir() {
			return nil
		}

		rp, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		m := reg.Match([]byte(rp))
		if m {
			fileCount++

			to, err := filepath.Abs(path)
			if err != nil {
				return err
			}

			var from string
			if p.rhs != "" {
				from = filepath.Join(root, p.rhs)
			} else {
				from = filepath.Join(root, filepath.Base(path))
			}

			// Skip if a file with the same name already exists
			_, err = os.Stat(from)
			if err == nil {
				// TODO: warn file already exists
				return nil
			}

			err = os.Symlink(to, from)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}
	if fileCount == 0 {
		return fmt.Errorf("No files matching the pattern. pattern=%s.", string(p.lhs))
	}

	return nil
}

type Package interface {
	Download(string) (io.ReadSeeker, string, error)
	FetchLatestRef() (string, error)
	GetDirName() (string, error)
	GetSpec() *PackageSpec
}

func newPackage(s *PackageSpec) (pkg Package, err error) {
	switch s.From {
	case "ghr":
		pkg, err = NewGHReleasePackage(s)
	default:
		err = fmt.Errorf("Invalid value specified: from=%s", s.From)
	}
	return
}
