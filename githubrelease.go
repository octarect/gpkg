package gpkg

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"

	"github.com/google/go-github/v53/github"
)

type ReleaseData struct {
	ref    string
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
		owner:  owner,
		repo:   repo,
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

func isCompatibleRelease(s string) bool {
	// TODO: At this point, only amd64 is supported.
	hasOS := strings.Contains(s, runtime.GOOS)
	hasArch := strings.Contains(s, runtime.GOARCH) || strings.Contains(s, "x86_64")
	return hasOS && hasArch
}

type GitHubRelease struct {
	owner string
	repo  string
	ref   string

	downloader io.ReadCloser
	length     int64
	assetName  string

	once sync.Once
}

func NewGitHubRelease(name, ref string) (*GitHubRelease, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("failed to get owner and repo")
	}
	owner := parts[0]
	repo := parts[1]

	return &GitHubRelease{
		owner: owner,
		repo:  repo,
		ref:   ref,
	}, nil
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
	dl, err := NewHTTPDownloader(name, url)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a downloader. err=%s", err)
	}
	return dl, nil
}
