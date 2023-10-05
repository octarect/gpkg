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

func Pick(root, expr string) error {
	var lhs, rhs string
	l := strings.Split(expr, "->")
	switch len(l) {
	case 2:
		rhs = strings.TrimSpace(l[1])
		fallthrough
	case 1:
		lhs = strings.TrimSpace(l[0])
	}

	reg, err := regexp.Compile(fmt.Sprintf(`\A%s\z`, lhs))
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
			if rhs != "" {
				dst = filepath.Join(root, rhs)
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
