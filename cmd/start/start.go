package start

import (
	"fmt"
	"net/http"

	"github.com/gnh123/scheduler/cmd/clicrud"
	"github.com/gnh123/scheduler/model"
)

type Start struct {
	clicrud.CrudOpt
}

// start子命令入口函数
func (r *Start) SubMain() {

	if len(r.GateAddr) == 0 {
		return
	}

	err := r.Crud(r.GateAddr[0]+model.TASK_CREATE_URL, http.MethodPost)
	if err != nil {
		fmt.Printf("start: %s\n", err)
	}
}
