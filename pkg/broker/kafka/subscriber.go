package kafka

import (
	"context"
	"errors"
	"sync"

	"github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/cago-frame/cago/pkg/gogo"
	"github.com/cago-frame/cago/pkg/logger"
	kgo "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type subscriber struct {
	topic   string
	readers []*kgo.Reader
	cancel  context.CancelFunc
	done    sync.WaitGroup
}

func newSubscribe(b *kafkaBroker, topic string, handler broker.Handler, options broker.SubscribeOptions) (broker.Subscriber, error) {
	if options.Group == "" {
		return nil, errors.New("kafka: Subscribe requires a non-empty Group (consumer group id)")
	}
	concurrent := max(options.Concurrent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	sub := &subscriber{
		topic:   topic,
		readers: make([]*kgo.Reader, 0, concurrent),
		cancel:  cancel,
	}

	for range concurrent {
		r := kgo.NewReader(kgo.ReaderConfig{
			Brokers:        b.config.Brokers,
			GroupID:        options.Group,
			Topic:          topic,
			CommitInterval: 0, // 禁用后台自动 commit，完全由我们控制
			Dialer:         b.dialer,
		})
		sub.readers = append(sub.readers, r)
		sub.done.Add(1)
		gogo.Go(func() error {
			defer sub.done.Done()
			runReader(ctx, r, topic, handler, options)
			return nil
		})
	}
	return sub, nil
}

// runReader 单个 Reader 的拉取循环，直到 ctx 被取消。
func runReader(ctx context.Context, r *kgo.Reader, topic string, handler broker.Handler, options broker.SubscribeOptions) {
	log := logger.Default().With(zap.String("topic", topic), zap.String("group", options.Group))
	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // 正常关闭
			}
			log.Error("kafka fetch message error", zap.Error(err))
			continue
		}

		ev := &event{
			topic:     topic,
			msg:       convertMessage(&msg),
			attempted: 0, // kafka-go 不暴露单条消息的 attempted 次数
		}

		callCtx := options.Context
		if callCtx == nil {
			callCtx = context.Background()
		}
		handleErr := handler(callCtx, ev)

		shouldCommit := decideCommit(handleErr, options, ev.isAct)
		if shouldCommit {
			if err := r.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil {
				log.Error("kafka commit error", zap.Error(err))
			}
		} else if handleErr != nil {
			log.Warn("kafka skip commit due to retry or manual ack pending",
				zap.Bool("retry", options.Retry), zap.Error(handleErr))
		}
	}
}

// decideCommit 根据 handler 返回错误、Ack/Retry 选项、以及是否显式 Ack，
// 决定是否 commit 这条消息的 offset。
//
// 映射表：
//
//	AutoAck=true  + 成功              → commit
//	AutoAck=true  + 失败 + Retry=false → 仍然 commit（丢弃，避免阻塞分区）
//	AutoAck=true  + 失败 + Retry=true  → 不 commit（下次 poll 重投递，阻塞分区）
//	AutoAck=false + 已 Ack             → commit
//	AutoAck=false + 未 Ack             → 不 commit
func decideCommit(handleErr error, options broker.SubscribeOptions, isAct bool) bool {
	if !options.AutoAck {
		return isAct
	}
	if handleErr == nil {
		return true
	}
	return !options.Retry
}

func convertMessage(m *kgo.Message) *broker.Message {
	header := make(map[string]string, len(m.Headers))
	for _, h := range m.Headers {
		header[h.Key] = string(h.Value)
	}
	return &broker.Message{
		Header: header,
		Body:   m.Value,
	}
}

func (s *subscriber) Topic() string { return s.topic }

func (s *subscriber) Unsubscribe() error {
	s.cancel()
	var firstErr error
	for _, r := range s.readers {
		if err := r.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.done.Wait()
	return firstErr
}
