package clicrud

import (
	"fmt"
	"os"
	"strings"

	"github.com/guonaihong/gout"
)

type CrudOpt struct {
	FileName string   `clop:"short;long" usage:"config filename" valid:"required"`
	GateAddr []string `clop:"short;long" usage:"gate address" valid:"required"`
}

// fileName是需要打开的文件名
// url是gate服务的地址
func Crud(fileName string, url string, method string) error {
	all, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	code := 0
	s := ""
	req := gout.New().SetMethod(strings.ToUpper(method))
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
