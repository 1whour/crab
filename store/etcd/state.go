package etcd

import (
	"context"
	"fmt"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

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

func (e *EtcdStore) UpdateState(ctx context.Context, taskName string, globalData string, rspModRevision int64, stateAction string) error {

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
	newValue, err := model.OnlyUpdateState(rspState.Kvs[0].Value, stateAction)
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
