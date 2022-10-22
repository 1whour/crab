package etcd

import (
	"context"
	"fmt"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 本子命令主要是为了测试脚本而写

type Etcd struct {
	GlobalTask bool `clop:"short;long" usage:"global task"`
	StateTask  bool `clop:"short;long" usage:"global state task"`

	EtcdAddr []string `clop:"short;long;greedy" usage:"etcd address" valid:"required"`
	TaskName string   `clop:"short;long" usage:"task name" valid:"required"`
	Get      bool     `clop:"long" usage:"get etcd value"`
	Debug    bool     `clop:"long" usage:"debug mode"`
}

var (
	defautlClient *clientv3.Client
	defaultKVC    clientv3.KV
)

func (e *Etcd) init() (err error) {

	if defautlClient, err = utils.NewEtcdClient(e.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	defaultKVC = clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	return nil
}

func (e *Etcd) SubMain() {
	if e.Debug {
		fmt.Println(e.EtcdAddr)
	}
	if err := e.init(); err != nil {
		fmt.Println(err)
		return
	}

	keyName := ""
	if e.GlobalTask {
		keyName = model.FullGlobalTask(e.TaskName)
	} else if e.StateTask {
		keyName = model.FullGlobalTaskState(e.TaskName)
	}

	if e.Get {

		rsp, err := defaultKVC.Get(context.TODO(), keyName)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(rsp.Kvs) > 0 {
			fmt.Printf("%s\n", rsp.Kvs[0].Value)
		}
	}
}
