package event_bus

import (
	"time"

	"github.com/cago-frame/cago/pkg/broker/broker"
)

type event struct {
	topic string
	data  *broker.Message
}

func (e *event) Topic() string {
	return e.topic
}

func (e *event) Message() *broker.Message {
	return e.data
}

func (e *event) Ack() error {
	return nil
}

func (e *event) Error() error {
	return nil
}

func (e *event) Requeue(delay time.Duration) error {
	return nil
}

func (e *event) Attempted() int {
	return -1
}
