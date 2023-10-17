package gpkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGitHubRelease(t *testing.T) {
	ref := "latest"
	type expected struct {
		owner string
		repo  string
	}
	ok, ng := false, true
	tests := map[string]struct {
		input        string
		expected     *expected
		expectsError bool
	}{
		"success": {"foo/bar", &expected{"foo", "bar"}, ok},
		"ng":      {"wrong", nil, ng},
	}

	for k, tt := range tests {
		t.Run(k, func(t *testing.T) {
			ghr, err := NewGitHubRelease(tt.input, ref)
			if tt.expectsError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				expected := &GitHubRelease{
					owner: tt.expected.owner,
					repo:  tt.expected.repo,
					ref:   ref,
				}
				checkDiff(t, GitHubRelease{}, ghr, expected, "client", "downloader", "length", "assetName")
			}
		})
	}
}
