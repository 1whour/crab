package etcd

import (
	"context"

	"github.com/1whour/crab/model"
)

func (e *EtcdStore) LockCreateDataAndState(ctx context.Context, taskName string, req *model.Param) error {
	return e.LockUnlock(ctx, taskName, func() error {
		return e.CreateDataAndState(ctx, taskName, req)
	})

}

func (e *EtcdStore) LockUpdateAction(ctx context.Context, taskName string, req *model.OnlyParam, rspModRevision int64, state string, action string) error {
	return e.LockUnlock(ctx, taskName, func() error {
		return e.UpdateAction(ctx, req, rspModRevision, state, action)
	})
}

func (e *EtcdStore) LockUpdateDataAndState(ctx context.Context, taskName string, req *model.Param, rspModRevision int64, state string, action string) error {
	return e.LockUnlock(ctx, taskName, func() error {
		return e.UpdateDataAndState(ctx, req, rspModRevision, state, action)
	})
}

func (e *EtcdStore) LockUpdateCallStateSuccessed(ctx context.Context, taskName string) error {
	return e.LockUnlock(ctx, taskName, func() error {
		return e.UpdateCallStateSuccessed(ctx, taskName)
	})
}
