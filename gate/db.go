package gate

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type LoginDB struct {
	DB *gorm.DB
}

type LoginCore struct {
	gorm.Model
	UserName string `gorm:"index:,unique" json:"userName" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 初始化
func newLoginDB(dsn string) (*LoginDB, error) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}))
	if err != nil {
		return nil, err
	}

	return &LoginDB{DB: db}, nil
}

// 插入数据
func (l *LoginDB) insert(login *LoginCore) error {
	return l.DB.Create(login).Error
}

// 查询数据
func (l *LoginDB) query(login *LoginCore) error {
	return l.DB.Where("user_name = ?", login.UserName).First(login).Error
}

// 单元测试用
func (l *LoginDB) resetTable() {
	l.deleteTable()
	l.createTable()
}

// 创建表, 单元测试用
func (l *LoginDB) createTable() {
	l.DB.AutoMigrate(&LoginCore{})
}

// 清空表, 单元测试用
func (l *LoginDB) deleteTable() error {
	return l.DB.Migrator().DropTable(&LoginCore{})
}
