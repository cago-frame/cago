package queue

import (
	"context"

	"github.com/cago-frame/cago/examples/simple/internal/task/queue/message"

	"github.com/cago-frame/cago/pkg/broker"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

const (
	ExampleTopic = "example" // 示例消息队列topic
)

// PublishExample 发布示例消息
func PublishExample(ctx context.Context, msg *message.ExampleMsg) error {
	return broker.Default().Publish(ctx, ExampleTopic, &broker2.Message{
		Body: msg.Marshal(),
	})
}

// SubscribeExample 订阅示例消息
func SubscribeExample(ctx context.Context, fn func(ctx context.Context, msg *message.ExampleMsg) error) error {
	_, err := broker.Default().Subscribe(ctx, ExampleTopic, func(ctx context.Context, event broker2.Event) error {
		msg := &message.ExampleMsg{}
		if err := msg.Unmarshal(event.Message().Body); err != nil {
			return err
		}
		return fn(ctx, msg)
	}, broker2.Retry())
	return err
}
