package lambda

import (
	"github.com/1whour/crab/slog"
)

type Option func(o *Lambda)

// 设置endpoint，endpoint是router服务的地址
func WithEndpoint(endpoint string) Option {
	return func(o *Lambda) {
		o.Endpoint = append(o.Endpoint, endpoint)
	}
}

// 设置namespace, 如果需要有一个集团里面有所隔离，可以用namesapce区别
// 如果没有设置值就在default组里面
func WithNamespace(namespace string) Option {
	return func(o *Lambda) {
		o.Namespace = namespace
	}
}

// 设置slog
func WithSlog(l *slog.Slog) Option {
	return func(o *Lambda) {
		o.Slog = l
	}
}

// 设置runtimeName
func WithTaskName(name string) Option {
	return func(o *Lambda) {
		o.NodeName = name
	}
}

// 如果有多个服务，可以用groupID做隔离
/*
func WithGroupID(groupID string) Option {
	return func(o *options) {
		o.GroupId = groupID
	}
}
*/
