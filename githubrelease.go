package gpkg

import (
	"context"
	"fmt"
	"regexp"
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
		if isCompatibleAssetForMachine(runtime.GOOS, runtime.GOARCH, a.GetName()) {
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

var (
	osDarwinRe  = regexp.MustCompile(`(?i)(darwin|macos)`)
	archAmd64Re = regexp.MustCompile(`(?i)(amd64|x86_64)`)
)

func isCompatibleAssetForMachine(os, arch, assetName string) bool {
	hasOS := false
	switch os {
	case "darwin":
		hasOS = osDarwinRe.MatchString(assetName)
	default:
		hasOS = strings.Contains(assetName, os)
	}

	hasArch := false
	switch arch {
	case "amd64":
		hasArch = archAmd64Re.MatchString(assetName)
	case "arm":
		hasArch = strings.Contains(assetName, "arm") && !strings.Contains(assetName, "arm64")
	default:
		hasArch = strings.Contains(assetName, arch)
	}
	return hasOS && hasArch
}
