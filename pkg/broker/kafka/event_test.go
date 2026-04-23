package kafka

import (
	"testing"
	"time"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/stretchr/testify/assert"
)

func TestEvent_TopicAndMessage(t *testing.T) {
	e := &event{
		topic: "orders",
		msg:   &broker2.Message{Body: []byte("x")},
	}
	assert.Equal(t, "orders", e.Topic())
	assert.NotNil(t, e.Message())
	assert.Equal(t, []byte("x"), e.Message().Body)
}

func TestEvent_Ack_SetsIsAct(t *testing.T) {
	e := &event{}
	assert.False(t, e.isAct)
	assert.Nil(t, e.Ack())
	assert.True(t, e.isAct)
}

func TestEvent_Requeue_Unsupported(t *testing.T) {
	e := &event{}
	err := e.Requeue(time.Second)
	assert.ErrorIs(t, err, ErrRequeueUnsupported)
	assert.False(t, e.isAct, "Requeue 不应改变 isAct")
}

func TestEvent_Attempted(t *testing.T) {
	e := &event{attempted: 3}
	assert.Equal(t, 3, e.Attempted())
}

func TestEvent_Error(t *testing.T) {
	e := &event{}
	assert.Nil(t, e.Error())
}
