package gpkg

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/google/go-github/v53/github"
	"github.com/gregjones/httpcache"
)

type GitHubRelease struct {
	owner string
	repo  string
	ref   string

	client     *github.Client
	downloader io.ReadCloser
	length     int64
	assetName  string
}

func NewGitHubRelease(name, ref string) (*GitHubRelease, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("failed to get owner and repo")
	}
	owner := parts[0]
	repo := parts[1]

	return &GitHubRelease{
		owner:  owner,
		repo:   repo,
		ref:    ref,
		client: github.NewClient(httpcache.NewMemoryCacheTransport().Client()),
	}, nil
}

func (ghr *GitHubRelease) GetDownloader() (Downloader, error) {
	r, err := ghr.fetchRelease(context.Background())
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

func (ghr *GitHubRelease) ShouldUpdate(currentRef string) (bool, string, error) {
	if ghr.ref == "latest" || ghr.ref == "" {
		rr, _, err := ghr.client.Repositories.GetLatestRelease(context.Background(), ghr.owner, ghr.repo)
		if err != nil {
			return false, "", err
		}
		return rr.GetTagName() != currentRef, rr.GetTagName(), nil
	}
	return ghr.ref != currentRef, ghr.ref, nil
}

type ReleaseData struct {
	ref    string
	assets map[string]string
}

func (ghr *GitHubRelease) fetchRelease(ctx context.Context) (*ReleaseData, error) {
	var rr *github.RepositoryRelease
	var err error

	if ghr.ref == "latest" || ghr.ref == "" {
		rr, _, err = ghr.client.Repositories.GetLatestRelease(ctx, ghr.owner, ghr.repo)
	} else {
		rr, _, err = ghr.client.Repositories.GetReleaseByTag(ctx, ghr.owner, ghr.repo, ghr.ref)
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
