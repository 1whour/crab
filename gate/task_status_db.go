package gate

import (
	"time"

	"github.com/1whour/crab/model"
	"gorm.io/gorm"
)

var (
	statusColumm = []string{"task_name", "trigger", "trigger_value", "status", "create_time", "update_time", "runtime_id"}
)

type pageStatus struct {
	Page `gorm:"-" json:"page"`

	Format string `gorm:"-" form:"format" json:"format"`
	// 任务名
	TaskName string `gorm:"index:,unique;not null;type:varchar(40)" json:"task_name"`
	// cron任务或者一次性任务
	Trigger string `gorm:"type:enum('cron', 'once');default:'cron';column:trigger" json:"trigger"`
	// cron表达式或者
	TriggerValue string `gorm:"type:varchar(40);column:trigger_value" json:"trigger_value"`

	// 任务状态
	// 这里和etcd的action和status有所区别，把etcd的状态归一化成stop, running两种
	Status string `gorm:"type:enum('stop', 'running');default:'stop';columm:status" json:"status"`

	// 创建时间
	CreateTime time.Time `gorm:"columm:create_time" json:"create_time"`

	// 更新时间
	UpdateTime time.Time `gorm:"columm:update_time" json:"update_time"`

	// Runtime ID, runtime唯一无二的标识
	RuntimeID string `gorm:"column:runtime_id;type:varchar(40)" json:"runtime_id"`
}

func paramToStatus(req *model.Param) (rv pageStatus) {
	rv.TaskName = req.Executer.TaskName
	if len(req.Trigger.Cron) > 0 {
		rv.Trigger = "cron"
		rv.TriggerValue = req.Trigger.Cron
	} else {
		rv.Trigger = "once"
		rv.TriggerValue = req.Trigger.Once
	}

	if req.IsCreate() || req.IsUpdate() {
		rv.Status = "running"
	}
	return
}

func onlyParamToStatus(req *model.OnlyParam) (rv pageStatus) {

	switch req.Action {
	case model.Stop:
		rv.Status = "stop"
	case model.Continue:
		rv.Status = "running"
	}

	return
}

// 状态表
type StatusTable struct {
	*gorm.DB
}

// 新建
func newStatusTable(db *gorm.DB) *StatusTable {
	return &StatusTable{DB: db}
}

// 插入
func (r *StatusTable) insert(result pageStatus) error {
	result.CreateTime = time.Now()
	result.UpdateTime = time.Now()
	return r.DB.Create(&result).Error
}

// 更新
func (l *StatusTable) update(result pageStatus) (err error) {
	result.UpdateTime = time.Now()
	err = l.DB.Model(&pageStatus{}).Where("task_name = ?", result.TaskName).Updates(result).Error
	return
}

// 查询
func (l *StatusTable) queryAndPage(p pageStatus) (rv []pageStatus, count int64, err error) {
	if p.Limit == 0 {
		p.Limit = 10
	}
	c := statusColumm
	order := ""
	if len(p.Sort) > 0 {
		if p.Sort[0] == '-' {
			order = p.Sort[1:] + " desc"
		}
	}
	db := l.DB.Debug().Model(&pageStatus{}).Select(c)

	if len(p.TaskName) > 0 {
		db.Where("task_name", p.TaskName)
	}

	if !p.StartTime.IsZero() {
		db.Where("create_time >= ?", p.CreateTime)
	}

	if !p.EndTime.IsZero() {
		db.Where("end_time <= ?", p.UpdateTime)
	}

	err = db.
		Order(order).
		Offset(p.Page.Page - 1).
		Limit(p.Limit).
		Find(&rv).Error
	if err != nil {
		return
	}

	l.DB.Debug().Model(&pageStatus{}).Count(&count)
	return
}

// 删除
func (r *StatusTable) delete(p pageStatus) (err error) {
	db := r.DB.Unscoped()
	if len(p.TaskName) > 0 {
		db.Where("task_name", p.TaskName)
	}

	if !p.StartTime.IsZero() {
		db.Where("create_time >= ?", p.StartTime)
	}

	if !p.EndTime.IsZero() {
		db.Where("update_time <= ?", p.EndTime)
	}

	db.Debug().Delete(pageStatus{})
	return
}

// 单元测试用
func (l *StatusTable) resetTable() error {
	if err := l.deleteTable(); err != nil {
		return err
	}
	return l.createTable()
}

// 创建表, 单元测试用
func (l *StatusTable) createTable() error {
	return l.DB.AutoMigrate(&pageStatus{})
}

// 清空表, 单元测试用
func (l *StatusTable) deleteTable() error {
	return l.DB.Migrator().DropTable(&pageStatus{})
}
