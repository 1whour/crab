package executer

import (
	"context"
	"os/exec"

	"github.com/1whour/crab/model"
)

func init() {
	Register("shell", createShellExecuter)
}

type shellExecuter struct {
	cmd *exec.Cmd
}

func (s *shellExecuter) Stop() error {
	return s.cmd.Process.Kill()
}

func (s *shellExecuter) Run() ([]byte, error) {
	return s.cmd.Output()
}

func createShellExecuter(ctx context.Context, param *model.Param) Executer {
	if param.Executer.Shell == nil {
		return nil
	}

	s := &shellExecuter{}

	shellParam := param.Executer.Shell

	if len(shellParam.Command) > 0 {
		s.cmd = exec.CommandContext(ctx, "bash", "-c", shellParam.Command)
	} else {
		s.cmd = exec.CommandContext(ctx, shellParam.Args[0], shellParam.Args[1:]...)
	}
	return s
}
