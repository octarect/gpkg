package gpkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func download(u string) (*bytes.Reader, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("gpkg/%s", Version))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code was returned. expected=200, got=%d, url=%s", resp.StatusCode, u)
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	br := bytes.NewReader(bs)

	return br, nil
}

func extractTarGz(gzipStream io.Reader, dst string) error {
	if dst == "" {
		return errors.New("no destination path provided")
	}

	gzr, err := gzip.NewReader(gzipStream)
	if err != nil {
		return fmt.Errorf("failed to read next file from archive: %s", err)
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
			return fmt.Errorf("failed to sanitize path %s", th.Name)
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

			if err := copyFile(path, tr, th.FileInfo().Mode()); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(dstPath string, src io.Reader, perm fs.FileMode) error {
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
