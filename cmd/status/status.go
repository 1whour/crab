package status

import (
	"fmt"
	"os"

	"github.com/gnh123/scheduler/model"
	"github.com/guonaihong/gout"
)

type Status struct {
	GateAddr []string `clop:"short;long" usage:"gate address" valid:"required"`
	Filter   []string `clop:"short;long;greedy" usage:"filter"`
	Debug    bool     `clop:"short;long" usage:"debug"`
}

func (s *Status) SubMain() {
	u := fmt.Sprintf("%s%s", s.GateAddr[0], model.TASK_STATUS_URL)

	err := gout.
		GET(u).
		Debug(s.Debug).
		SetJSON(gout.H{"format": "table",
			"filter": s.Filter}).
		BindBody(os.Stdout).Do()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
