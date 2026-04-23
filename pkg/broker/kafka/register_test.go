package kafka

import (
	"testing"

	"github.com/cago-frame/cago/pkg/broker"
	"github.com/stretchr/testify/assert"
)

func TestKafkaFactoryRegistered(t *testing.T) {
	// init() 应已经把 "kafka" 工厂注册进去
	assert.NotNil(t, broker.GetFactory("kafka"))
}
