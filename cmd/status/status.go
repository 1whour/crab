package status

import (
	"fmt"
	"os"

	"github.com/gnh123/scheduler/model"
	"github.com/guonaihong/gout"
)

type Status struct {
	GateAddr []string `clop:"short;long" usage:"gate address" valid:"required"`
}

func (s *Status) SubMain() {
	u := fmt.Sprintf("%s%s%s", s.GateAddr[0], model.TASK_STATUS_CLIENT_URL, "table")
	err := gout.GET(u).Debug(false).BindBody(os.Stdout).Do()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
