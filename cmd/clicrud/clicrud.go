package clicrud

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gnh123/scheduler/model"
	"github.com/guonaihong/gout"
)

type CrudOpt struct {
	FileName string   `clop:"short;long" usage:"config filename" valid:"required"`
	GateAddr []string `clop:"short;long" usage:"gate address" valid:"required"`
	Debug    bool     `clop:"short;long" usage:"debug mode"`
}

type Rm struct {
	CrudOpt
}

// 删除子命令入口
func (r *Rm) SubMain() {
	r.Crud(r.GateAddr[0]+model.TASK_DELETE_URL, http.MethodDelete)
}

type Start struct {
	CrudOpt
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

type Stop struct {
	CrudOpt
}

// stop子命令入口
func (s *Stop) SubMain() {
	s.Crud(s.GateAddr[0]+model.TASK_STOP_URL, http.MethodPost)
}

type Update struct {
	CrudOpt
}

// 更新子命令入口
func (u *Update) SubMain() {
	u.Crud(u.GateAddr[0]+model.TASK_UPDATE_URL, http.MethodPut)
}

// fileName是需要打开的文件名
// url是gate服务的地址
func (c *CrudOpt) Crud(url string, method string) error {
	fileName := c.FileName
	all, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	code := 0
	s := ""
	req := gout.New().SetMethod(strings.ToUpper(method)).SetURL(url).Debug(c.Debug)
	if strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml") {
		req.SetYAML(all)
	} else if strings.HasSuffix(fileName, ".json") {
		req.SetJSON(all)
	} else {
		req.SetBody(all)
	}
	err = req.Code(&code).BindBody(&s).Do()
	if err != nil {
		return err
	}

	if code != 200 {
		return fmt.Errorf("cli crud: http.StatusCode(%d) != 200, rsp(%s)", code, s)
	}
	return nil
}
