package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/google/go-github/v53/github"
	"github.com/h2non/filetype"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cp "github.com/otiai10/copy"
)

type Config struct {
	CachePath string `mapstructure:"cache_path"`
	Specs     []*Spec `mapstructure:"packages"`
}
var cfg Config

func (c *Config) GetPackagesPath() string {
	return path.Join(c.CachePath, "packages")
}

var (
	version = "v0.1.0"
	rootCmd = &cobra.Command{
		Use: "gpkg",
		Short: "A general package manager",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := "./config.toml"
			viper.SetConfigFile(cfgPath)
			if err := viper.ReadInConfig(); err != nil {
				return err
			}
			if err := viper.Unmarshal(&cfg); err != nil {
				return err
			}
			if cfg.CachePath == "" {
				usrCacheDir, err := os.UserCacheDir()
				if err != nil {
					return err
				}
				cfg.CachePath = filepath.Join(usrCacheDir, "gpkg")
			}
			return nil
		},
	}
	versionCmd = &cobra.Command{
		Use: "version",
		Short: "Print the version of gpkg",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
	updateCmd = &cobra.Command{
		Use: "update",
		Short: "Install or update packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := commandUpdate(); err != nil {
				return err
			}
			return nil
		},
	}
)

func main() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
	return
}

func commandUpdate() error {
	for _, spec := range cfg.Specs {
		if err := spec.Init(); err != nil {
			return err
		}
	}

	es := reconcile(cfg.Specs)
	if len(es) > 0 {
		PrintReconcileErrors(es)
		return errors.New("failed to update packages")
	}

	return nil
}

var errorFormat = `
Error updating %s:
  => %s
`

func PrintReconcileErrors(es []*ReconcileError) {
	for _, e := range es {
		fmt.Fprintf(os.Stderr, strings.TrimSpace(errorFormat), e.spec.Name, e.err)
		fmt.Println()
	}
}

type ReconcileError struct {
	spec *Spec
	err  error
}

func reconcile(specs []*Spec) ([]*ReconcileError) {
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
		go func(spec *Spec, bar *ProgressBar, wg *sync.WaitGroup) {
			defer wg.Done()

			err := func() error {
				tmpDir, err := os.MkdirTemp("", "gpkg-*")
				if err != nil {
					return err
				}
				defer os.RemoveAll(tmpDir)

				dl, err := spec.Package.GetDownloader()
				if err != nil {
					return err
				}
				defer dl.Close()
				bar.SetTotal(dl.GetContentLength())

				r := io.TeeReader(dl, bar)

				if err = extract(r, tmpDir, dl.GetAssetName()); err != nil {
					return err
				}

				pkgCachePath := filepath.Join(cfg.GetPackagesPath(), spec.GetDirName())
				if err = cp.Copy(tmpDir, pkgCachePath); err != nil {
					return err
				}

				if spec.Pick != "" {
					if err := NewPicker(spec.Pick).Do(pkgCachePath); err != nil {
						return err
					}
				}

				return nil
			}()

			if err != nil {
				es = append(es, &ReconcileError{
					spec: spec,
					err: err,
				})
			}

		}(spec, bars[i], wg)
	}
	wg.Wait()
	pool.Stop()

	return es
}

type Package interface {
	GetDownloader() (Downloader, error)
}

type HTTP struct {
	url string
}

func NewHTTP(url string) (*HTTP, error) {
	return &HTTP{
		url: url,
	}, nil
}

func (h *HTTP) GetDownloader() (Downloader, error) {
	req, err := http.NewRequest(http.MethodGet, h.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gpkg/v0.0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code was returned. expected=200, got=%d, url=%s", resp.StatusCode, h.url)
	}

	return NewWrapperDownloader(resp.Body, path.Base(req.URL.Path), resp.ContentLength), nil
}

type ProgressBar struct {
	Bar *pb.ProgressBar
}

func NewProgressBar(name string) *ProgressBar {
		bar := pb.Full.New(0)
		bar.SetRefreshRate(time.Millisecond * 500)
		bar.Set(pb.Bytes, true)
		bar.Set("prefix", fmt.Sprintf("%s: ", name))

		return &ProgressBar{
			Bar: bar,
		}
}

func (b *ProgressBar) SetTotal(n int64) {
	b.Bar.SetTotal(n)
}

func (b *ProgressBar) Write(data []byte) (int, error) {
	n := len(data)
	b.Bar.Add(n)
	return n, nil
}

type ReleaseData struct {
	ref string
	assets map[string]string
}

type ReleaseFetcher interface {
	Fetch(context.Context, string) (*ReleaseData, error)
}

type GitHubRepo struct {
	owner  string
	repo   string
	client *github.Client
}

func NewGitHubRepo(owner, repo string) *GitHubRepo {
	return &GitHubRepo{
		owner: owner,
		repo: repo,
		client: github.NewClient(nil),
	}
}

func (r *GitHubRepo) GetRelease(ctx context.Context, ref string) (*ReleaseData, error) {
	var rr *github.RepositoryRelease
	var err error

	if ref == "latest" || ref == "" {
		rr, _, err = r.client.Repositories.GetLatestRelease(ctx, r.owner, r.repo)
	} else {
		rr, _, err = r.client.Repositories.GetReleaseByTag(ctx, r.owner, r.repo, ref)
	}
	if err != nil {
		return nil, err
	}

	rel := &ReleaseData{}
	rel.ref = rr.GetTagName()
	rel.assets = make(map[string]string, len(rr.Assets))
	for _, a := range rr.Assets {
		rel.assets[a.GetName()] = a.GetBrowserDownloadURL()
	}

	return rel, nil
}

type GitHubRelease struct {
	owner string
	repo  string
	ref   string

	downloader io.ReadCloser
	length int64
	assetName string

	once sync.Once
}

func NewGitHubRelease(name, ref string) (*GitHubRelease, error){
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("failed to get owner and repo")
	}
	owner := parts[0]
	repo := parts[1]

	return &GitHubRelease{
		owner: owner,
		repo: repo,
		ref: ref,
	}, nil
}

func (ghr *GitHubRelease) Read(p []byte) (int, error) {
	var err error
	ghr.once.Do(func() {
		gr := NewGitHubRepo(ghr.owner, ghr.repo)
		r, err := gr.GetRelease(context.Background(), ghr.ref)
		if err != nil {
			return
		}
		var name, url string
		for k, v := range r.assets {
			if isCompatibleRelease(k) {
				name = k
				url = v
				break
			}
		}
		if name == "" {
			err = fmt.Errorf("No compatible asset found. ref=%s", ghr.ref)
			return
		}

		resp, err := download(url)
		if err != nil {
			return
		}
		ghr.downloader = resp.Body
		ghr.assetName = name
		ghr.length = resp.ContentLength
	})
	if err != nil {
		return 0, err
	}
	return ghr.downloader.Read(p)
}

func (ghr *GitHubRelease) GetDownloader() (Downloader, error) {
	gr := NewGitHubRepo(ghr.owner, ghr.repo)
	r, err := gr.GetRelease(context.Background(), ghr.ref)
	if err != nil {
		return nil, err
	}
	var name, url string
	for k, v := range r.assets {
		if isCompatibleRelease(k) {
			name = k
			url = v
			break
		}
	}
	if name == "" {
		return nil, fmt.Errorf("No compatible asset found. ref=%s", ghr.ref)
	}

	resp, err := download(url)
	if err != nil {
		return nil, err
	}
	return NewWrapperDownloader(resp.Body, name, resp.ContentLength), nil
}

type Downloader interface {
	io.ReadCloser
	GetAssetName() string
	GetContentLength() int64
}

type WrapperDownloader struct {
	r     io.ReadCloser
	name  string
	total int64
}

func NewWrapperDownloader(r io.ReadCloser, name string, total int64) *WrapperDownloader {
	return &WrapperDownloader{
		r: r,
		name: name,
		total: total,
	}
}

func (dl *WrapperDownloader) Read(p []byte) (int, error) {
	return dl.r.Read(p)
}

func (dl *WrapperDownloader) Close() error {
	return dl.r.Close()
}

func (dl *WrapperDownloader) GetAssetName() string {
	return dl.name
}

func (dl *WrapperDownloader) GetContentLength() int64 {
	return dl.total
}

func isCompatibleRelease(s string) bool {
	// TODO: At this point, only amd64 is supported.
	hasOS := strings.Contains(s, runtime.GOOS)
	hasArch := strings.Contains(s, runtime.GOARCH) || strings.Contains(s, "x86_64")
	return hasOS && hasArch
}

func download(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("gpkg/%s", "0.0.1"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

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

	switch(ft.MIME.Value) {
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

type Spec struct {
	From string 
	Name string
	Pick string
	Ref  string

	Package Package
}

func (s *Spec) Init() error {
	var pkg Package
	var err error

	switch(s.From) {
	case "ghr":
		pkg, err = NewGitHubRelease(s.Name, s.Ref)
	default:
		return fmt.Errorf("invalid spec. from=%s", s.From)
	}
	if err != nil {
		return fmt.Errorf("failed to init spec: %s", err)
	}
	s.Package = pkg

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
