package gate

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 注册
func (g *Gate) register(c *gin.Context) {
	lc := LoginCore{}
	if err := c.ShouldBindJSON(&lc); err != nil {
		g.error(c, 500, err.Error())
		return
	}

	if err := g.loginDb.insert(&lc); err != nil {
		g.error(c, 500, err.Error())
		return
	}
}

// 登录
func (g *Gate) login(c *gin.Context) {
	lc := LoginCore{}

	if err := g.loginDb.query(&lc); err != nil {
		g.error(c, 500, err.Error())
		return
	}
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
}

// 获取用户信息
func (g *Gate) userInfo(c *gin.Context) {
	p := Page{}
	if err := c.ShouldBindQuery(&p); err != nil {
		g.error(c, 500, err.Error())
		return
	}
}
