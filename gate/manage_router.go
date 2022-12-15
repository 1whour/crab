package gate

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guonaihong/gutil/jwt"
	"gorm.io/gorm"
)

const (
	secretToken = "@@112233"
	serverName  = "ktuo"
)

type userInfoData struct {
	Token        string   `json:"token"`
	Introduction string   `json:"introduction"`
	Avatar       string   `json:"avatar"`
	Name         string   `json:"name"`
	Rule         []string `json:"rule"`
}

type wrapToken struct {
	Token string `json:"token"`
}

type wrapData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type userList struct {
	Total int64 `json:"total"`
	Items any   `json:"items"`
}

// 注册账号
func (g *Gate) register(c *gin.Context) {
	lc := LoginCore{}
	if err := c.ShouldBindJSON(&lc); err != nil {
		g.error(c, 500, err.Error())
		return
	}

	g.Debug().Msgf("register info :%v", lc)
	if err := g.loginDb.insert(&lc); err != nil {
		g.error(c, 500, err.Error())
		return
	}
}

// 登录
func (g *Gate) login(c *gin.Context) {
	lc := LoginCore{}

	if err := c.ShouldBindJSON(&lc); err != nil {
		g.error(c, 500, err.Error())
		return
	}

	rv, err := g.loginDb.queryNeedPassword(lc)
	if err != nil {
		g.error(c, 500, err.Error())
		return
	}

	if rv.UserName != lc.UserName || rv.Password != md5sum(lc.Password) {
		g.Error().Msgf("rv.UserName:(%s):req.UserName(%s), rv.Password:(%s), md5sum(%s)", rv.UserName, lc.UserName,
			rv.Password, md5sum(lc.Password))
		g.error(c, 500, "wrong account")
		return
	}

	token, err := jwt.GenToken(time.Hour*24, lc.UserName, secretToken)
	if err != nil {
		g.error(c, 500, err.Error())
		return
	}

	//c.Header("token", token)
	c.JSON(200, wrapData{
		Data: wrapToken{token},
	})
}

func (g *Gate) logout(c *gin.Context) {
	c.JSON(200, wrapData{})
}

// 删除
func (g *Gate) deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		g.error(c, 500, err.Error())
		return
	}

	lc := LoginCore{Model: gorm.Model{ID: uint(id)}}

	g.loginDb.delete(&lc)
	c.JSON(200, wrapData{})
}

// 获取用户信息
func (g *Gate) getUserInfo(c *gin.Context) {

	val, err := jwt.ParseToken(c.Request.Header.Get(tokenHeader), secretToken)
	if err != nil {
		g.error(c, 500, err.Error())
		return
	}

	g.Debug().Msgf("token:%#v", val)
	lc := LoginCore{UserName: val.Issuer}
	rv, err := g.loginDb.query(lc)
	if err != nil {
		g.error(c, 500, err.Error())
		return
	}
	c.JSON(200, wrapData{
		Data: userInfoData{
			Rule: []string{rv.Rule}},
	},
	)
}

// 获取用户信息列表
func (g *Gate) GetUserInfoList(c *gin.Context) {
	p := Page{}
	if err := c.ShouldBindQuery(&p); err != nil {
		g.error(c, 500, err.Error())
		return
	}

	// 默认10
	if p.Limit == 0 {
		p.Limit = 10
	}

	rv, count, err := g.loginDb.queryAndPage(p)
	if err != nil {
		g.error(c, 500, err.Error())
		return
	}

	c.JSON(200, wrapData{Data: userList{
		Total: count,
		Items: rv,
	}})
}
