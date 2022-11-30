package executer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gnh123/ktuo/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestShellExecuter(t *testing.T) {

	ts := mockserver()
	defer ts.Close()

	conf := `
apiVersion: v0.0.1
kind: oneRuntime
trigger:
  cron: "* * * * * *" #每秒触发一次
executer:
  shell:
    command: curl -v -X POST -H "ktuo-http-executer:D79DCF87-595A-4B18-B76F-9BD8277C35EF" -d "test hello" 127.0.0.1:8181/task #command和args的作用是等价的，唯一的区别是命令放在一个字符串或者slice里面。
  `
	var param model.Param
	var err error
	err = yaml.Unmarshal([]byte(conf), &param)
	assert.NoError(t, err)

	param.Executer.Shell.Command = strings.Replace(param.Executer.Shell.Command, "127.0.0.1:8181", ts.URL, -1)
	fmt.Printf("%v, %s\n", ts.URL, param.Executer.Shell.Command)
	payload, err := createShellExecuter(context.TODO(), &param).Run()
	assert.NoError(t, err)
	assert.Equal(t, len(payload), 0)
}
