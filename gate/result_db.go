package gate

import (
	"time"

	"gorm.io/gorm"
)

type ResultCore struct {
	gorm.Model
	// 任务id
	ID string `gorm:"index:,unique;not null;type:varchar(40)" json:"id"`
	// 任务名
	TaskName string `gorm:"type:varchar(40)" json:"taskName"`
	// 任务类型
	TaskType string `gorm:"type":enum('cron');default:"cron"`

	// 任务开始时间
	StartTime time.Time
	// 任务结束时间
	EndTime time.Time
	// 执行状态,
	Status string `gorm:"type:enum('success', 'failed');default:'success'"`
	// 执行结果
	Result string
}

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
func (r *ResultTable) insert(result *ResultTable) error {
	return r.DB.Create(result).Error
}

// 查询
func (l *ResultTable) queryAndPage(p PageResult) (rv []ResultTable, count int64, err error) {
	c := column
	order := ""
	if len(p.Sort) > 0 {
		if p.Sort[0] == '-' {
			order = p.Sort[1:] + " desc"
		}
	}

	where := map[string]any{}
	if len(p.ID) > 0 {
		where["id"] = p.ID
	}

	err = l.DB.Debug().Model(&ResultTable{}).Select(c).
		Where(where).
		Order(order).
		Offset(p.Page.Page - 1).
		Limit(p.Limit).
		Find(&rv).Error
	if err != nil {
		return
	}

	l.DB.Debug().Model(&ResultTable{}).Count(&count)
	return
}

// 删除
func (r *ResultTable) delete(result *ResultCore) (err error) {
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
	l.DB.AutoMigrate(&ResultTable{})
}

// 清空表, 单元测试用
func (l *ResultTable) deleteTable() error {
	return l.DB.Migrator().DropTable(&ResultTable{})
}
