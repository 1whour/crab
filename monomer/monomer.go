package monomer

import (
	"os"
	"sync"
	"time"

	"github.com/1whour/crab/gate"
	"github.com/1whour/crab/mjobs"
	"github.com/1whour/crab/runtime"
	"github.com/1whour/crab/slog"

	"github.com/antlabs/deepcopy"
)

type Monomer struct {
	// runtime
	EtcdAddr     []string      `clop:"short;long;greedy" usage:"etcd address"`
	Endpoint     []string      `clop:"long" usage:"endpoint address"`
	Level        string        `clop:"short;long" usage:"log level" default:"error"`
	WriteTimeout time.Duration `clop:"short;long" usage:"Timeout when writing messages" default:"3s"`

	// gate
	ServerAddr   string        `clop:"short;long" usage:"server address"`
	AutoFindAddr bool          `clop:"short;long" usage:"Automatically find unused ip:port, Only takes effect when ServerAddr is empty"`
	Name         string        `clop:"short;long" usage:"The name of the gate. If it is not filled, the default is uuid"`
	LeaseTime    time.Duration `clop:"long" usage:"lease time" default:"7s"`
	WriteTime    time.Duration `clop:"long" usage:"write timeout" default:"4s"`
	DSN          string        `clop:"--dsn" usage:"database dsn" valid:"requried"`

	// mjobs的字段是runtime和gate字段的一部分
	// ....

	// 日志对象
	*slog.Slog
}

func (m *Monomer) SubMain() {

	m.Slog = slog.New(os.Stdout).SetLevel(m.Level)
	var r runtime.Runtime
	err := deepcopy.Copy(&r, m).Do()
	if err != nil {
		m.Error().Msgf("deepcopy.runtime fail:%s", err)
		return
	}

	var g gate.Gate
	err = deepcopy.Copy(&g, m).Do()
	if err != nil {
		m.Error().Msgf("deepcopy.gate fail:%s", err)
		return
	}

	var mj mjobs.Mjobs
	err = deepcopy.Copy(&mj, m).Do()
	if err != nil {
		mj.Error().Msgf("deepcopy.gate fail:%s", err)
		return
	}

	r.Slog, g.Slog, mj.Slog = m.Slog, m.Slog, m.Slog
	var wg sync.WaitGroup
	wg.Add(3)
	defer wg.Wait()

	go func() {
		r.Debug().Msgf("start runtime")
		r.SubMain()
	}()
	go func() {
		g.Debug().Msgf("start gate")
		g.SubMain()
	}()

	mj.Debug().Msgf("start mjobs")
	mj.SubMain()
}
