package kafka

import (
	"errors"
	"time"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

// ErrRequeueUnsupported Kafka 语义下 Requeue 无优雅映射（无 per-message requeue）。
// 如需重试，请使用 SubscribeOption 的 Retry 语义，或在业务层走独立的 retry topic。
var ErrRequeueUnsupported = errors.New("kafka: Requeue is unsupported; use Retry option or a retry topic")

// event 把一条 kafka-go 消息适配成 broker2.Event。
type event struct {
	topic     string
	msg       *broker2.Message
	attempted int
	// isAct 是否已经显式 Ack（影响 subscriber 自动 commit 的判定）
	isAct bool
}

func (e *event) Topic() string             { return e.topic }
func (e *event) Message() *broker2.Message { return e.msg }
func (e *event) Attempted() int            { return e.attempted }
func (e *event) Error() error              { return nil }

// Ack 标记这条消息已处理。真正的 offset commit 由 subscriber 循环统一执行。
func (e *event) Ack() error {
	e.isAct = true
	return nil
}

// Requeue Kafka 不支持 per-message requeue，始终返回 ErrRequeueUnsupported。
func (e *event) Requeue(_ time.Duration) error {
	return ErrRequeueUnsupported
}
