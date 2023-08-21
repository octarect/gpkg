package gpkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
)

func extract(r io.Reader, path, name string) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	br := bytes.NewReader(b)

	ft, err := filetype.MatchReader(br)
	if err != nil {
		return err
	}

	br.Seek(0, io.SeekStart)

	switch ft.MIME.Value {
	case "application/gzip":
		return extractTarGz(br, path)
	default:
		return copyFile(br, filepath.Join(path, name), 0755)
	}
}

func copyFile(src io.Reader, dstPath string, perm fs.FileMode) error {
	f, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, src); err != nil {
		return err
	}
	return nil
}

func extractTarGz(r io.Reader, dst string) error {
	if dst == "" {
		return errors.New("no destination path provided.")
	}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %s", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		th, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read next file from archive: %s", err)
		}

		// Preemptively check type flag to avoid reporting a misleading error in
		// trying to sanitize the header name.
		switch th.Typeflag {
		case tar.TypeDir:
		case tar.TypeReg:
		default:
			// TODO: warn
			continue
		}

		// Ensure the target path remains rooted at dst and has no `../` escaping outside.
		path := filepath.Join(dst, th.Name)
		if !strings.HasPrefix(path, dst) {
			return fmt.Errorf("failed to sanitize path: %s", th.Name)
		}

		switch th.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(path, 0744); err != nil && !os.IsExist(err) {
				return err
			}
		case tar.TypeReg:
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0744); err != nil {
				return err
			}

			if err := copyFile(tr, path, th.FileInfo().Mode()); err != nil {
				return err
			}
		}
	}

	return nil
}
