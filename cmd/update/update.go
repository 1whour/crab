package update

import (
	"net/http"

	"github.com/gnh123/scheduler/cmd/clicrud"
	"github.com/gnh123/scheduler/model"
)

type Update struct {
	clicrud.CrudOpt
}

// 更新子命令入口
func (u *Update) SubMain() {
	u.Crud(u.GateAddr[0]+model.TASK_UPDATE_URL, http.MethodPut)
}
