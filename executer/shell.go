package executer

import (
	"context"
	"os/exec"

	"github.com/gnh123/scheduler/model"
)

func init() {
	Register("shell", createShellExecuter)
}

type shellExecuter struct {
	cmd *exec.Cmd
}

func (s *shellExecuter) Cancel() error {
	return s.cmd.Process.Kill()
}

func (s *shellExecuter) Run() error {
	return s.cmd.Run()
}

func createShellExecuter(ctx context.Context, param *model.Param) Executer {
	s := &shellExecuter{}

	shellParam := param.Shell

	if len(shellParam.Command) > 0 {
		s.cmd = exec.CommandContext(ctx, "bash", "-c", shellParam.Command)
	} else {
		s.cmd = exec.CommandContext(ctx, shellParam.Args[0], shellParam.Args[1:]...)
	}
	return s
}
