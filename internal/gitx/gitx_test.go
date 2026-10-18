package gitx

import "testing"

func TestGitHubOwnerRepo(t *testing.T) {
	cases := []struct {
		url         string
		owner, repo string
		ok          bool
	}{
		{"git@github.com:foo/bar.git", "foo", "bar", true},
		{"https://github.com/foo/bar", "foo", "bar", true},
		{"https://github.com/foo/bar.git", "foo", "bar", true},
		{"ssh://git@github.com/foo/bar.git", "foo", "bar", true},
		{"https://gitlab.com/foo/bar.git", "", "", false},
		{"not-a-url", "", "", false},
	}
	for _, c := range cases {
		o, r, ok := githubOwnerRepo(c.url)
		if ok != c.ok || o != c.owner || r != c.repo {
			t.Errorf("githubOwnerRepo(%q) = (%q,%q,%v), want (%q,%q,%v)", c.url, o, r, ok, c.owner, c.repo, c.ok)
		}
	}
}
