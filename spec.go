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

type Spec struct {
	From string
	Name string
	Pick string
	Ref  string

	Source Source
}

func (s *Spec) Init() error {
	var src Source
	var err error

	switch s.From {
	case "ghr":
		src, err = NewGitHubRelease(s.Name, s.Ref)
	default:
		return fmt.Errorf("invalid spec. from=%s", s.From)
	}
	if err != nil {
		return fmt.Errorf("failed to init spec: %s", err)
	}
	s.Source = src

	return nil
}

func (s *Spec) GetDirName() string {
	return strings.Replace(s.Name, "/", "---", -1)
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

	cnt := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		rp, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		m := reg.Match([]byte(rp))
		if m {
			cnt++
			src, err := filepath.Abs(path)
			if err != nil {
				return err
			}

			var dst string
			if p.rhs != "" {
				dst = filepath.Join(root, p.rhs)
			} else {
				dst = filepath.Join(root, filepath.Base(path))
			}

			// Skip if a file with the same name already exists
			_, err = os.Stat(dst)
			if err == nil {
				// TODO: warn file already exists
				return nil
			}

			srcF, err := os.Open(src)
			if err != nil {
				return err
			}
			defer srcF.Close()

			dstF, err := os.Create(dst)
			if err != nil {
				return err
			}
			defer dstF.Close()

			_, err = io.Copy(dstF, srcF)
			if err != nil {
				return err
			}

			fi, err := srcF.Stat()
			if err != nil {
				return err
			}
			if err = os.Chmod(dst, fi.Mode()); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}
