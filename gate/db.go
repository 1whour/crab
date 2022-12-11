package gate

import (
	"crypto/md5"
	"fmt"

	"gorm.io/gorm"
)

type Page struct {
	Size int `form:"size"`
	Page int `form:"page"`
}

type LoginDB struct {
	DB *gorm.DB
}

type LoginCore struct {
	gorm.Model
	UserName string `gorm:"index:,unique" json:"userName" binding:"required"`
	Email    string `gorm:"index:,unique" json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 初始化
func newLoginDB(db *gorm.DB) (*LoginDB, error) {

	return &LoginDB{DB: db}, nil
}

// 插入数据
func (l *LoginDB) insert(login *LoginCore) error {
	// 密码换成md5串
	login.Password = fmt.Sprintf("%x", md5.Sum([]byte(login.Password)))
	return l.DB.Create(login).Error
}

// 查询数据
func (l *LoginDB) query(login *LoginCore) error {
	return l.DB.Where("user_name = ? AND password = ?", login.UserName, login.Password).First(login).Error
}

// 删除用户
func (l *LoginDB) delete(login *LoginCore) error {
	l.DB.Delete(login)
	return nil
}

// 查看用户信息
func (l *LoginDB) userInfo(p Page) (rv []LoginDB, err error) {
	err = l.DB.Offset(p.Page).Limit(p.Size).Find(&rv).Error
	return
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
