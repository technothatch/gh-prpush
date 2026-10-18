package run

import (
	"bytes"
	"fmt"
	"os/exec"
)

type Runner interface {
	Run(name string, args ...string) (stdout string, stderr string, err error)
}

type ExecRunner struct{}

func (ExecRunner) Run(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func Fake(responses map[string]struct {
	Out  string
	Err  string
	Fail bool
}) Runner {
	return fakeRunner{responses}
}

type fakeRunner struct {
	responses map[string]struct {
		Out  string
		Err  string
		Fail bool
	}
}

func (f fakeRunner) Run(name string, args ...string) (string, string, error) {
	key := name + " " + join(args)
	r, ok := f.responses[key]
	if !ok {
		return "", "", fmt.Errorf("fake runner: unexpected command %q", key)
	}
	if r.Fail {
		return r.Out, r.Err, fmt.Errorf("command failed: %s", key)
	}
	return r.Out, r.Err, nil
}

func join(args []string) string {
	s := ""
	for i, a := range args {
		if i > 0 {
			s += " "
		}
		s += a
	}
	return s
}
