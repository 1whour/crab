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
	ID string
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

	where := map[string]any{}
	if len(p.ID) > 0 {
		where["task_id"] = p.ID
	}

	err = l.DB.Debug().Model(&model.ResultCore{}).Select(c).
		Where(where).
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
func (r *ResultTable) delete(result *model.ResultCore) (err error) {
	r.DB.Unscoped().Delete(result)
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
