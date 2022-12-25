package status

import (
	"fmt"
	"os"

	"github.com/1whour/crab/model"
	"github.com/guonaihong/gout"
)

type Status struct {
	GateAddr []string `clop:"short;long" usage:"gate address" valid:"required"`
	UserName string   `clop:"short;long" usage:"username" valid:"required"`

	Password string `clop:"short;long" usage:"password" valid:"required"`

	Debug bool `clop:"short;long" usage:"debug"`
}

func (s *Status) SubMain() {
	u := fmt.Sprintf("%s%s", s.GateAddr[0], model.TASK_UI_STATUS_URL)

	err := gout.
		GET(u).
		Debug(s.Debug).
		SetQuery(gout.H{"format": "table"}).
		BindBody(os.Stdout).Do()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
