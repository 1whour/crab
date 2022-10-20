package stop

import (
	"net/http"

	"github.com/gnh123/scheduler/cmd/clicrud"
	"github.com/gnh123/scheduler/model"
)

type Stop struct {
	clicrud.CrudOpt
}

// stop子命令入口
func (s *Stop) SubMain() {
	s.Crud(s.GateAddr[0]+model.TASK_STOP_URL, http.MethodPost)
}
