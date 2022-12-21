package model

import (
	"time"
)

type ResultCore struct {
	ID uint `gorm:"primarykey"`
	//gorm.Model
	// 任务id, 普通索引
	TaskID string `gorm:"index;not null;type:varchar(40)" json:"task_id" binding:"required"`
	// 任务名，普通索引
	TaskName string `gorm:"index;type:varchar(40)" json:"task_name" binding:"required"`
	// 任务类型
	TaskType string `gorm:"type:enum('cron');default:cron" json:"task_type"`
	// 任务开始时间
	StartTime time.Time `gorm:"columm:start_time" json:"start_time"`
	// 任务结束时间
	EndTime time.Time `gorm:"columm:end_time" json:"end_time"`
	// 执行状态,
	TaskStatus string `gorm:"type:enum('success', 'failed');default:'success';columm:task_status" json:"task_status"`
	// 执行结果
	Result string `gorm:"type:varchar(512);columm:result" json:"result"`
}

type ResultCoreDelete struct {
	ID uint `gorm:"primarykey"`
	//gorm.Model
	// 任务id, 普通索引
	TaskID string `gorm:"index;not null;type:varchar(40)" json:"task_id" binding:"required"`
}
