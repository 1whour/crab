package gate

import (
	"encoding/json"
	"os"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func (r *Gate) autoNewAddrAndRegister() {
	r.autoNewAddr()
	_, err := defautlClient.Revoke(r.ctx, r.leaseID)
	if err != nil {
		r.Error().Msgf("revoke leaseID:%d %v\n", r.leaseID, err)
		return
	}
	go func() {
		if err := r.registerGateNode(); err != nil {
			r.Error().Msgf("registerGateNode fail:%s\n", err)
		}
	}()
}

// gate的地址
// model.GateNodePrefix 注册到/scheduler/gate/node/gate_name
func (r *Gate) registerGateNode() (err error) {
	defer func() {
		if err != nil {
			r.Error().Msgf("registerGateNode err:%s\n", err)
		}
	}()
	addr := r.ServerAddr
	if addr == "" {
		r.Error().Msgf("The service startup address is empty, please set -s ip:port")
		os.Exit(1)
	}

	leaseID, err := utils.NewLeaseWithKeepalive(r.ctx, r.Slog, defautlClient, r.LeaseTime)
	if err != nil {
		return err
	}

	r.leaseID = leaseID
	// 注册自己的节点信息
	nodeName := model.FullGateNode(r.NodeName())
	r.Debug().Msgf("gate.register.node:%s, host:%s\n", nodeName, addr)
	_, err = defautlClient.Put(r.ctx, nodeName, addr, clientv3.WithLease(leaseID))
	return err
}

func (r *Gate) delRuntimeNode(who model.Whoami) {
	runtimeNode := who.Name
	if len(runtimeNode) == 0 {
		return
	}

	nodeName := model.FullRuntimeNode(who)
	_, err := defautlClient.Delete(r.ctx, nodeName)
	if err != nil {
		r.Error().Msgf("gate.delete.runtime.node %s\n", err)
	}
}

// 注册runtime节点，并负责节点lease的续期
func (r *Gate) registerRuntimeWithKeepalive(who model.Whoami, keepalive chan bool) error {

	lease, leaseID, err := utils.NewLease(r.ctx, r.Slog, defautlClient, r.LeaseTime)
	if err != nil {
		r.Error().Msgf("registerRuntimeWithKeepalive.NewLease fail:%s\n", err)
		return err
	}
	// 注册runtime绑定的gate

	// 注册自己的节点信息
	nodeName := model.FullRuntimeNode(who)
	r.Info().Msgf("gate.register.runtime.node:%s, host:%s\n", nodeName, r.ServerAddr)
	info := model.RegisterRuntime{Whoami: who, Ip: r.ServerAddr}
	all, err := json.Marshal(&info)
	if err != nil {
		r.Error().Msgf("gate.register.runtime.node:%s, host:%s, marshal json fail:%s\n", nodeName, r.ServerAddr, err)
		return err
	}

	_, err = defautlClient.Put(r.ctx, nodeName, string(all), clientv3.WithLease(leaseID))

	if err != nil {
		r.Error().Msgf("gate.register.runtime.node %s\n", err)
		return err
	}

	for range keepalive {
		lease.KeepAliveOnce(r.ctx, leaseID)
	}

	return err
}
