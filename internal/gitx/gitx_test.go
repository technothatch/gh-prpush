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

// A same-repository PR must find the base repository's remote whatever it is
// called, not only one named "origin".
func TestPickRemoteSameRepoMatchesTheBaseRepository(t *testing.T) {
	remotes := []Remote{
		{Name: "upstream", URL: "https://github.com/technothatch/gh-prpush.git"},
		{Name: "origin", URL: "https://github.com/somebody/gh-prpush.git"},
	}
	rm, err := PickRemote(remotes, "technothatch/gh-prpush", "technothatch", false)
	if err != nil {
		t.Fatalf("PickRemote: %v", err)
	}
	if rm.Name != "upstream" {
		t.Errorf("expected the base repository's remote, got %q", rm.Name)
	}
}

// Without a match, "origin" stays the fallback so the common case is
// unchanged.
func TestPickRemoteSameRepoFallsBackToOrigin(t *testing.T) {
	remotes := []Remote{
		{Name: "upstream", URL: "https://github.com/other/repo.git"},
		{Name: "origin", URL: "https://github.com/technothatch/gh-prpush.git"},
	}
	rm, err := PickRemote(remotes, "technothatch/gh-prpush", "technothatch", false)
	if err != nil {
		t.Fatalf("PickRemote: %v", err)
	}
	if rm.Name != "origin" {
		t.Errorf("expected the origin fallback, got %q", rm.Name)
	}
}

// The match is case-insensitive, like GitHub itself.
func TestPickRemoteSameRepoMatchIsCaseInsensitive(t *testing.T) {
	remotes := []Remote{
		{Name: "Upstream", URL: "https://github.com/TechnoThatch/GH-PRPush.git"},
	}
	rm, err := PickRemote(remotes, "technothatch/gh-prpush", "technothatch", false)
	if err != nil {
		t.Fatalf("PickRemote: %v", err)
	}
	if rm.Name != "Upstream" {
		t.Errorf("expected the case-insensitive match, got %q", rm.Name)
	}
}

// A cross-repository PR still pushes to the fork, not to the base remote.
func TestPickRemoteCrossRepoPrefersTheFork(t *testing.T) {
	remotes := []Remote{
		{Name: "upstream", URL: "https://github.com/technothatch/gh-prpush.git"},
		{Name: "somebody", URL: "https://github.com/somebody/gh-prpush.git"},
	}
	rm, err := PickRemote(remotes, "technothatch/gh-prpush", "somebody", true)
	if err != nil {
		t.Fatalf("PickRemote: %v", err)
	}
	if rm.Name != "somebody" {
		t.Errorf("expected the fork remote, got %q", rm.Name)
	}
}
