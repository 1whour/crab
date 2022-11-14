package executer

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func mockserver() *httptest.Server {

	router := gin.New()
	router.POST("/task", func(c *gin.Context) {
		c.JSON(200, "{}")
	})

	return httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
}

func modifyConfig(ts *httptest.Server, s string) (string, error) {

	u, err := url.Parse(ts.URL)
	if err != nil {
		return "", err
	}

	hostPort := u.Host
	pos := strings.Index(hostPort, ":")
	s = strings.Replace(s, "127.0.0.1", u.Host[:pos], -1)
	s = strings.Replace(s, "8080", u.Host[pos+1:], -1)
	return s, nil
}
