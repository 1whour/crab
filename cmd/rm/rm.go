package rm

import (
	"net/http"

	"github.com/gnh123/scheduler/cmd/clicrud"
	"github.com/gnh123/scheduler/model"
)

type Rm struct {
	clicrud.CrudOpt
}

// 删除子命令入口
func (r *Rm) SubMain() {
	r.Crud(r.GateAddr[0]+model.TASK_DELETE_URL, http.MethodDelete)
}
