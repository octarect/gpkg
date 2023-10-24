package gpkg

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/google/go-github/v53/github"
	"github.com/gregjones/httpcache"
)

type GitHubRelease struct {
	owner  string
	repo   string
	ref    string
	client releaseGetter
}

type releaseGetter interface {
	GetLatestRelease(context.Context, string, string) (*github.RepositoryRelease, *github.Response, error)
	GetReleaseByTag(context.Context, string, string, string) (*github.RepositoryRelease, *github.Response, error)
}

var _ releaseGetter = &github.RepositoriesService{}

func NewGitHubRelease(name, ref string, client releaseGetter) (*GitHubRelease, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("failed to get owner and repo")
	}
	owner := parts[0]
	repo := parts[1]

	if client == nil {
		client = github.NewClient(httpcache.NewMemoryCacheTransport().Client()).Repositories
	}

	return &GitHubRelease{
		owner:  owner,
		repo:   repo,
		ref:    ref,
		client: client,
	}, nil
}

func (ghr *GitHubRelease) GetDownloader() (Downloader, error) {
	var err error
	var rr *github.RepositoryRelease
	if ghr.ref == "latest" || ghr.ref == "" {
		rr, _, err = ghr.client.GetLatestRelease(context.Background(), ghr.owner, ghr.repo)
	} else {
		rr, _, err = ghr.client.GetReleaseByTag(context.Background(), ghr.owner, ghr.repo, ghr.ref)
	}
	if err != nil {
		return nil, err
	}

	var name, url string
	for _, a := range rr.Assets {
		if isCompatibleRelease(a.GetName()) {
			name = a.GetName()
			url = a.GetBrowserDownloadURL()
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
		rr, _, err := ghr.client.GetLatestRelease(context.Background(), ghr.owner, ghr.repo)
		if err != nil {
			return false, "", err
		}
		return rr.GetTagName() != currentRef, rr.GetTagName(), nil
	}
	return ghr.ref != currentRef, ghr.ref, nil
}

func isCompatibleRelease(s string) bool {
	// TODO: At this point, only amd64 is supported.
	hasOS := strings.Contains(s, runtime.GOOS)
	hasArch := strings.Contains(s, runtime.GOARCH) || strings.Contains(s, "x86_64")
	return hasOS && hasArch
}
