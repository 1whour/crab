package executer

import (
	"context"
	"fmt"
	"testing"

	"github.com/gnh123/ktuo/model"
	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v3"
	//yaml "github.com/goccy/go-yaml"
)

// 使用如下结构体触发http请求
func TestHTTPExecuterRun(t *testing.T) {

	ts := mockserver()
	defer ts.Close()

	var s = `
apiVersion: v0.0.1 #api版本号
kind: oneRuntime #只在一个runtime上运行
trigger:
  cron: "* * * * * *"
executer:
  taskName: first-task
  http:
    method: post
    scheme: http
    host: 127.0.0.1
    port: 8080
    path: "/task"
    headers:
    - name: Bid
      value: xxxx
    - name: token
      value: vvvv
    body: |
      {"a":"b"}
`

	var err error
	var param model.Param

	s, err = modifyConfig(ts, s)
	assert.NoError(t, err)

	err = yaml.Unmarshal([]byte(s), &param)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	fmt.Printf("%s\n", ts.URL)
	payload, err := createHTTPExecuter(context.TODO(), &param).Run()
	assert.NoError(t, err)
	assert.Equal(t, len(payload), 0)
}
