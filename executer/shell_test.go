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
	var param model.Param
	err := yaml.Unmarshal([]byte(conf), &param.Executer)
	assert.NoError(t, err)

	err = createShellExecuter(context.TODO(), &param).Run()
	assert.NoError(t, err)
}
