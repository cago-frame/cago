package redis_stream

import (
	"errors"
	"time"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

// ErrRequeueUnsupported Redis Stream 没有原生的延迟重投递能力，
// 无法为 Requeue(delay) 提供符合语义的实现。
// 需要重试请使用 SubscribeOption 的 Retry：失败不 Ack，消息留在 pending 列表，
// 由后台 XAUTOCLAIM 在 ClaimMinIdle 之后重新投递。
var ErrRequeueUnsupported = errors.New("redis_stream: Requeue is unsupported; use the Retry option instead")

// event 把一条 stream 消息适配成 broker2.Event。
type event struct {
	topic     string
	id        string
	msg       *broker2.Message
	attempted int
	// isAct 是否已经显式 Ack（影响 subscriber 的自动确认判定）
	isAct bool
}

func (e *event) Topic() string             { return e.topic }
func (e *event) Message() *broker2.Message { return e.msg }
func (e *event) Attempted() int            { return e.attempted }
func (e *event) Error() error              { return nil }

// Ack 标记这条消息已处理，真正的 XACK 由 subscriber 循环统一执行。
func (e *event) Ack() error {
	e.isAct = true
	return nil
}

// Requeue Redis Stream 不支持延迟重投递，始终返回 ErrRequeueUnsupported。
func (e *event) Requeue(_ time.Duration) error {
	return ErrRequeueUnsupported
}
