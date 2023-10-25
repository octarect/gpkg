package gpkg

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
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

func TestIsCompatibleAssetForMachine(t *testing.T) {
	osList := []struct {
		value string
		ids   []string
	}{
		{
			"linux",
			[]string{
				"linux",
			},
		},
		{
			"darwin",
			[]string{
				"darwin",
				"macos",
			},
		},
	}
	archList := []struct {
		value string
		ids   []string
	}{
		{
			"386",
			[]string{
				"i386",
			},
		},
		{
			"amd64",
			[]string{
				"amd64",
				"x86_64",
			},
		},
		{
			"arm",
			[]string{
				"arm",
			},
		},
		{
			"arm64",
			[]string{
				"arm64",
			},
		},
	}

	filesMatrix := make(map[string]map[string][]string)
	fileFormat := func(osID, archID string) string {
		return fmt.Sprintf("foo-v0.0.1-%s-%s", osID, archID)
	}
	for _, os := range osList {
		if _, ok := filesMatrix[os.value]; !ok {
			filesMatrix[os.value] = make(map[string][]string)
		}
		for _, arch := range archList {
			filesMatrix[os.value][arch.value] = []string{}
			for _, osID := range os.ids {
				for _, archID := range arch.ids {
					file := fileFormat(osID, archID)
					filesMatrix[os.value][arch.value] = append(filesMatrix[os.value][arch.value], file)
				}
			}
		}
	}

	type testCase struct {
		name         string
		goOS         string
		goArch       string
		expectedFile string
		files        []string
	}
	tests := []testCase{}
	for _, os := range osList {
		for _, arch := range archList {
			// Make a list of files other than the ID and the architecture to be tested.
			// Each test assumes that the file with IDs to be tested exists in addition to them.
			otherFiles := []string{}
			for osValue, osMap := range filesMatrix {
				for archValue, osArchFiles := range osMap {
					if osValue != os.value || archValue != arch.value {
						otherFiles = append(otherFiles, osArchFiles...)
					}
				}
			}

			for _, osID := range os.ids {
				for _, archID := range arch.ids {
					tc := testCase{
						name:         fmt.Sprintf("%s(%s) && %s(%s)", os.value, osID, arch.value, archID),
						goOS:         os.value,
						goArch:       arch.value,
						expectedFile: fileFormat(osID, archID),
					}
					tc.files = append(tc.files, otherFiles...)
					tc.files = append(tc.files, fileFormat(osID, archID))
					tests = append(tests, tc)
				}
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found string
			for _, file := range tt.files {
				if isCompatibleAssetForMachine(tt.goOS, tt.goArch, file) {
					found = file
					break
				}
			}
			if found != "" {
				assert.Equal(t, tt.expectedFile, found)
			} else {
				t.Errorf("No compatible asset found: %s", tt.expectedFile)
			}
		})
	}
}
