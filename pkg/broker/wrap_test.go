package broker

import (
	"context"
	"testing"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	wrap2 "github.com/cago-frame/cago/pkg/utils/wrap"
	"github.com/stretchr/testify/assert"
)

type emptyBroker struct {
	broker2.Broker
}

type emptyEvent struct {
	broker2.Event
}

func (e *emptyEvent) Message() *broker2.Message {
	return &broker2.Message{
		Body: []byte("test-data"),
	}
}

func (e *emptyBroker) Publish(ctx context.Context, topic string,
	data *broker2.Message, opts ...broker2.PublishOption) error {
	return nil
}

func (e *emptyBroker) Subscribe(ctx context.Context, topic string,
	h broker2.Handler, opts ...broker2.SubscribeOption) (broker2.Subscriber, error) {
	_ = h(ctx, &emptyEvent{})
	return nil, nil
}

func Test_newWrap(t *testing.T) {
	w := wrap2.New()
	var (
		publishCallCount   int
		subscribeCallCount int
	)
	w.Wrap(func(ctx *wrap2.Context) {
		switch ctx.Name() {
		case "Publish":
			publishCallCount++
			assert.Equal(t, ctx.Args(0), "prefix.topic")
			assert.Equal(t, ctx.Args(1).(*broker2.Message).Body, []byte("test-data"))
		case "Subscribe":
			subscribeCallCount++
			assert.Equal(t, ctx.Args(0), "prefix.topic")
			assert.Equal(t, ctx.Args(1).(*emptyEvent).Message().Body, []byte("test-data"))
			assert.Equal(t, ctx.Args(2).(broker2.SubscribeOptions).Group, "group")
		}
	})
	b := newWrap(&emptyBroker{}, w, &Options{
		topicPrefix: "prefix",
	})
	_ = b.Publish(context.Background(), "topic", &broker2.Message{
		Body: []byte("test-data"),
	})
	_, _ = b.Subscribe(context.Background(), "topic", func(ctx context.Context, event broker2.Event) error {
		return nil
	}, broker2.Group("group"))

	assert.Equal(t, publishCallCount, 1)
	assert.Equal(t, subscribeCallCount, 1)
}
