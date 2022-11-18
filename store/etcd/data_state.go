package etcd

import (
	"context"
	"errors"
	"fmt"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 本文件包含任务保存到etcd, 以及任务从任务队列分发到本地队列的逻辑

type EtcdStore struct {
	defaultKVC    clientv3.KV
	defaultClient *clientv3.Client
}

func NewStore(EtcdAddr []string) (*EtcdStore, error) {

	defautlClient, err := utils.NewEtcdClient(EtcdAddr)
	if err != nil { //初始etcd客户端
		return nil, err
	}

	defaultKVC := clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	return &EtcdStore{
		defaultKVC:    defaultKVC,
		defaultClient: defautlClient,
	}, nil
}

// 创建全局状态与数据队列
func (e *EtcdStore) CreateDataAndState(ctx context.Context, taskName string, globalData string) error {

	// 创建数据队列
	globalTaskName := model.FullGlobalTask(taskName)

	// 创建状态队列
	globalTaskStateName := model.FullGlobalTaskState(taskName)
	// 创建全局状态队列里面的队列
	state, err := model.NewState()
	if err != nil {
		return err
	}

	txn := e.defaultKVC.Txn(ctx)
	txn.If(clientv3.Compare(clientv3.CreateRevision(globalTaskName), "=", 0)).
		Then(
			clientv3.OpPut(globalTaskName, globalData),
			clientv3.OpPut(globalTaskStateName, string(state)),
		).Else()

	txnRsp, err := txn.Commit()
	if err != nil {
		return fmt.Errorf("Transaction execution failed err :%v", err)
	}

	if !txnRsp.Succeeded {
		return fmt.Errorf("Transaction execution failed")
	}
	return nil
}

// 更新全局数据与状态队列
func (e *EtcdStore) UpdateDataAndState(ctx context.Context, taskName string, globalData string, rspModRevision int64, state string, action string) error {

	globalTaskName := model.FullGlobalTask(taskName)
	// 创建全局状态队列key名
	globalTaskStateName := model.FullGlobalTaskState(taskName)
	// 获取全局状态队列里面的值
	rspState, err := e.defaultKVC.Get(ctx, globalTaskStateName)
	if err != nil {
		return fmt.Errorf("get.globalTaskStateName err :%w", err)
	}

	//rspModRevision := rsp.Kvs[0].ModRevision
	rspStateModRevision := rspState.Kvs[0].ModRevision

	// 更新json中的State是CanRun
	newValue, err := model.UpdateState(rspState.Kvs[0].Value, "", state, action)
	if err != nil {
		return fmt.Errorf("updateTask, onlyUpdateState(CanRun) err :%v", err)
	}

	// 使用事务更新
	txn := e.defaultKVC.Txn(ctx)
	txn.If(clientv3.Compare(clientv3.ModRevision(globalTaskName), "=", rspModRevision),
		clientv3.Compare(clientv3.ModRevision(globalTaskStateName), "=", rspStateModRevision),
	).
		Then(
			clientv3.OpPut(globalTaskName, globalData),            //更新全局队列里面的数据
			clientv3.OpPut(globalTaskStateName, string(newValue)), //更新全局状态队列里面的状态
		).Else()

	// 提交事务
	txnRsp, err := txn.Commit()
	if err != nil {
		return err
	}

	if !txnRsp.Succeeded {
		return fmt.Errorf("Transaction execution failed")
	}
	return nil
}

// 更新本地队列和全局队列
func (e *EtcdStore) UpdateLocalAndGlobal(ctx context.Context, taskName string, runtimeNode string, rsp *clientv3.GetResponse, action string) (err error) {

	modRevision := rsp.Kvs[0].ModRevision
	fullTaskState := model.FullGlobalTaskState(taskName)

	// 生成本地队列的名字, 包含runtime和taskName
	ltaskPath := model.RuntimeNodeToLocalTask(runtimeNode, taskName)

	// 更新状态中的runtimeNode
	newValue, err := model.UpdateState(rsp.Kvs[0].Value, runtimeNode, model.Running, action)
	if err != nil {
		return err
	}

	txn := e.defaultKVC.Txn(ctx)
	txnRsp, err := txn.If(
		clientv3.Compare(clientv3.ModRevision(fullTaskState), "=", modRevision),
	).Then(
		clientv3.OpPut(fullTaskState, string(newValue)),
		// 向本地队列写入任务, 目前本地队列的值没啥作用
		clientv3.OpPut(ltaskPath, model.CanRun),
	).Commit()

	if err != nil {
		return err
	}

	if !txnRsp.Succeeded {
		err = errors.New("Transaction execution failed")
	}
	return
}
