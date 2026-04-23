package kafka

import (
	"errors"
	"testing"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/stretchr/testify/assert"
)

func TestBuildKafkaMessage_WithKey(t *testing.T) {
	opts := broker2.PublishOptions{}
	WithKey("user-1")(&opts)
	msg := buildKafkaMessage("orders", &broker2.Message{
		Header: map[string]string{"trace-id": "abc"},
		Body:   []byte("hello"),
	}, &opts)

	assert.Equal(t, "orders", msg.Topic)
	assert.Equal(t, []byte("user-1"), msg.Key)
	assert.Equal(t, []byte("hello"), msg.Value)
	assert.Len(t, msg.Headers, 1)
	assert.Equal(t, "trace-id", msg.Headers[0].Key)
	assert.Equal(t, []byte("abc"), msg.Headers[0].Value)
}

func TestBuildKafkaMessage_NoKey(t *testing.T) {
	opts := broker2.PublishOptions{}
	msg := buildKafkaMessage("orders", &broker2.Message{Body: []byte("x")}, &opts)
	assert.Nil(t, msg.Key)
	assert.Equal(t, []byte("x"), msg.Value)
	assert.Empty(t, msg.Headers)
}

func TestNewBroker_EmptyBrokers(t *testing.T) {
	_, err := NewBroker(Config{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Brokers")
}

func TestBuildSASL_UnknownMechanism(t *testing.T) {
	_, err := buildSASL(&SASLConfig{Mechanism: "UNKNOWN"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

var errTest = errors.New("test")

func TestDecideCommit(t *testing.T) {
	type tc struct {
		name    string
		err     error
		autoAck bool
		retry   bool
		isAct   bool
		want    bool
	}
	cases := []tc{
		{"auto ack + success", nil, true, false, false, true},
		{"auto ack + fail + no retry", errTest, true, false, false, true},
		{"auto ack + fail + retry", errTest, true, true, false, false},
		{"manual ack + acked", nil, false, false, true, true},
		{"manual ack + not acked", nil, false, false, false, false},
		{"manual ack + fail + acked", errTest, false, true, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decideCommit(c.err, broker2.SubscribeOptions{AutoAck: c.autoAck, Retry: c.retry}, c.isAct)
			assert.Equal(t, c.want, got)
		})
	}
}
