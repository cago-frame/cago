package nsq

import (
	"context"
	"encoding/json"

	"github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/nsqio/go-nsq"
)

type Config struct {
	Addr          string
	NSQLookupAddr []string
}

type nsqBroker struct {
	config    *Config
	nsqConfig *nsq.Config
	producer  *nsq.Producer
	options   *broker.Options
}

func NewBroker(config Config, options ...broker.Option) (broker.Broker, error) {
	opts := &broker.Options{}
	for _, o := range options {
		o(opts)
	}
	nsqConfig := nsq.NewConfig()
	producer, err := nsq.NewProducer(config.Addr, nsqConfig)
	if err != nil {
		return nil, err
	}
	producer.SetLoggerLevel(nsq.LogLevelError)
	return &nsqBroker{
		config:    &config,
		nsqConfig: nsqConfig,
		producer:  producer,
		options:   opts,
	}, nil
}

func (b *nsqBroker) Publish(ctx context.Context, topic string, data *broker.Message, opt ...broker.PublishOption) error {
	bt, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return b.producer.Publish(topic, bt)
}

func (b *nsqBroker) Subscribe(ctx context.Context, topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	return newSubscribe(b, topic, h, broker.NewSubscribeOptions(opts...))
}

func (b *nsqBroker) Close() error {
	b.producer.Stop()
	return nil
}

func (b *nsqBroker) String() string {
	return "nsq"
}
