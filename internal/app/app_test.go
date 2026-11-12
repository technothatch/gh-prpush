package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/technothatch/gh-prpush/internal/run"
)

func fake(calls map[string]struct {
	Out  string
	Err  string
	Fail bool
}) run.Runner {
	return run.Fake(calls)
}

const prSameRepo = `{"headRefName":"feature","isCrossRepository":false,"headRepositoryOwner":{"login":"technothatch"}}`

const prCrossRepo = `{"headRefName":"fix-thing","isCrossRepository":true,"headRepositoryOwner":{"login":"somebody"}}`

const remoteOutput = `origin	git@github.com:technothatch/gh-prpush.git (fetch)
origin	git@github.com:technothatch/gh-prpush.git (push)
somebody	git@github.com:somebody/gh-prpush.git (fetch)
somebody	git@github.com:somebody/gh-prpush.git (push)
`

const remoteOutputNoOwnerRemote = `origin	https://github.com/technothatch/gh-prpush.git (fetch)
origin	https://github.com/technothatch/gh-prpush.git (push)
fork	https://github.com/somebody/gh-prpush.git (fetch)
fork	https://github.com/somebody/gh-prpush.git (push)
`

func TestPushSameRepoUsesOrigin(t *testing.T) {
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {Out: prSameRepo},
		"git remote -v": {Out: remoteOutput},
		"git push origin HEAD:refs/heads/feature": {Out: "everything up-to-date\n"},
	})
	var out strings.Builder
	if err := Run(nil, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "origin") || !strings.Contains(out.String(), "feature") {
		t.Errorf("unexpected output: %q", out.String())
	}
}

func TestPushCrossRepoMatchesOwnerURL(t *testing.T) {
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {Out: prCrossRepo},
		"git remote -v": {Out: remoteOutputNoOwnerRemote},
		"git push fork HEAD:refs/heads/fix-thing": {Out: "ok\n"},
	})
	var out strings.Builder
	if err := Run(nil, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "fork") {
		t.Errorf("expected fork remote, got: %q", out.String())
	}
}

func TestPushCrossRepoMatchesOwnerName(t *testing.T) {
	remotes := `origin	git@github.com:technothatch/gh-prpush.git (fetch)
origin	git@github.com:technothatch/gh-prpush.git (push)
somebody	ssh://git@internal.example/somebody/repo.git (fetch)
somebody	ssh://git@internal.example/somebody/repo.git (push)
`
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {Out: prCrossRepo},
		"git remote -v": {Out: remotes},
		"git push somebody HEAD:refs/heads/fix-thing": {Out: "ok\n"},
	})
	var out strings.Builder
	if err := Run(nil, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "somebody") {
		t.Errorf("expected somebody remote, got: %q", out.String())
	}
}

func TestNoPRError(t *testing.T) {
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {Err: "no pull requests found for branch \"main\"", Fail: true},
	})
	var out strings.Builder
	err := Run(nil, &out, &out, r)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gh pr create") {
		t.Errorf("error should suggest gh pr create: %v", err)
	}
}

func TestForceWithLease(t *testing.T) {
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {Out: prSameRepo},
		"git remote -v": {Out: remoteOutput},
		"git push origin HEAD:refs/heads/feature --force-with-lease": {Out: "ok\n"},
	})
	var out strings.Builder
	if err := Run([]string{"--force-with-lease"}, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

const prCrossRepoWithRepo = `{"headRefName":"fix-thing","isCrossRepository":true,"headRepositoryOwner":{"login":"somebody"},"headRepository":{"nameWithOwner":"somebody/gh-prpush"}}`

const remoteOutputNoFork = `origin	git@github.com:technothatch/gh-prpush.git (fetch)
origin	git@github.com:technothatch/gh-prpush.git (push)
`

const remoteOutputWithFork = `origin	git@github.com:technothatch/gh-prpush.git (fetch)
origin	git@github.com:technothatch/gh-prpush.git (push)
somebody	git@github.com:somebody/gh-prpush.git (fetch)
somebody	git@github.com:somebody/gh-prpush.git (push)
`

// Without --add-remote a missing fork remote is still an error: adding a
// remote changes the user's repository, so it must be asked for.
func TestMissingForkRemoteIsAnErrorByDefault(t *testing.T) {
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {Out: prCrossRepoWithRepo},
		"git remote -v": {Out: remoteOutputNoFork},
	})
	var out strings.Builder
	err := Run(nil, &out, &out, r)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no remote found for fork owner") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --add-remote adds the fork as a remote named after its owner, then pushes.
func TestAddRemoteAddsTheForkAndPushes(t *testing.T) {
	r := run.FakeSequence(map[string][]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {{Out: prCrossRepoWithRepo}},
		"git remote -v": {
			{Out: remoteOutputNoFork},
			{Out: remoteOutputWithFork},
		},
		"gh repo view somebody/gh-prpush --json sshUrl,url": {
			{Out: `{"sshUrl":"git@github.com:somebody/gh-prpush.git","url":"https://github.com/somebody/gh-prpush"}`},
		},
		"git remote add somebody git@github.com:somebody/gh-prpush.git": {{Out: ""}},
		"git push somebody HEAD:refs/heads/fix-thing":                   {{Out: "ok\n"}},
	})
	var out strings.Builder
	if err := Run([]string{"--add-remote"}, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "added remote somebody") {
		t.Errorf("expected the added remote to be reported: %q", got)
	}
	if !strings.Contains(got, "pushed HEAD to somebody (fix-thing)") {
		t.Errorf("expected a push to the new remote: %q", got)
	}
}

// The fork URL comes from gh, so the SSH/HTTPS choice follows the user's setup.
func TestAddRemoteUsesTheURLFromGh(t *testing.T) {
	r := run.FakeSequence(map[string][]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {{Out: prCrossRepoWithRepo}},
		"git remote -v": {
			{Out: remoteOutputNoFork},
			{Out: remoteOutputWithFork},
		},
		"gh repo view somebody/gh-prpush --json sshUrl,url": {
			{Out: `{"sshUrl":"","url":"https://github.com/somebody/gh-prpush"}`},
		},
		"git remote add somebody https://github.com/somebody/gh-prpush.git": {{Out: ""}},
		"git push somebody HEAD:refs/heads/fix-thing":                       {{Out: "ok\n"}},
	})
	var out strings.Builder
	if err := Run([]string{"--add-remote"}, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "https://github.com/somebody/gh-prpush.git") {
		t.Errorf("expected the HTTPS URL: %q", out.String())
	}
}

// The URL scheme follows the remotes already in the repository: gh's
// credential helper is https-only, so an SSH URL would fail to authenticate.
func TestAddRemoteMatchesTheExistingRemoteScheme(t *testing.T) {
	httpsRemotes := `origin	https://github.com/technothatch/gh-prpush.git (fetch)
origin	https://github.com/technothatch/gh-prpush.git (push)
`
	r := run.FakeSequence(map[string][]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {{Out: prCrossRepoWithRepo}},
		"git remote -v": {
			{Out: httpsRemotes},
			{Out: httpsRemotes + "somebody	https://github.com/somebody/gh-prpush.git (fetch)\nsomebody	https://github.com/somebody/gh-prpush.git (push)\n"},
		},
		"gh repo view somebody/gh-prpush --json sshUrl,url": {
			{Out: `{"sshUrl":"git@github.com:somebody/gh-prpush.git","url":"https://github.com/somebody/gh-prpush"}`},
		},
		"git remote add somebody https://github.com/somebody/gh-prpush.git": {{Out: ""}},
		"git push somebody HEAD:refs/heads/fix-thing":                       {{Out: "ok\n"}},
	})
	var out strings.Builder
	if err := Run([]string{"--add-remote"}, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "added remote somebody (https://github.com/somebody/gh-prpush.git)") {
		t.Errorf("expected an https URL for https remotes: %q", got)
	}
	if strings.Contains(got, "git@github.com") {
		t.Errorf("the SSH URL should not be used when the remotes are https: %q", got)
	}
}

// With SSH remotes, the SSH URL is the right one.
func TestAddRemoteUsesSSHWhenRemotesAreSSH(t *testing.T) {
	r := run.FakeSequence(map[string][]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {{Out: prCrossRepoWithRepo}},
		"git remote -v": {
			{Out: remoteOutputNoFork},
			{Out: remoteOutputWithFork},
		},
		"gh repo view somebody/gh-prpush --json sshUrl,url": {
			{Out: `{"sshUrl":"git@github.com:somebody/gh-prpush.git","url":"https://github.com/somebody/gh-prpush"}`},
		},
		"git remote add somebody git@github.com:somebody/gh-prpush.git": {{Out: ""}},
		"git push somebody HEAD:refs/heads/fix-thing":                   {{Out: "ok\n"}},
	})
	var out strings.Builder
	if err := Run([]string{"--add-remote"}, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "added remote somebody (git@github.com:somebody/gh-prpush.git)") {
		t.Errorf("expected the SSH URL for SSH remotes: %q", out.String())
	}
}

// A same-repository PR never needs a new remote.
func TestAddRemoteIsIgnoredForSameRepoPRs(t *testing.T) {
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {Out: prSameRepo},
		"git remote -v": {Out: remoteOutput},
		"git push origin HEAD:refs/heads/feature": {Out: "ok\n"},
	})
	var out strings.Builder
	if err := Run([]string{"--add-remote"}, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(out.String(), "added remote") {
		t.Errorf("no remote should be added for a same-repo PR: %q", out.String())
	}
}

// `gh pr view` cannot resolve the PR for the current branch when the fork is
// not a registered remote (cli/cli#2942). `gh pr list --head` still finds it,
// so the tool falls back to it.
func TestFallsBackToPrListWhenPrViewCannotResolve(t *testing.T) {
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {
			Err:  "no pull requests found for branch \"fix-thing\"",
			Fail: true,
		},
		"git rev-parse --abbrev-ref HEAD": {Out: "fix-thing\n"},
		"gh pr list --head fix-thing --state open --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {
			Out: "[" + prCrossRepoWithRepo + "]",
		},
		"git remote -v": {Out: remoteOutputWithFork},
		"git push somebody HEAD:refs/heads/fix-thing": {Out: "ok\n"},
	})
	var out strings.Builder
	if err := Run(nil, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "pushed HEAD to somebody (fix-thing)") {
		t.Errorf("expected the push to succeed via the fallback: %q", out.String())
	}
}

// The fallback must still work with --add-remote, adding the fork remote
// when none exists.
func TestFallbackToPrListWithAddRemote(t *testing.T) {
	r := &statefulRunner{
		responses: map[string]string{
			"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": "",
			"git rev-parse --abbrev-ref HEAD": "fix-thing\n",
			"gh pr list --head fix-thing --state open --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": "[" + prCrossRepoWithRepo + "]",
			"gh repo view somebody/gh-prpush --json sshUrl,url":                                                                       `{"sshUrl":"git@github.com:somebody/gh-prpush.git","url":"https://github.com/somebody/gh-prpush"}`,
			"git remote add somebody git@github.com:somebody/gh-prpush.git":                                                           "",
			"git push somebody HEAD:refs/heads/fix-thing":                                                                             "ok\n",
		},
		fail: map[string]bool{
			"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": true,
		},
		remotesBefore: remoteOutputNoFork,
		remotesAfter:  remoteOutputWithFork,
	}
	var out strings.Builder
	if err := Run([]string{"--add-remote"}, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "added remote somebody") {
		t.Errorf("expected the remote to be added: %q", got)
	}
	if !strings.Contains(got, "pushed HEAD to somebody (fix-thing)") {
		t.Errorf("expected a push to the new remote: %q", got)
	}
}

// statefulRunner answers "git remote -v" differently before and after a
// "git remote add", which is how a real repository behaves.
type statefulRunner struct {
	responses     map[string]string
	fail          map[string]bool
	remotesBefore string
	remotesAfter  string
	added         bool
}

func (s *statefulRunner) Run(name string, args ...string) (string, string, error) {
	key := name + " " + strings.Join(args, " ")
	if key == "git remote -v" {
		if s.added {
			return s.remotesAfter, "", nil
		}
		return s.remotesBefore, "", nil
	}
	if strings.HasPrefix(key, "git remote add ") {
		s.added = true
	}
	out, ok := s.responses[key]
	if !ok {
		return "", "", fmt.Errorf("statefulRunner: unexpected command %q", key)
	}
	if s.fail[key] {
		return out, "no pull requests found for branch", fmt.Errorf("command failed: %s", key)
	}
	return out, "", nil
}

// When several forks use the same branch name, the one whose owner matches a
// registered remote wins.
func TestFallbackPrefersTheRemoteThatMatches(t *testing.T) {
	twoPRs := `[{"number":1,"headRefName":"fix-thing","isCrossRepository":true,` +
		`"headRepositoryOwner":{"login":"someoneelse"},"headRepository":{"nameWithOwner":"someoneelse/gh-prpush"}},` +
		`{"number":2,"headRefName":"fix-thing","isCrossRepository":true,` +
		`"headRepositoryOwner":{"login":"somebody"},"headRepository":{"nameWithOwner":"somebody/gh-prpush"}}]`
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {
			Err:  "no pull requests found for branch \"fix-thing\"",
			Fail: true,
		},
		"git rev-parse --abbrev-ref HEAD": {Out: "fix-thing\n"},
		"gh pr list --head fix-thing --state open --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {
			Out: twoPRs,
		},
		"git remote -v": {Out: remoteOutputWithFork},
		"git push somebody HEAD:refs/heads/fix-thing": {Out: "ok\n"},
	})
	var out strings.Builder
	if err := Run(nil, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "pushed HEAD to somebody") {
		t.Errorf("expected the matching fork to be used: %q", out.String())
	}
}

// Ambiguity is reported rather than resolved by guesswork.
func TestFallbackReportsAmbiguity(t *testing.T) {
	twoPRs := `[{"number":1,"headRefName":"fix-thing","isCrossRepository":true,` +
		`"headRepositoryOwner":{"login":"someoneelse"},"headRepository":{"nameWithOwner":"someoneelse/gh-prpush"}},` +
		`{"number":2,"headRefName":"fix-thing","isCrossRepository":true,` +
		`"headRepositoryOwner":{"login":"somebody"},"headRepository":{"nameWithOwner":"somebody/gh-prpush"}}]`
	r := fake(map[string]struct {
		Out  string
		Err  string
		Fail bool
	}{
		"gh pr view --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {
			Err:  "no pull requests found for branch \"fix-thing\"",
			Fail: true,
		},
		"git rev-parse --abbrev-ref HEAD": {Out: "fix-thing\n"},
		"gh pr list --head fix-thing --state open --json number,headRefName,isCrossRepository,headRepositoryOwner,headRepository": {
			Out: twoPRs,
		},
		"git remote -v": {Out: remoteOutputNoFork},
	})
	var out strings.Builder
	err := Run(nil, &out, &out, r)
	if err == nil {
		t.Fatal("expected an error for an ambiguous branch")
	}
	if !strings.Contains(err.Error(), "several pull requests") {
		t.Errorf("unexpected error: %v", err)
	}
}
