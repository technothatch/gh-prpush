package app

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/technothatch/gh-prpush/internal/gitx"
	"github.com/technothatch/gh-prpush/internal/run"
)

type PR struct {
	HeadRefName         string `json:"headRefName"`
	IsCrossRepository   bool   `json:"isCrossRepository"`
	HeadRepositoryOwner struct {
		Login string `json:"login"`
	} `json:"headRepositoryOwner"`
}

func viewPR(r run.Runner) (PR, error) {
	out, errb, err := r.Run("gh", "pr", "view", "--json", "headRefName,isCrossRepository,headRepositoryOwner")
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
	rest := args
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force-with-lease":
			force = true
		case "-h", "--help":
			fmt.Fprint(stdout, `gh pr push - push the head branch of the current pull request

Usage:
  gh pr push [--force-with-lease]

Pushes the checked-out branch of the PR you are on to the correct remote
(your fork when the PR is cross-repository, origin otherwise).

Implements https://github.com/cli/cli/issues/2189 as an extension.
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
		return err
	}

	remotes, err := gitx.ListRemotes(runner)
	if err != nil {
		return err
	}

	remote, err := gitx.PickRemote(remotes, pr.HeadRepositoryOwner.Login, pr.IsCrossRepository)
	if err != nil {
		return err
	}

	if err := gitx.Push(runner, remote, pr.HeadRefName, force); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "pushed HEAD to %s (%s)\n", remote.Name, pr.HeadRefName)
	return nil
}
