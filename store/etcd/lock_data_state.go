package etcd

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func (e *EtcdStore) LockCreateDataAndState(ctx context.Context, taskName string, globalData string) error {
	return e.LockUnlock(ctx, taskName, func() error {
		return e.CreateDataAndState(ctx, taskName, globalData)
	})

}

func (e *EtcdStore) LockUpdateLocalAndGlobal(ctx context.Context, taskName string, runtimeNode string, rsp *clientv3.GetResponse, action string) (err error) {
	return e.LockUnlock(ctx, taskName, func() error {
		return e.UpdateLocalAndGlobal(ctx, taskName, runtimeNode, rsp, action)
	})
}

func (e *EtcdStore) LockUpdateCallStateSuccessed(ctx context.Context, taskName string) error {
	return e.LockUnlock(ctx, taskName, func() error {
		return e.UpdateCallStateSuccessed(ctx, taskName)
	})
}

func (e *EtcdStore) LockUpdateCallStateFailed(ctx context.Context, taskName string) error {
	return e.LockUnlock(ctx, taskName, func() error {
		return e.UpdateCallStateFailed(ctx, taskName)
	})
}
