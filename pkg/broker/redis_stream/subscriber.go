package redis_stream

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/cago-frame/cago/pkg/gogo"
	"github.com/cago-frame/cago/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type subscriber struct {
	topic     string
	group     string
	consumers []string
	client    redis.UniversalClient
	config    Config
	handler   broker.Handler
	options   broker.SubscribeOptions
	cancel    context.CancelFunc
	done      sync.WaitGroup
}

func newSubscribe(ctx context.Context, b *redisStreamBroker, topic string,
	handler broker.Handler, options broker.SubscribeOptions) (broker.Subscriber, error) {
	if options.Group == "" {
		return nil, errors.New("redis_stream: Subscribe requires a non-empty Group (consumer group name)")
	}
	// MKSTREAM 保证 stream 还不存在时也能建组，"0" 表示从最早的消息开始消费
	if err := b.client.XGroupCreateMkStream(ctx, topic, options.Group, "0").Err(); err != nil &&
		!isBusyGroup(err) {
		return nil, fmt.Errorf("redis_stream: create consumer group: %w", err)
	}
	concurrent := max(options.Concurrent, 1)

	runCtx, cancel := context.WithCancel(context.Background())
	sub := &subscriber{
		topic:     topic,
		group:     options.Group,
		consumers: make([]string, 0, concurrent),
		client:    b.client,
		config:    b.config,
		handler:   handler,
		options:   options,
		cancel:    cancel,
	}
	host, _ := os.Hostname()
	for i := range concurrent {
		consumer := fmt.Sprintf("%s-%d-%d", host, os.Getpid(), i)
		sub.consumers = append(sub.consumers, consumer)
		sub.done.Add(1)
		gogo.Go(func() error {
			defer sub.done.Done()
			sub.runConsumer(runCtx, consumer)
			return nil
		})
	}
	// Redis 不会自动重投 pending 消息，需要单独一个循环把超时未确认的消息认领回来
	sub.done.Add(1)
	gogo.Go(func() error {
		defer sub.done.Done()
		sub.runClaimer(runCtx, sub.consumers[0])
		return nil
	})
	return sub, nil
}

// runConsumer 单个消费者的拉取循环，直到 ctx 被取消。
func (s *subscriber) runConsumer(ctx context.Context, consumer string) {
	log := s.logger().With(zap.String("consumer", consumer))
	for {
		if ctx.Err() != nil {
			return
		}
		streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    s.group,
			Consumer: consumer,
			Streams:  []string{s.topic, ">"},
			Count:    s.config.Count,
			Block:    s.config.Block,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return // 正常关闭
			}
			// Block 超时没有新消息，属于正常情况
			if errors.Is(err, redis.Nil) {
				continue
			}
			log.Error("redis stream read group error", zap.Error(err))
			// 避免 redis 不可用时空转打满 CPU
			s.sleep(ctx, s.config.Block)
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				// 新投递的消息，投递次数固定为 1；重投的次数由 claimer 查询后填充
				s.process(ctx, &msg, 1)
			}
		}
	}
}

// runClaimer 周期性地把 pending 列表中空闲超过 ClaimMinIdle 的消息认领回来重新投递。
// 它同时覆盖两种情况：handler 失败未 Ack 的重试，以及消费者崩溃后的消息接管。
func (s *subscriber) runClaimer(ctx context.Context, consumer string) {
	ticker := time.NewTicker(s.config.ClaimInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.claimOnce(ctx, consumer)
		}
	}
}

// claimOnce 执行一轮完整的 XAUTOCLAIM 扫描（分页直到游标回到起点）。
func (s *subscriber) claimOnce(ctx context.Context, consumer string) {
	log := s.logger().With(zap.String("consumer", consumer))
	start := "0-0"
	for {
		if ctx.Err() != nil {
			return
		}
		messages, next, err := s.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   s.topic,
			Group:    s.group,
			Consumer: consumer,
			MinIdle:  s.config.ClaimMinIdle,
			Start:    start,
			Count:    s.config.Count,
		}).Result()
		if err != nil {
			if ctx.Err() == nil {
				log.Error("redis stream auto claim error", zap.Error(err))
			}
			return
		}
		if len(messages) > 0 {
			attempts := s.pendingAttempts(ctx, consumer)
			for _, msg := range messages {
				attempted, ok := attempts[msg.ID]
				if !ok {
					attempted = 1
				}
				s.process(ctx, &msg, attempted)
			}
		}
		// 游标回到 0-0 表示已经扫完一轮
		if next == "" || next == "0-0" {
			return
		}
		start = next
	}
}

// pendingAttempts 查询当前 consumer 名下 pending 消息的投递次数，
// 用于给 Event.Attempted() 提供真实值。查询失败时返回空 map，由调用方回退到 1。
func (s *subscriber) pendingAttempts(ctx context.Context, consumer string) map[string]int {
	pending, err := s.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   s.topic,
		Group:    s.group,
		Start:    "-",
		End:      "+",
		Count:    s.config.Count,
		Consumer: consumer,
	}).Result()
	if err != nil {
		return nil
	}
	attempts := make(map[string]int, len(pending))
	for _, p := range pending {
		attempts[p.ID] = int(p.RetryCount)
	}
	return attempts
}

// process 调用业务 handler，并按 AutoAck / Retry 语义决定是否 XACK。
func (s *subscriber) process(ctx context.Context, msg *redis.XMessage, attempted int) {
	ev := &event{
		topic:     s.topic,
		id:        msg.ID,
		msg:       decodeMessage(msg),
		attempted: attempted,
	}
	callCtx := s.options.Context
	if callCtx == nil {
		callCtx = context.Background()
	}
	handleErr := s.handler(callCtx, ev)

	if !decideAck(handleErr, s.options, ev.isAct) {
		if handleErr != nil {
			s.logger().Warn("redis stream skip ack, message stays pending for redelivery",
				zap.String("id", msg.ID), zap.Bool("retry", s.options.Retry), zap.Error(handleErr))
		}
		return
	}
	if err := s.client.XAck(ctx, s.topic, s.group, msg.ID).Err(); err != nil && ctx.Err() == nil {
		s.logger().Error("redis stream ack error", zap.String("id", msg.ID), zap.Error(err))
	}
}

// decideAck 根据 handler 返回错误、Ack/Retry 选项、以及是否显式 Ack，
// 决定是否 XACK 这条消息。未 XACK 的消息会留在 pending 列表，
// 由 claimer 在 ClaimMinIdle 之后重新投递。
//
// 映射表：
//
//	AutoAck=true  + 成功               → ack
//	AutoAck=true  + 失败 + Retry=false → 仍然 ack（丢弃，避免 pending 无限堆积）
//	AutoAck=true  + 失败 + Retry=true  → 不 ack（等待 claimer 重投）
//	AutoAck=false + 已 Ack             → ack
//	AutoAck=false + 未 Ack             → 不 ack
func decideAck(handleErr error, options broker.SubscribeOptions, isAct bool) bool {
	if !options.AutoAck {
		return isAct
	}
	if handleErr == nil {
		return true
	}
	return !options.Retry
}

func (s *subscriber) sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (s *subscriber) logger() *zap.Logger {
	return logger.Default().With(zap.String("topic", s.topic), zap.String("group", s.group))
}

func (s *subscriber) Topic() string { return s.topic }

func (s *subscriber) Unsubscribe() error {
	s.cancel()
	s.done.Wait()
	// 清理消费者，避免 redis 中残留大量已下线的 consumer 记录
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var firstErr error
	for _, consumer := range s.consumers {
		if err := s.client.XGroupDelConsumer(ctx, s.topic, s.group, consumer).Err(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// isBusyGroup 判断错误是否为“消费组已存在”，这是可以忽略的。
func isBusyGroup(err error) bool {
	return strings.Contains(err.Error(), "BUSYGROUP")
}
