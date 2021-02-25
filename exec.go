package lxdops

import (
	"bytes"
	"io"
	"os"
	"strings"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

// NopCloser returns a ReadCloser with a no-op Close method wrapping
// the provided Reader r.
func NopCloser(r io.Writer) io.WriteCloser {
	return nopCloser{r}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

type execRunner struct {
	Server    lxd.InstanceServer
	Container string
	Error     error
	dir       string
	uid       uint32
	gid       uint32
}

func (s *execRunner) Dir(dir string) *execRunner {
	s.dir = dir
	return s
}

func (s *execRunner) Uid(uid uint32) *execRunner {
	s.uid = uid
	return s
}

func (s *execRunner) Gid(gid uint32) *execRunner {
	s.gid = gid
	return s
}

func (s *execRunner) run(content string, captureOutput bool, execArgs []string) ([]byte, error) {
	if s.Error != nil {
		return nil, s.Error
	}
	var post api.InstanceExecPost
	post.Command = execArgs
	post.WaitForWS = true
	post.Cwd = s.dir
	post.User = s.uid
	post.Group = s.gid

	var buf bytes.Buffer
	var args lxd.InstanceExecArgs
	args.Stderr = os.Stderr
	if captureOutput {
		args.Stdout = NopCloser(&buf)
	} else {
		args.Stdout = os.Stdout
	}

	if content != "" {
		args.Stdin = io.NopCloser(strings.NewReader(content))
	}
	op, err := s.Server.ExecInstance(s.Container, post, &args)
	if err != nil {
		return nil, AnnotateLXDError(s.Container, err)
	}
	err = op.Wait()
	if err != nil {
		return nil, AnnotateLXDError(s.Container, err)
	}
	return buf.Bytes(), nil
}

func (s *execRunner) Run(content string, execArgs ...string) error {
	_, err := s.run(content, false, execArgs)
	return err
}

func (s *execRunner) Output(content string, execArgs ...string) ([]byte, error) {
	return s.run(content, true, execArgs)
}
