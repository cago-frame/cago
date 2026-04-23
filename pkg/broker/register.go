package broker

import (
	"context"

	"github.com/cago-frame/cago/configs"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

// Factory 根据 configs 构建具体 broker 实例。由各 broker 子包在 init() 中注册。
type Factory func(ctx context.Context, config *configs.Config) (broker2.Broker, error)

var factories = map[string]Factory{}

// RegisterBroker 注册一个 broker 工厂。重复注册会覆盖。
// 通常在 broker 子包的 init() 中调用。
func RegisterBroker(name string, f Factory) {
	factories[name] = f
}

// GetFactory 获取已注册的 broker 工厂；未注册返回 nil。
func GetFactory(name string) Factory {
	return factories[name]
}
