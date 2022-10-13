package client

import (
	"github.com/gnh123/scheduler/slog"
)

type Option func(o *options)

type options struct {
	Endpoint  string `json:"endpoint"`
	Namespace string `json:"namespace"`
	GroupId   string `json:"group_id"`
	*slog.Slog
}

// 设置endpoint，endpoint是router服务的地址
func WithEndpoint(endpoint string) Option {
	return func(o *options) {
		o.Endpoint = endpoint
	}
}

// 设置namespace, 如果有多个环境，可以用namesapce做隔离
func WithNamespace(namespace string) Option {
	return func(o *options) {
		o.Namespace = namespace
	}
}

// 设置slog
func WithSlog(l *slog.Slog) Option {
	return func(o *options) {
		o.Slog = l
	}
}

// 如果有多个服务，可以用groupID做隔离
func WithGroupID(groupID string) Option {
	return func(o *options) {
		o.GroupId = groupID
	}
}
