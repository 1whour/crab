package executer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gnh123/scheduler/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRun(t *testing.T) {

	router := gin.New()
	router.POST("/", func(c *gin.Context) {
		c.JSON(200, "{}")
	})

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

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

	u, err := url.Parse(ts.URL)
	assert.NoError(t, err)

	hostPort := u.Host
	pos := strings.Index(hostPort, ":")
	s = strings.Replace(s, "127.0.0.1", u.Host[:pos], -1)
	s = strings.Replace(s, "8080", u.Host[pos+1:], -1)

	var param model.ExecutorParam

	err = yaml.Unmarshal([]byte(s), &param)

	assert.NoError(t, err)
	if err != nil {
		return
	}

	err = createHTTPExecuter(context.TODO(), &param).Run()
	assert.NoError(t, err)
}
