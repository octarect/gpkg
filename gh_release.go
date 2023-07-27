package gpkg

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/google/go-github/v53/github"
)

type Release struct {
	ref    string
	assets map[string]string
}

type ReleaseHolder interface {
	GetRelease(context.Context, string) (*Release, error)
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

func (r *GitHubRepo) GetRelease(ctx context.Context, ref string) (*Release, error) {
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

	rel := &Release{}
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

type GHReleasePackage struct {
	Spec  *PackageSpec
	owner string
	repo  string

	releases ReleaseHolder
}

func NewGHReleasePackage(s *PackageSpec) (*GHReleasePackage, error) {
	// Extract owner and repo from name
	parts := strings.Split(s.Name, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("failed to get owner and repo")
	}
	owner := parts[0]
	repo := parts[1]

	return &GHReleasePackage{
		Spec:     s,
		owner:    owner,
		repo:     repo,
		releases: NewGitHubRepo(owner, repo),
	}, nil
}

func (p *GHReleasePackage) GetSpec() *PackageSpec {
	return p.Spec
}

func (p *GHReleasePackage) Download(path string) (dl io.ReadSeeker, name string, err error) {
	r, err := p.releases.GetRelease(context.Background(), p.Spec.Ref)
	if err != nil {
		err = fmt.Errorf("Failed to get release. ref=%s: %s", p.Spec.Ref, err)
		return
	}

	var url string
	for k, v := range r.assets {
		if isCompatibleRelease(k) {
			name = k
			url = v
			break
		}
	}
	if url == "" {
		err = fmt.Errorf("No compatible asset found. ref=%s", p.Spec.Ref)
		return
	}

	dl, err = download(url)
	return
}

func (p *GHReleasePackage) FetchLatestRef() (string, error) {
	r, err := p.releases.GetRelease(context.Background(), "latest")
	if err != nil {
		return "", fmt.Errorf("Failed to get latest release: %s", err)
	}
	return r.ref, nil
}

func (p *GHReleasePackage) GetDirName() (string, error) {
	return fmt.Sprintf("%s---%s", p.owner, p.repo), nil
}
