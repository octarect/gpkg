package gpkg

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubRelease(t *testing.T) {
	type input struct {
		name string
		ref  string
	}
	tests := []struct {
		name     string
		input    input
		expected *GitHubRelease
		recvErr  bool
	}{
		{
			"valid",
			input{"foo/bar", "latest"},
			&GitHubRelease{"foo", "bar", "latest", nil},
			false,
		},
		{
			"wrong format of name",
			input{"foo", "latest"},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGitHubRelease(tt.input.name, tt.input.ref, nil)
			if tt.recvErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				checkDiff(t, GitHubRelease{}, tt.expected, got, "client")
			}
		})
	}
}

type mockRepositoriesService struct {
	servers []*httptest.Server
	data    *github.RepositoryRelease
	err     error
}

var _ releaseGetter = &mockRepositoriesService{}

func (s *mockRepositoriesService) GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
	return s.GetReleaseByTag(ctx, owner, repo, "latest")
}

func (s *mockRepositoriesService) GetReleaseByTag(ctx context.Context, owner, repo, ref string) (*github.RepositoryRelease, *github.Response, error) {
	if *s.data.TagName != ref {
		return nil, nil, fmt.Errorf("tag not found. tag=%s", ref)
	}
	if s.err != nil {
		return nil, nil, s.err
	}
	return s.data, nil, nil
}

func newMockRepositoriesService(tag string, assetNames []string) *mockRepositoriesService {
	svc := &mockRepositoriesService{}
	rel := &github.RepositoryRelease{}
	rel.TagName = &tag
	for _, name := range assetNames {
		srv := newTestServer("/"+name, 200, name)
		svc.servers = append(svc.servers, srv)
		u := fmt.Sprintf("%s/%s", srv.URL, name)
		rel.Assets = append(rel.Assets, &github.ReleaseAsset{
			Name:               &name,
			BrowserDownloadURL: &u,
		})
	}
	svc.data = rel
	return svc
}

func (s *mockRepositoriesService) Close() {
	for _, srv := range s.servers {
		srv.Close()
	}
}

func TestGitHubRelease_GetDownloader(t *testing.T) {
	type input struct {
		name string
		ref  string
	}
	tests := []struct {
		name     string
		input    input
		service  *mockRepositoriesService
		expected *HTTPDownloader
		recvErr  bool
	}{
		{
			"valid",
			input{"foo/bar", "latest"},
			newMockRepositoriesService(
				"latest",
				[]string{
					"foo-v1.0.0-x86_64-linux",
				},
			),
			&HTTPDownloader{
				name:  "foo-v1.0.0-x86_64-linux",
				total: int64(len("foo-v1.0.0-x86_64-linux")),
			},
			false,
		},
		{
			"no compatible asset exists",
			input{"foo/bar", "latest"},
			newMockRepositoriesService(
				"latest",
				[]string{
					"foo",
				},
			),
			nil,
			true,
		},
		{

			"tag not found",
			input{"foo/bar", "v9.9.9"},
			newMockRepositoriesService(
				"v1.0.0",
				[]string{
					"foo-v1.0.0-x86_64-linux",
				},
			),
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer tt.service.Close()

			ghr, err := NewGitHubRelease(tt.input.name, tt.input.ref, tt.service)
			require.NoError(t, err)
			got, err := ghr.GetDownloader()
			if tt.recvErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				checkDiff(t, HTTPDownloader{}, tt.expected, got, "ReadCloser")
			}
		})
	}
}
