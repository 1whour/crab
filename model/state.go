package model

import "encoding/json"

const (
	CanRun  = "canrun"  //可以运行，任务被创建时的状态
	Running = "running" //任务被分配之后，运行中, oneRuntime记录runtimeNode的编号
	Stop    = "stop"    //这个任务被中止
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
}

func (s State) IsRunning() bool {
	return s.State == Running
}

func (s State) IsStop() bool {
	return s.State == Stop
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
