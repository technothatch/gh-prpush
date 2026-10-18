package app

import (
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
		"gh pr view --json headRefName,isCrossRepository,headRepositoryOwner": {Out: prSameRepo},
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
		"gh pr view --json headRefName,isCrossRepository,headRepositoryOwner": {Out: prCrossRepo},
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
		"gh pr view --json headRefName,isCrossRepository,headRepositoryOwner": {Out: prCrossRepo},
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
		"gh pr view --json headRefName,isCrossRepository,headRepositoryOwner": {Err: "no pull requests found for branch \"main\"", Fail: true},
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
		"gh pr view --json headRefName,isCrossRepository,headRepositoryOwner": {Out: prSameRepo},
		"git remote -v": {Out: remoteOutput},
		"git push origin HEAD:refs/heads/feature --force-with-lease": {Out: "ok\n"},
	})
	var out strings.Builder
	if err := Run([]string{"--force-with-lease"}, &out, &out, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
