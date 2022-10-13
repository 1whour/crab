package client

import "github.com/gin-gonic/gin"

type Handler func(c *gin.Context)

type Executor interface {
	Register(taskName string, handler Handler) //注册
}
