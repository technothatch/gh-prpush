package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/technothatch/gh-prpush/internal/gitx"
	"github.com/technothatch/gh-prpush/internal/run"
)

type PR struct {
	Number              int    `json:"number"`
	HeadRefName         string `json:"headRefName"`
	IsCrossRepository   bool   `json:"isCrossRepository"`
	HeadRepositoryOwner struct {
		Login string `json:"login"`
	} `json:"headRepositoryOwner"`
	HeadRepository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"headRepository"`
}

// prFields are the pull request fields the tool asks gh for.
//
// `gh pr view` has no baseRepository field, but for a same-repository pull
// request the head repository is the base repository, which is all that is
// needed to recognise the repository's own remote.
const prFields = "number,headRefName,isCrossRepository,headRepositoryOwner,headRepository"

func viewPR(r run.Runner) (PR, error) {
	out, errb, err := r.Run("gh", "pr", "view", "--json", prFields)
	if err != nil {
		return PR{}, fmt.Errorf("no pull request found for this branch; create one with 'gh pr create' (%s)", firstLine(errb, out))
	}
	var pr PR
	if err := json.Unmarshal([]byte(out), &pr); err != nil {
		return PR{}, fmt.Errorf("parse gh pr view output: %w", err)
	}
	if pr.HeadRefName == "" {
		return PR{}, fmt.Errorf("gh pr view returned an empty headRefName")
	}
	return pr, nil
}

// currentBranch returns the checked-out branch name.
func currentBranch(r run.Runner) (string, error) {
	out, errb, err := r.Run("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse --abbrev-ref HEAD: %v %s", err, strings.TrimSpace(errb))
	}
	branch := strings.TrimSpace(out)
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("cannot determine the current branch (detached HEAD?)")
	}
	return branch, nil
}

// listPRsByBranch finds pull requests whose head branch matches, across
// forks.
//
// `gh pr view` cannot do this on its own when the fork is not registered as
// a remote: gh resolves the PR for the current branch through git's push
// target configuration and builds a qualified head ref (`owner:branch`). With
// no `branch.<name>.remote` set, it looks for the branch in the base repo
// only and finds nothing - even though `gh pr list --head` finds it. That is
// the state described in cli/cli#2942, so we fall back to it.
func listPRsByBranch(r run.Runner, branch string) ([]PR, error) {
	out, errb, err := r.Run("gh", "pr", "list", "--head", branch, "--state", "open",
		"--json", prFields)
	if err != nil {
		return nil, fmt.Errorf("gh pr list --head %s: %v %s", branch, err, strings.TrimSpace(errb))
	}
	var prs []PR
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, fmt.Errorf("parse gh pr list output: %w", err)
	}
	return prs, nil
}

// pickPR chooses between several pull requests that share a head branch.
// A candidate whose fork owner matches one of the remotes wins; otherwise a
// single candidate is used as-is.
func pickPR(prs []PR, remotes []gitx.Remote) (PR, error) {
	switch len(prs) {
	case 0:
		return PR{}, fmt.Errorf("no pull request found for this branch")
	case 1:
		return prs[0], nil
	}
	var matches []PR
	for _, pr := range prs {
		owner := pr.HeadRepositoryOwner.Login
		for _, rm := range remotes {
			if o, _, ok := gitx.GitHubOwnerRepo(rm.URL); ok && strings.EqualFold(o, owner) {
				matches = append(matches, pr)
				break
			}
			if strings.EqualFold(rm.Name, owner) {
				matches = append(matches, pr)
				break
			}
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	owners := make([]string, 0, len(prs))
	for _, pr := range prs {
		owners = append(owners, pr.HeadRepositoryOwner.Login)
	}
	return PR{}, fmt.Errorf(
		"several pull requests use the head branch %q (%s); pass the PR number to disambiguate",
		prs[0].HeadRefName, strings.Join(owners, ", "))
}

// forkRemoteURL asks gh for the clone URL of the fork that owns the PR head.
// Asking gh avoids guessing between SSH and HTTPS.
//
// The scheme follows the remotes already in the repository: a user whose
// remotes are HTTPS has their credentials configured for HTTPS (gh's
// credential helper is https-only by default), so an SSH URL would fail with
// "Permission denied (publickey)".
func forkRemoteURL(r run.Runner, nameWithOwner string, remotes []gitx.Remote) (string, error) {
	if nameWithOwner == "" {
		return "", fmt.Errorf("the pull request has no head repository")
	}
	out, errb, err := r.Run("gh", "repo", "view", nameWithOwner, "--json", "sshUrl,url")
	if err != nil {
		return "", fmt.Errorf("gh repo view %s: %v %s", nameWithOwner, err, strings.TrimSpace(errb))
	}
	var repo struct {
		SSHUrl string `json:"sshUrl"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal([]byte(out), &repo); err != nil {
		return "", fmt.Errorf("parse gh repo view output: %w", err)
	}
	httpsURL := ""
	if repo.URL != "" {
		httpsURL = repo.URL + ".git"
	}
	if usesHTTPS(remotes) {
		if httpsURL != "" {
			return httpsURL, nil
		}
		return "", fmt.Errorf("gh repo view %s returned no https URL", nameWithOwner)
	}
	if repo.SSHUrl != "" {
		return repo.SSHUrl, nil
	}
	if httpsURL != "" {
		return httpsURL, nil
	}
	return "", fmt.Errorf("gh repo view %s returned no clone URL", nameWithOwner)
}

// usesHTTPS reports whether the repository's remotes use HTTPS URLs.
func usesHTTPS(remotes []gitx.Remote) bool {
	for _, rm := range remotes {
		if strings.HasPrefix(rm.URL, "https://") {
			return true
		}
	}
	return false
}

func firstLine(s ...string) string {
	for _, str := range s {
		for _, line := range splitLines(str) {
			if line != "" {
				return line
			}
		}
	}
	return ""
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, trimCR(s[start:i]))
			start = i + 1
		}
	}
	out = append(out, trimCR(s[start:]))
	return out
}

func trimCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}

func Run(args []string, stdout, stderr io.Writer, r ...run.Runner) error {
	var runner run.Runner
	if len(r) > 0 && r[0] != nil {
		runner = r[0]
	} else {
		runner = run.ExecRunner{}
	}

	force := false
	addRemote := false
	rest := args
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force-with-lease":
			force = true
		case "--add-remote":
			addRemote = true
		case "-h", "--help":
			fmt.Fprint(stdout, `gh prpush - push the head branch of the current pull request

Usage:
  gh prpush [--force-with-lease] [--add-remote]

Pushes the checked-out branch of the PR you are on to the correct remote
(your fork when the PR is cross-repository, origin otherwise).

Flags:
  --force-with-lease  pass --force-with-lease to git push
  --add-remote        if the fork has no remote yet, add one named after the
                      fork owner and push to it

Note: this is a gh extension, so it runs as 'gh prpush'. The upstream
feature request is https://github.com/cli/cli/issues/2189.
`)
			return nil
		default:
			rest = args[i:]
			i = len(args)
		}
	}
	_ = rest

	pr, err := viewPR(runner)
	if err != nil {
		// `gh pr view` resolves the PR for the current branch through git's
		// push-target configuration, which it cannot do when the fork is not
		// registered as a remote (cli/cli#2942). `gh pr list --head` finds the
		// same PR, so fall back to it.
		branch, branchErr := currentBranch(runner)
		if branchErr != nil {
			return err
		}
		prs, listErr := listPRsByBranch(runner, branch)
		if listErr != nil {
			return err
		}
		remotes, remotesErr := gitx.ListRemotes(runner)
		if remotesErr != nil {
			return remotesErr
		}
		pr, err = pickPR(prs, remotes)
		if err != nil {
			return err
		}
	}

	remotes, err := gitx.ListRemotes(runner)
	if err != nil {
		return err
	}

	// For a same-repository PR the head repository is the repository the PR
	// was opened against, so its remote is the one to push to.
	remote, err := gitx.PickRemote(remotes, pr.HeadRepository.NameWithOwner, pr.HeadRepositoryOwner.Login, pr.IsCrossRepository)
	if err != nil {
		// The fork has no remote yet - a common state after `gh pr checkout`
		// on a machine where the fork was never cloned. Offer to add one
		// instead of failing with instructions the user has to copy.
		if !addRemote || !pr.IsCrossRepository {
			return err
		}
		owner := pr.HeadRepositoryOwner.Login
		if owner == "" {
			return err
		}
		url, urlErr := forkRemoteURL(runner, pr.HeadRepository.NameWithOwner, remotes)
		if urlErr != nil {
			return fmt.Errorf("%v; could not determine the fork URL: %w", err, urlErr)
		}
		if err := gitx.AddRemote(runner, owner, url); err != nil {
			return fmt.Errorf("%v; could not add the fork remote: %w", err, err)
		}
		fmt.Fprintf(stdout, "added remote %s (%s)\n", owner, url)
		remotes, err = gitx.ListRemotes(runner)
		if err != nil {
			return err
		}
		remote, err = gitx.PickRemote(remotes, "", owner, true)
		if err != nil {
			return err
		}
	}

	if err := gitx.Push(runner, remote, pr.HeadRefName, force); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "pushed HEAD to %s (%s)\n", remote.Name, pr.HeadRefName)
	return nil
}
