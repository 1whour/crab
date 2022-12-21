package gate

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/1whour/crab/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testInitResultTable(t *testing.T) *ResultTable {
	passwd := os.Getenv("CRAB_MYSQL_PASSWD")
	dsn := fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/crab?charset=utf8mb4&parseTime=True&loc=Local", passwd)

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	assert.NoError(t, err)
	result := newResultTable(db)

	result.resetTable()

	return result
}

func Test_ResultTable_Insert(t *testing.T) {

	var err error
	result := testInitResultTable(t)

	now := time.Now()
	id := uuid.New().String()
	insertData := model.ResultCore{
		TaskID:   id,
		TaskName: "test", TaskType: "cron",
		StartTime:  now.Add(-time.Second),
		EndTime:    now,
		TaskStatus: "success",
		Result:     "exec ok",
	}
	err = result.insert(insertData)
	assert.NoError(t, err)

	err = result.insert(insertData)
	assert.NoError(t, err)

	rv, _, err := result.queryAndPage(PageResult{TaskID: id, Page: Page{Limit: 2}})
	assert.Equal(t, rv[0].TaskID, id)
	assert.Equal(t, rv[1].TaskID, id)
	assert.NoError(t, err)
}

func Test_Result_GetList(t *testing.T) {

	var err error
	result := testInitResultTable(t)

	insertAll := []model.ResultCore{}
	for i := 0; i < 15; i++ {
		val := model.ResultCore{TaskID: fmt.Sprintf("id:%d", i), TaskName: fmt.Sprintf("name:%d", i), StartTime: time.Now(), EndTime: time.Now()}
		err = result.insert(val)
		insertAll = append(insertAll, val)

		assert.NoError(t, err)
	}

	all, _, err := result.queryAndPage(PageResult{Page: Page{Limit: 10, Page: 1}})
	assert.NotEqual(t, len(all), 0)
	for i, v := range insertAll[:10] {
		assert.Equal(t, v.TaskID, all[i].TaskID)
		assert.Equal(t, v.TaskName, all[i].TaskName)
	}

}
