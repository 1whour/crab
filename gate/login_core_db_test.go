package gate

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testInitTable(t *testing.T) *LoginDB {
	passwd := os.Getenv("CRAB_MYSQL_PASSWD")
	dsn := fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/crab?charset=utf8mb4&parseTime=True&loc=Local", passwd)

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	assert.NoError(t, err)
	login, err := newLoginTable(db)
	assert.NoError(t, err)

	login.resetTable()

	return login
}

// 此函数依赖mysql是否存在
func Test_Login_Create(t *testing.T) {

	var err error
	login := testInitTable(t)

	err = login.insert(&LoginCore{UserName: "guo", Password: "123", Email: "1@qq.com"})
	assert.NoError(t, err)
	err = login.insert(&LoginCore{UserName: "guo", Password: "123"})
	assert.Error(t, err)

	rv, err := login.queryNeedPassword(LoginCore{UserName: "guo", Password: "123"})
	assert.Equal(t, LoginCore{Model: gorm.Model{ID: 1}, UserName: "guo", Email: "1@qq.com", Password: md5sum("123")}, rv)
	assert.NoError(t, err)
}

// 测试删除，先插入，删除，查询没有为正确
func Test_Login_Delete(t *testing.T) {
	var err error
	login := testInitTable(t)

	err = login.insert(&LoginCore{UserName: "guo", Password: "123", Email: "1@qq.com"})
	assert.NoError(t, err)

	err = login.delete(&LoginCore{UserName: "guo", Password: "123", Email: "1@qq.com"})
	assert.NoError(t, err)

	rv, err := login.queryNeedPassword(LoginCore{UserName: "guo"})
	//rv, err := login.queryNeedPassword(LoginCore{UserName: "guo", Password: "123"})
	assert.Equal(t, rv, LoginCore{})
	assert.Error(t, err)
}

// 测试更新, 先插入，更新，查询数据是否符合预期
func Test_Login_Update(t *testing.T) {
	var err error
	login := testInitTable(t)

	err = login.insert(&LoginCore{UserName: "guo", Password: "123", Email: "1@qq.com", Rule: "admin"})
	assert.NoError(t, err)

	rv, err := login.query(LoginCore{UserName: "guo", Password: "123"})

	rv.UserName = "1whour"
	rv.Email = "2@qq.com"
	err = login.update(&rv)
	assert.NoError(t, err)
	rv2, err2 := login.query(rv)
	assert.NoError(t, err2)
	assert.Equal(t, rv, rv2)
}

func Test_Login_Get(t *testing.T) {

	var err error
	login := testInitTable(t)

	err = login.insert(&LoginCore{UserName: "guo", Password: "123", Email: "gnh@xx.com"})
	assert.NoError(t, err)
	err = login.insert(&LoginCore{UserName: "guo", Password: "123", Email: "gnh@xx.com"})
	assert.Error(t, err)

	rv, err := login.query(LoginCore{UserName: "guo"})
	assert.NoError(t, err)
	assert.Equal(t, rv.UserName, "guo")
	assert.Equal(t, rv.Email, "gnh@xx.com")

}

// 测试获取。#插入，获取
// 测试分页功能
func Test_Login_GetList(t *testing.T) {

	var err error
	login := testInitTable(t)

	insertAll := []LoginCore{}
	for i := 0; i < 15; i++ {
		val := LoginCore{UserName: fmt.Sprintf("g%d", i), Email: fmt.Sprintf("%d@x.com", i), Password: "111111", Rule: "admin"}
		err = login.insert(&val)
		val.Password = md5sum(val.Password)
		insertAll = append(insertAll, val)

		assert.NoError(t, err)
	}

	all, _, err := login.queryAndPage(PageLogin{Page: Page{Limit: 10, Page: 1}}, false)
	assert.NotEqual(t, len(all), 0)
	for i, v := range insertAll[:10] {
		assert.Equal(t, v.UserName, all[i].UserName)
		assert.Equal(t, v.Email, all[i].Email)
	}

}
