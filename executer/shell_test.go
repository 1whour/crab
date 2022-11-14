package executer

import (
	"context"
	"testing"

	"github.com/gnh123/scheduler/model"
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
    command: curl -v -X POST -H "scheduler-http-executer:D79DCF87-595A-4B18-B76F-9BD8277C35EF" -d "test hello" 127.0.0.1:8181/task #command和args的作用是等价的，唯一的区别是命令放在一个字符串或者slice里面。
  `
	var param model.Param
	var err error
	conf, err = modifyConfig(ts, conf)
	assert.NoError(t, err)

	err = yaml.Unmarshal([]byte(conf), &param)
	assert.NoError(t, err)

	err = createShellExecuter(context.TODO(), &param).Run()
	assert.NoError(t, err)
}
