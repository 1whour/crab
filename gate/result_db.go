package gate

import (
	"github.com/1whour/crab/model"
	"gorm.io/gorm"
)

var (
	resultColumm = []string{"id", "task_id", "task_name", "task_type", "task_status", "result", "start_time", "end_time"}
)

type PageResult struct {
	Page
	TaskID     string `form:"task_id" json:"task_id"`
	NeedUpdate bool   `form:"need_update"`
}

type ResultTable struct {
	*gorm.DB
}

// 新建
func newResultTable(db *gorm.DB) *ResultTable {
	return &ResultTable{DB: db}
}

// 插入
func (r *ResultTable) insert(result model.ResultCore) error {
	return r.DB.Create(&result).Error
}

// 查询
func (l *ResultTable) queryAndPage(p PageResult) (rv []model.ResultCore, count int64, err error) {
	c := resultColumm
	order := ""
	if len(p.Sort) > 0 {
		if p.Sort[0] == '-' {
			order = p.Sort[1:] + " desc"
		}
	}
	db := l.DB.Debug().Model(&model.ResultCore{}).Select(c)

	if len(p.TaskID) > 0 {
		db.Where("task_id", p.TaskID)
	}

	if !p.StartTime.IsZero() {
		db.Where("start_time >= ?", p.StartTime)
	}

	if !p.EndTime.IsZero() {
		db.Where("end_time <= ?", p.EndTime)
	}

	err = db.
		Order(order).
		Offset(p.Page.Page - 1).
		Limit(p.Limit).
		Find(&rv).Error
	if err != nil {
		return
	}

	l.DB.Debug().Model(&model.ResultCore{}).Count(&count)
	return
}

// 删除
func (r *ResultTable) delete(p PageResult) (err error) {
	db := r.DB.Unscoped()
	if len(p.TaskID) > 0 {
		db.Where("task_id", p.TaskID)
	}

	if !p.StartTime.IsZero() {
		db.Where("start_time >= ?", p.StartTime)
	}

	if !p.EndTime.IsZero() {
		db.Where("end_time <= ?", p.EndTime)
	}

	db.Delete(model.ResultCore{})
	return
}

// 单元测试用
func (l *ResultTable) resetTable() {
	l.deleteTable()
	l.createTable()
}

// 创建表, 单元测试用
func (l *ResultTable) createTable() {
	l.DB.AutoMigrate(&model.ResultCore{})
}

// 清空表, 单元测试用
func (l *ResultTable) deleteTable() error {
	return l.DB.Migrator().DropTable(&model.ResultCore{})
}
