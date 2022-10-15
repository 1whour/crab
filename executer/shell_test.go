package executer

import (
	"context"
	"testing"

	"github.com/gnh123/scheduler/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestShellExecuter(t *testing.T) {

	conf := `
  shell:
    commond: echo "hello"
    args:
    - echo
    - "hello"
  `
	var param model.ExecutorParam
	err := yaml.Unmarshal([]byte(conf), &param)
	assert.NoError(t, err)

	err = createShellExecuter(context.TODO(), &param).Run()
	assert.NoError(t, err)
}
