package executer

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/gnh123/scheduler/model"
	"github.com/guonaihong/gout"
	"github.com/guonaihong/gout/dataflow"
	"github.com/guonaihong/gstl/ifop"
)

func init() {
	Register("http", createHTTPExecuter)
}

var _ Executer = (*httpExecuter)(nil)

type httpExecuter struct {
	req    *dataflow.Gout     //http client
	ctx    context.Context    //新生成的ctx
	cancel context.CancelFunc //取消用的cancel
	param  *model.ExecuterParam
}

// 运行
func (h *httpExecuter) Run() error {
	httpData := h.param.HTTP

	h.req.SetMethod(strings.ToUpper(httpData.Method))

	// 查询字符串
	if len(httpData.Querys) > 0 {
		q := make([]string, 0, len(httpData.Querys))
		querys := httpData.Querys
		for i := 0; i < len(querys); i++ {
			q = append(q, querys[i].Name, querys[i].Value)
		}
		h.req.SetQuery(q)
	}

	// http header
	if len(httpData.Headers) > 0 {
		h2 := make([]string, 0, len(httpData.Headers))
		headers := httpData.Headers
		for i := 0; i < len(headers); i++ {
			h2 = append(h2, headers[i].Name, headers[i].Value)
		}
		h.req.SetHeader(h2)
	}

	var u url.URL
	u.Scheme = httpData.Scheme
	u.Host = ifop.IfElse(httpData.Port != 0, fmt.Sprintf("%s:%d", u.Host, httpData.Port), httpData.Host)
	u.Path = httpData.Path

	h.req.SetURL(u.String())     //设置url
	h.req.SetBody(httpData.Body) //设置body
	h.req.WithContext(h.ctx)     //设置context
	code := 0
	err := h.req.Code(&code).Debug(true).Do()

	return ifop.IfElse(err != nil,
		err,
		ifop.IfElse(code != 200,
			errors.New("httpExecuter, http code != 200"),
			nil,
		))

}

// cancel
func (h *httpExecuter) Stop() error {
	h.cancel()
	return nil
}

func createHTTPExecuter(ctx context.Context, param *model.Param) Executer {
	h := &httpExecuter{}
	h.ctx, h.cancel = context.WithCancel(ctx)

	h.param = &param.Executer
	h.req = gout.New()
	return h
}
