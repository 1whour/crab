package executer

import (
	"context"
	"testing"

	"github.com/gnh123/scheduler/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRun(t *testing.T) {

	var s = `
http:
    method: post
    scheme: http
    host: 127.0.0.1
    port: 8080
    headers:
    - name: Bid
      value: xxxx
    - name: token
      value: vvvv
    body: |
      {"a":"b"}
`

	var param model.ExecutorParam
	err := yaml.Unmarshal([]byte(s), &param)
	assert.NoError(t, err)

	err = createHTTPExecuter(context.TODO(), &param).Run()
	assert.NoError(t, err)
}
