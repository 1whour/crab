package model

import (
	"encoding/json"
	"time"
)

const (
	CanRun  = "canrun"  //可以运行，任务被创建时的状态
	Running = "running" //任务被分配之后，运行中, oneRuntime字段绑定runtimeNode的节点
	Failed  = "failed"  //这个任务发送到runtime节点失败
)

// 集群稳定的前提下(当runtime的个数>=1 gate的个数>=1)，什么样的任务可以被恢复?

// 1.如果是Create和Update的任务，任务绑定的runtime是空, State是任何状态，都需要被恢复, 这是一个还需要被运行的状态
// 2.如果是Stop和Rm的任务, 如果runtimeNode不为空。Number == 0时会尝试一次
type State struct {
	// 每个任务从全局队列中分配到本地队列都会绑定一个runtime
	RuntimeNode string
	//运行状态, CanRun, Running, failed
	State string
	//任务的状态, Create, Update, Stop, Rm
	Action string
	//任务下次的次数, 每次下发就会+1
	Successed int
	//创建时间, 日志作用
	CreateTime time.Time
	//更新时间, 日志作用
	UpdateTime time.Time
}

func (s State) IsFailed() bool {
	return s.State == Failed
}

func (s State) IsRunning() bool {
	return s.State == Running
}

func (s State) IsCanRun() bool {
	return s.State == CanRun
}

// 创建任务时调用
func NewState() ([]byte, error) {
	now := time.Now()
	return json.Marshal(&State{State: CanRun, Action: Create, CreateTime: now, UpdateTime: now})
}

// 删除，更新，stop时调用
func UpdateState(value []byte, runtimeNode string, state string, action string) ([]byte, error) {
	s, err := ValueToState(value)
	if err != nil {
		return nil, err
	}

	s.RuntimeNode = runtimeNode
	s.State = state
	s.Action = action
	s.UpdateTime = time.Now()
	return json.Marshal(&s)
}

func ValueToState(value []byte) (s State, err error) {
	err = json.Unmarshal(value, &s)
	return
}
