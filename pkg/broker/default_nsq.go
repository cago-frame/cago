package broker

import (
	"context"

	"github.com/cago-frame/cago/configs"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/cago-frame/cago/pkg/broker/nsq"
)

// nsq 作为默认 broker 内联注册（类似 database/db 默认注册 mysql）。
// 用户导入主 broker 包即自动可用，无需额外 import _。
//
// 其他 broker（event_bus、kafka）需要用户显式 import _，例如：
//
//	import _ "github.com/cago-frame/cago/pkg/broker/event_bus"
//	import _ "github.com/cago-frame/cago/pkg/broker/kafka"
func init() {
	RegisterBroker("nsq", func(ctx context.Context, cfg *configs.Config) (broker2.Broker, error) {
		c := &nsq.Config{}
		if err := cfg.Scan(ctx, "broker.nsq", c); err != nil {
			return nil, err
		}
		return nsq.NewBroker(*c)
	})
}
