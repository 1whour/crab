package gate

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 此函数依赖mysql是否存在
func Test_Login_Create(t *testing.T) {
	passwd := os.Getenv("KTUO_MYSQL_PASSWD")
	dsn := fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local", passwd)

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}))
	assert.NoError(t, err)
	login, err := newLoginDB(db)
	assert.NoError(t, err)

	login.resetTable()

	err = login.insert(&LoginCore{UserName: "guo", Password: "123"})
	assert.NoError(t, err)
	login.insert(&LoginCore{UserName: "guo", Password: "123"})
	assert.Error(t, err)

	login.query()
}

func Test_Login_Delete(t *testing.T) {

}

func Test_Login_Update(t *testing.T) {

}

func Test_Login_Get(t *testing.T) {

	passwd := os.Getenv("KTUO_MYSQL_PASSWD")
	dsn := fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local", passwd)

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}))
	assert.NoError(t, err)
	login, err := newLoginDB(db)
	assert.NoError(t, err)

	login.resetTable()

	err = login.insert(&LoginCore{UserName: "guo", Password: "123"})
	assert.NoError(t, err)
	login.insert(&LoginCore{UserName: "guo", Password: "123"})
	assert.Error(t, err)

	for i := 0; i < 100; i++ {
		err = login.insert(&LoginCore{UserName: "g1", Password: "1"})
		assert.NoError(t, err)
	}

	login.query()
}
