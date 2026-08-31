package redis_stream

import (
	"errors"
	"testing"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestEncodeValues(t *testing.T) {
	values := encodeValues(&broker2.Message{
		Header: map[string]string{"traceparent": "abc"},
		Body:   []byte("hello"),
	})
	assert.Equal(t, "hello", values[fieldBody])
	assert.Equal(t, "abc", values[headerPrefix+"traceparent"])
	assert.Len(t, values, 2)
}

func TestEncodeValues_NoHeader(t *testing.T) {
	values := encodeValues(&broker2.Message{Body: []byte("x")})
	assert.Equal(t, map[string]interface{}{fieldBody: "x"}, values)
}

func TestDecodeMessage(t *testing.T) {
	msg := decodeMessage(&redis.XMessage{
		ID: "1-0",
		Values: map[string]interface{}{
			fieldBody:                 "hello",
			headerPrefix + "trace-id": "abc",
			// 非本 broker 写入的字段应被忽略，不污染 Header
			"foreign": "ignored",
		},
	})
	assert.Equal(t, []byte("hello"), msg.Body)
	assert.Equal(t, map[string]string{"trace-id": "abc"}, msg.Header)
}

func TestDecodeMessage_EmptyBody(t *testing.T) {
	msg := decodeMessage(&redis.XMessage{ID: "1-0", Values: map[string]interface{}{}})
	assert.Empty(t, msg.Body)
	assert.NotNil(t, msg.Header)
}

// 编解码必须可往返，否则 trace 透传会在 header 上丢信息
func TestEncodeDecode_RoundTrip(t *testing.T) {
	in := &broker2.Message{
		Header: map[string]string{"a": "1", "b": "2"},
		Body:   []byte("body"),
	}
	out := decodeMessage(&redis.XMessage{ID: "1-0", Values: encodeValues(in)})
	assert.Equal(t, in.Header, out.Header)
	assert.Equal(t, in.Body, out.Body)
}

func TestNewBroker_NoAddrNoClient(t *testing.T) {
	_, err := NewBroker(Config{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "addr")
}

func TestNewBrokerWithClient_NilClient(t *testing.T) {
	_, err := NewBrokerWithClient(nil, Config{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "client")
}

var errTest = errors.New("test")

func TestDecideAck(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		autoAck bool
		retry   bool
		isAct   bool
		want    bool
	}{
		{"自动确认 + 成功", nil, true, false, false, true},
		{"自动确认 + 失败 + 不重试", errTest, true, false, false, true},
		{"自动确认 + 失败 + 重试", errTest, true, true, false, false},
		{"手动确认 + 已 Ack", nil, false, false, true, true},
		{"手动确认 + 未 Ack", nil, false, false, false, false},
		{"手动确认 + 失败 + 已 Ack", errTest, false, true, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decideAck(c.err, broker2.SubscribeOptions{AutoAck: c.autoAck, Retry: c.retry}, c.isAct)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestEvent_RequeueUnsupported(t *testing.T) {
	e := &event{}
	assert.ErrorIs(t, e.Requeue(0), ErrRequeueUnsupported)
}
