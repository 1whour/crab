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
// 2.如果是Stop和Rm的任务, 如果runtimeNode不为空。InRuntime == 0时会尝试一次
type State struct {
	// 每个任务从全局队列中分配到本地队列都会绑定一个runtime
	RuntimeNode string
	//运行时状态, CanRun, Running, failed
	State string
	//任务本身的状态, Create, Update, Stop, Rm
	Action string
	// true表示任务正在运行，false表示任务没有运行
	InRuntime bool
	//创建时间, 日志作用
	CreateTime time.Time
	//更新时间, 日志作用
	UpdateTime time.Time
	// ack消息标记
	Ack bool
	// 从数据字段移过来
	Kind string
	// 是否是Lambda函数，必须要绑定
	Lambda bool
}

func (s State) IsOneRuntime() bool {
	return s.Kind == "" || s.Kind == "oneRuntime"
}

func (s State) IsBroadcast() bool {
	return s.Kind == "broadcast"
}

func (s State) IsCreate() bool {
	return s.Action == Create
}

func (s State) IsUpdate() bool {
	return s.Action == Update
}

func (s State) IsStop() bool {
	return s.Action == Stop
}

func (s State) IsRemove() bool {
	return s.Action == Rm
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
func NewState(kind string, lambda bool) ([]byte, error) {
	now := time.Now()
	return json.Marshal(&State{State: CanRun, Action: Create, CreateTime: now, UpdateTime: now, Kind: kind, Lambda: lambda})
}

func UpdateStateAck(value []byte, successed bool) ([]byte, error) {
	s, err := ValueToState(value)
	if err != nil {
		return nil, err
	}
	if successed {
		switch s.Action {
		case Create, Update:
			s.InRuntime = true
		case Stop, Rm:
			s.InRuntime = false
		default:
			panic("未知的新状态:" + s.Action)
		}
	} else {
		s.State = Failed
	}
	// ack消费标记
	s.Ack = true

	s.UpdateTime = time.Now()
	return json.Marshal(s)
}

// 删除，更新，stop时调用
func UpdateState(value []byte, runtimeNode string, state string, action string) ([]byte, error) {
	s, err := ValueToState(value)
	if err != nil {
		return nil, err
	}

	if len(runtimeNode) > 0 {
		s.RuntimeNode = runtimeNode
	}
	s.State = state
	s.Action = action
	s.UpdateTime = time.Now()
	s.Ack = false
	return json.Marshal(&s)
}

func ValueToState(value []byte) (s State, err error) {
	err = json.Unmarshal(value, &s)
	return
}
