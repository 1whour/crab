package executer

import (
	"context"

	"github.com/1whour/ktuo/model"
)

// 留空，下个版本实现
func init() {
	Register("grpc", createGRPCExecuter)
}

type grpcExecuter struct {
}

func (s *grpcExecuter) Stop() error {
	return nil
}

func (s *grpcExecuter) Run() error {
	return nil
}

func createGRPCExecuter(ctx context.Context, param *model.Param) Executer {
	return nil
}
