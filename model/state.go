package model

import "encoding/json"

const (
	CanRun  = "canrun"  //可以运行，任务被创建时的状态
	Running = "running" //任务被分配之后，运行中, oneRuntime字段绑定runtimeNode的节点
	//Succeeded = "succeeded" //这个任务成功地发生到runtime节点, 目前还不需要这个状态
	Failed = "failed" //这个任务发送到runtime节点失败
)

var (
	CanRunJSON  string
	RunningJSON string
)

func init() {
	canRun, _ := MarshalToJson("", CanRun)
	running, _ := MarshalToJson("", Running)

	CanRunJSON = string(canRun)
	RunningJSON = string(running)
}

type State struct {
	RuntimeNode string //这个任务绑定的runtime节点
	State       string //运行状态
	Action      string //任务的状态, TODO
}

/*
func (s State) IsSucceeded() bool {
	return s.State == Succeeded
}
*/

func (s State) IsFailed() bool {
	return s.State == Failed
}

func (s State) IsRunning() bool {
	return s.State == Running
}

func (s State) IsCanRun() bool {
	return s.State == CanRun
}

func MarshalToJson(runtimeNode string, state string) ([]byte, error) {
	return json.Marshal(&State{RuntimeNode: runtimeNode, State: state})
}

func OnlyUpdateRuntimeNode(value []byte, runtimeNode string) ([]byte, error) {
	s, err := ValueToState(value)
	if err != nil {
		return nil, err
	}

	s.RuntimeNode = runtimeNode
	return json.Marshal(&s)
}

func OnlyUpdateState(value []byte, state string) ([]byte, error) {
	s, err := ValueToState(value)
	if err != nil {
		return nil, err
	}

	s.State = state
	return json.Marshal(&s)
}

func ValueToState(value []byte) (s State, err error) {
	err = json.Unmarshal(value, &s)
	return
}
