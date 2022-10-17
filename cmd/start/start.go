package start

import (
	"net/http"

	"github.com/gnh123/scheduler/cmd/clicrud"
	"github.com/gnh123/scheduler/model"
)

type Start struct {
	clicrud.CrudOpt
}

// start子命令入口函数
func (r *Start) SubMain() {

	clicrud.Crud(r.FileName, r.GateAddr[0]+model.TASK_CREATE_URL, http.MethodPost)
}
