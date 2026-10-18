package gitx

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/technothatch/gh-prpush/internal/run"
)

type Remote struct {
	Name string
	URL  string
}

var remoteLine = regexp.MustCompile(`^(\S+)\s+(\S+)\s+\(.*\)$`)

func ListRemotes(r run.Runner) ([]Remote, error) {
	out, errb, err := r.Run("git", "remote", "-v")
	if err != nil {
		return nil, fmt.Errorf("git remote -v: %v %s", err, strings.TrimSpace(errb))
	}
	seen := map[string]Remote{}
	for _, line := range strings.Split(out, "\n") {
		m := remoteLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		seen[m[1]] = Remote{Name: m[1], URL: m[2]}
	}
	remotes := make([]Remote, 0, len(seen))
	for _, rm := range seen {
		remotes = append(remotes, rm)
	}
	sort.Slice(remotes, func(i, j int) bool { return remotes[i].Name < remotes[j].Name })
	return remotes, nil
}

func githubOwnerRepo(url string) (owner, repo string, ok bool) {
	u := strings.TrimSuffix(strings.TrimSuffix(url, "/"), ".git")
	u = strings.TrimPrefix(u, "git@github.com:")
	u = strings.TrimPrefix(u, "ssh://git@github.com/")
	u = strings.TrimPrefix(u, "https://github.com/")
	u = strings.TrimPrefix(u, "http://github.com/")
	u = strings.TrimPrefix(u, "github.com/")
	parts := strings.Split(u, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func PickRemote(remotes []Remote, owner string, crossRepo bool) (Remote, error) {
	if !crossRepo {
		for _, rm := range remotes {
			if rm.Name == "origin" {
				return rm, nil
			}
		}
		return Remote{}, fmt.Errorf("no 'origin' remote found (remotes: %s)", names(remotes))
	}

	for _, rm := range remotes {
		if o, _, ok := githubOwnerRepo(rm.URL); ok && strings.EqualFold(o, owner) {
			return rm, nil
		}
	}
	for _, rm := range remotes {
		if strings.EqualFold(rm.Name, owner) {
			return rm, nil
		}
	}
	return Remote{}, fmt.Errorf("no remote found for fork owner %q (remotes: %s); add it with: git remote add %s https://github.com/%s/<repo>.git", owner, names(remotes), owner, owner)
}

func names(remotes []Remote) string {
	if len(remotes) == 0 {
		return "none"
	}
	parts := make([]string, len(remotes))
	for i, rm := range remotes {
		parts[i] = rm.Name
	}
	return strings.Join(parts, ", ")
}

func Push(r run.Runner, remote Remote, branch string, forceWithLease bool) error {
	args := []string{"push", remote.Name, "HEAD:refs/heads/" + branch}
	if forceWithLease {
		args = append(args, "--force-with-lease")
	}
	_, errb, err := r.Run("git", args...)
	if err != nil {
		return fmt.Errorf("git push: %v %s", err, strings.TrimSpace(errb))
	}
	return nil
}
