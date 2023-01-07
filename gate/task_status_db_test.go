package gate

import (
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testInitStatusTable(t *testing.T) *StatusTable {
	passwd := os.Getenv("CRAB_MYSQL_PASSWD")
	dsn := fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/crab?charset=utf8mb4&parseTime=True&loc=Local", passwd)

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	assert.NoError(t, err)
	status := newStatusTable(db)
	assert.NoError(t, err)

	err = status.resetTable()
	assert.NoError(t, err)

	return status
}

// 此函数依赖mysql是否存在
func Test_status_CreateAndGet(t *testing.T) {

	var err error
	status := testInitStatusTable(t)

	now := time.Now()
	err = status.insert(pageStatus{TaskName: "guo", Status: "stop", RuntimeID: "runtime id1", CreateTime: now, UpdateTime: now})
	assert.NoError(t, err)
	err = status.insert(pageStatus{TaskName: "guo", Status: "running", RuntimeID: "runtime id1"})
	assert.Error(t, err)

	rv, _, err := status.queryAndPage(pageStatus{TaskName: "guo"})
	assert.Equal(t, rv[0].TaskName, "guo")
	assert.Equal(t, rv[0].Status, "stop")
	assert.Equal(t, rv[0].RuntimeID, "runtime id1")
	assert.NoError(t, err)
}

// 测试删除，先插入，删除，查询没有为正确
func Test_status_Delete(t *testing.T) {
	var err error
	status := testInitStatusTable(t)

	err = status.insert(pageStatus{TaskName: "guo", Status: "stop", RuntimeID: "runtime id1"})
	assert.NoError(t, err)

	err = status.delete(pageStatus{TaskName: "guo"})
	assert.NoError(t, err)

	rv, _, err := status.queryAndPage(pageStatus{TaskName: "guo"})
	//rv, err := status.queryAndPage(pageStatus{UserName: "guo", Password: "123"})
	assert.Len(t, rv, 0)
	assert.NoError(t, err)
}

// 测试更新, 先插入，更新，查询数据是否符合预期
func Test_status_Update(t *testing.T) {
	var err error
	status := testInitStatusTable(t)

	err = status.insert(pageStatus{TaskName: "guo", Status: "stop", RuntimeID: "runtime id1"})
	assert.NoError(t, err)

	var rv pageStatus
	rv.TaskName = "guo"
	rv.Status = "running"
	rv.RuntimeID = "runtime id2"
	err = status.update(rv)
	assert.NoError(t, err)
	rv2, _, err2 := status.queryAndPage(rv)
	assert.NoError(t, err2)

	assert.Equal(t, rv2[0].TaskName, "guo")
	assert.Equal(t, rv2[0].Status, "running")
	assert.Equal(t, rv2[0].RuntimeID, "runtime id2")
}

// 测试获取。#插入，获取
// 测试分页功能
func Test_status_GetList(t *testing.T) {

	var err error
	status := testInitStatusTable(t)

	insertAll := []pageStatus{}
	now := time.Now()
	for i := 0; i < 15; i++ {
		val := pageStatus{TaskName: fmt.Sprintf("guo:%d", i), Status: "stop", CreateTime: now, UpdateTime: now, RuntimeID: fmt.Sprintf("runtimeID:%d", i)}
		err = status.insert(val)
		insertAll = append(insertAll, val)

		assert.NoError(t, err)
	}

	sort.Slice(insertAll, func(i, j int) bool {
		return insertAll[i].TaskName < insertAll[j].TaskName
	})

	all, _, err := status.queryAndPage(pageStatus{Page: Page{Limit: 10, Page: 1, Sort: "+task_name"}})
	assert.NotEqual(t, len(all), 0)
	for i, v := range insertAll[:10] {
		assert.Equal(t, v.TaskName, all[i].TaskName)
		assert.Equal(t, v.Status, all[i].Status)
		//assert.Equal(t, v.CreateTime, all[i].CreateTime)
		//assert.Equal(t, v.UpdateTime, all[i].UpdateTime)
	}

}
