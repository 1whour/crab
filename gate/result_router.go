package gate

import (
	"time"

	"github.com/1whour/crab/model"
	"github.com/antlabs/deepcopy"
	"github.com/gin-gonic/gin"
)

type Page struct {
	Limit     int       `form:"limit"`
	Page      int       `form:"page"`
	Sort      string    `form:"sort"`
	StartTime time.Time `form:"start_time"`
	EndTime   time.Time `form:"end_time"`
}

// 保存result结果
func (g *Gate) saveResult(ctx *gin.Context) {
	// 获取数据
	var rc model.ResultCore
	if err := ctx.ShouldBindJSON(&rc); err != nil {
		g.error2(ctx, 500, err.Error())
		return
	}

	// 写入数据库
	if err := g.resultTable.insert(rc); err != nil {
		g.error2(ctx, 500, err.Error())
		return
	}
}

// 获取列表里面的数据
func (g *Gate) getResultList(c *gin.Context) {
	p := PageResult{}
	if err := c.ShouldBindQuery(&p); err != nil {
		g.error(c, 500, err.Error())
		return
	}

	// 默认10
	if p.Limit == 0 {
		p.Limit = 10
	}

	rv, count, err := g.resultTable.queryAndPage(p)
	if err != nil {
		g.error(c, 500, err.Error())
		return
	}

	c.JSON(200, wrapData{Data: userList{
		Total: count,
		Items: rv,
	}})
}

// 删除日志
func (g *Gate) deleteResult(c *gin.Context) {

	lc := model.ResultCoreDelete{}

	err := c.ShouldBindJSON(&lc)
	if err != nil {
		g.error2(c, 500, err.Error())
		return
	}

	lc2 := model.ResultCore{}
	deepcopy.Copy(&lc2, &lc).Do()
	g.resultTable.delete(&lc2)
	c.JSON(200, wrapData{})
}
