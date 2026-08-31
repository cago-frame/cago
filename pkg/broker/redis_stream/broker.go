package redis_stream

import (
	"context"
	"errors"
	"strings"

	"github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/redis/go-redis/v9"
)

const (
	// fieldBody stream 中承载消息体的字段名
	fieldBody = "body"
	// headerPrefix stream 中承载消息头的字段名前缀，
	// 加前缀是为了和 body 以及其他生产者写入的字段隔离
	headerPrefix = "h:"
)

// injectedClient 由 SetClient 注入的 redis 客户端，
// 供配置里没有填 Addr 的场景使用（复用已有连接）。
var injectedClient redis.UniversalClient

// SetClient 注入一个已有的 redis 客户端给 redis_stream broker 使用。
// 必须在 broker 组件启动之前调用，且此时 broker.redis_stream.addr 应留空。
// 注入的客户端生命周期由调用方负责，broker.Close() 不会关闭它。
func SetClient(client redis.UniversalClient) {
	injectedClient = client
}

type redisStreamBroker struct {
	config Config
	client redis.UniversalClient
	// ownClient 客户端是否由本 broker 创建，只有自建的才在 Close 时关闭
	ownClient bool
}

// NewBroker 根据配置构造一个 Redis Stream broker。
// Addr 为空时会退回到 SetClient 注入的客户端。
func NewBroker(cfg Config) (broker.Broker, error) {
	if cfg.Addr == "" {
		if injectedClient == nil {
			return nil, errors.New("redis_stream: addr must not be empty, or inject a client with SetClient")
		}
		return NewBrokerWithClient(injectedClient, cfg)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &redisStreamBroker{
		config:    cfg.withDefaults(),
		client:    client,
		ownClient: true,
	}, nil
}

// NewBrokerWithClient 使用调用方提供的 redis 客户端构造 broker，
// 客户端不会被 Close() 关闭。
func NewBrokerWithClient(client redis.UniversalClient, cfg Config) (broker.Broker, error) {
	if client == nil {
		return nil, errors.New("redis_stream: client must not be nil")
	}
	return &redisStreamBroker{
		config:    cfg.withDefaults(),
		client:    client,
		ownClient: false,
	}, nil
}

func (b *redisStreamBroker) Publish(ctx context.Context, topic string,
	data *broker.Message, _ ...broker.PublishOption) error {
	args := &redis.XAddArgs{
		Stream: topic,
		Values: encodeValues(data),
	}
	if b.config.MaxLen > 0 {
		args.MaxLen = b.config.MaxLen
		args.Approx = true
	}
	return b.client.XAdd(ctx, args).Err()
}

func (b *redisStreamBroker) Subscribe(ctx context.Context, topic string,
	h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	return newSubscribe(ctx, b, topic, h, broker.NewSubscribeOptions(opts...))
}

// Close 关闭 broker。只有自建的客户端才会被关闭，注入的客户端由调用方管理。
func (b *redisStreamBroker) Close() error {
	if !b.ownClient {
		return nil
	}
	return b.client.Close()
}

func (b *redisStreamBroker) String() string { return "redis_stream" }

// encodeValues 把 broker.Message 编码成 stream 的字段集合。
func encodeValues(data *broker.Message) map[string]interface{} {
	values := make(map[string]interface{}, len(data.Header)+1)
	values[fieldBody] = string(data.Body)
	for k, v := range data.Header {
		values[headerPrefix+k] = v
	}
	return values
}

// decodeMessage 把 stream 消息解码回 broker.Message，
// 无法识别的字段（其他生产者写入的）会被忽略。
func decodeMessage(m *redis.XMessage) *broker.Message {
	msg := &broker.Message{Header: make(map[string]string)}
	for k, v := range m.Values {
		s, ok := v.(string)
		if !ok {
			continue
		}
		switch {
		case k == fieldBody:
			msg.Body = []byte(s)
		case strings.HasPrefix(k, headerPrefix):
			msg.Header[strings.TrimPrefix(k, headerPrefix)] = s
		}
	}
	return msg
}
