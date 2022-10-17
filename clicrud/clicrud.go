package clicrud

import (
	"fmt"
	"os"
	"strings"

	"github.com/guonaihong/gout"
)

// fileName是需要打开的文件名
// url是gate服务的地址
func Crud(fileName string, url string, method string) error {
	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	code := 0
	s := ""
	err = gout.New().SetMethod(strings.ToUpper(method)).SetBody(fd).Code(&code).BindBody(&s).Do()
	if err != nil {
		return err
	}

	if code != 200 {
		return fmt.Errorf("cli crud: http.StatusCode(%d) != 200, rsp(%s)", code, s)
	}
	return nil
}
