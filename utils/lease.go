package utils

import (
	"context"
	"time"

	"github.com/1whour/ktuo/slog"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func NewLeaseWithKeepalive(ctx context.Context, log *slog.Slog, client *clientv3.Client, ttl time.Duration) (clientv3.LeaseID, error) {
	// 创建一个lease对象
	lease := clientv3.NewLease(client)
	// 申请一个ttl/time.Second的lease
	leaseGrantResp, err := lease.Grant(ctx, int64(ttl/time.Second))
	if err != nil {
		return 0, err
	}

	// 拿到lease ID
	leaseID := leaseGrantResp.ID

	// 自动续约
	keepRespChan, err := lease.KeepAlive(ctx, leaseID)

	go func() {

		// 自动应答
		for rsp := range keepRespChan {
			if rsp == nil {
				log.Warn().Msgf("The lease has expired")
				return
			}
		}
	}()

	return leaseID, nil
}

func NewLease(ctx context.Context, log *slog.Slog, client *clientv3.Client, ttl time.Duration) (clientv3.Lease, clientv3.LeaseID, error) {
	// 创建一个lease对象
	lease := clientv3.NewLease(client)
	// 申请一个ttl/time.Second的lease
	leaseGrantResp, err := lease.Grant(ctx, int64(ttl/time.Second))
	return lease, leaseGrantResp.ID, err
}
