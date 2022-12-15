package gate

import (
	"crypto/md5"
	"fmt"

	"gorm.io/gorm"
)

var (
	column             = []string{"id", "user_name", "email", "rule"}
	columnWithPassword = []string{"id", "user_name", "password", "email", "rule"}
)

type Page struct {
	Limit int `form:"limit"`
	Page  int `form:"page"`
}

type LoginDB struct {
	DB *gorm.DB
}

type LoginCore struct {
	gorm.Model
	UserName string `gorm:"index:,unique;not null" json:"username" binding:"required"`
	Email    string `gorm:"index:,unique" json:"email"`
	Password string `gorm:"type:varchar(50)" json:"password" binding:"required"`
	Rule     string `gorm:"type:varchar(10)" json:"rule"`
}

// 初始化
func newLoginDB(db *gorm.DB) (*LoginDB, error) {
	return &LoginDB{DB: db}, nil
}

func md5sum(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// 插入数据
func (l *LoginDB) insert(login *LoginCore) error {
	// 密码换成md5串
	login.Password = fmt.Sprintf("%x", md5.Sum([]byte(login.Password)))
	return l.DB.Create(login).Error
}

// 查询数据
func (l *LoginDB) queryNeedPassword(login LoginCore) (ld LoginCore, err error) {
	err = l.DB.Model(&LoginCore{}).Select(column, "password").Where("user_name = ? AND password = ?", login.UserName, md5sum(login.Password)).First(&ld).Error
	return
}

// 查询数据
func (l *LoginDB) query(login LoginCore) (ld LoginCore, err error) {
	err = l.DB.Model(&LoginCore{}).Select(column).Where("user_name = ?", login.UserName).First(&ld).Error
	ld.Password = ""
	return
}

// 更新
func (l *LoginDB) update(login *LoginCore) (err error) {
	err = l.DB.Model(&LoginCore{}).Where("id = ?", login.ID).Updates(login).Error
	return
}

// 删除用户
func (l *LoginDB) delete(login *LoginCore) error {
	l.DB.Delete(login)
	return nil
}

// 查看用户信息
func (l *LoginDB) queryAndPage(p Page) (rv []LoginCore, count int64, err error) {
	err = l.DB.Debug().Model(&LoginCore{}).Select(column).Offset(p.Page - 1).Limit(p.Limit).Find(&rv).Error
	if err != nil {
		return
	}

	l.DB.Debug().Model(&LoginCore{}).Count(&count)
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
