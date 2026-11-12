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
	return &fakeRunner{responses: responses}
}

// FakeSequence returns a runner that answers the same command with a
// different response each time it is called. Responses beyond the end of the
// slice repeat the last one.
func FakeSequence(responses map[string][]struct {
	Out  string
	Err  string
	Fail bool
}) Runner {
	return &fakeRunner{sequences: responses, calls: map[string]int{}}
}

type fakeRunner struct {
	responses map[string]struct {
		Out  string
		Err  string
		Fail bool
	}
	sequences map[string][]struct {
		Out  string
		Err  string
		Fail bool
	}
	calls map[string]int
}

func (f *fakeRunner) Run(name string, args ...string) (string, string, error) {
	key := name + " " + join(args)

	if f.sequences != nil {
		seq, ok := f.sequences[key]
		if !ok {
			return "", "", fmt.Errorf("fake runner: unexpected command %q", key)
		}
		idx := f.calls[key]
		f.calls[key]++
		if idx >= len(seq) {
			idx = len(seq) - 1
		}
		r := seq[idx]
		if r.Fail {
			return r.Out, r.Err, fmt.Errorf("command failed: %s", key)
		}
		return r.Out, r.Err, nil
	}

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
