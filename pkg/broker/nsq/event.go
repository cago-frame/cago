package nsq

import (
	"time"

	"github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/nsqio/go-nsq"
)

type event struct {
	topic   string
	data    *broker.Message
	message *nsq.Message
	// isAct 是否已经执行过Ack或者Requeue
	isAct     bool
	attempted int
}

func (e *event) Topic() string {
	return e.topic
}

func (e *event) Message() *broker.Message {
	return e.data
}

func (e *event) Ack() error {
	e.message.Finish()
	e.isAct = true
	return nil
}

func (e *event) Error() error {
	return nil
}

func (e *event) Requeue(delay time.Duration) error {
	e.message.Requeue(delay)
	e.isAct = true
	return nil
}

func (e *event) Attempted() int {
	return e.attempted
}
