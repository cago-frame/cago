package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNSQFactoryRegisteredByDefault(t *testing.T) {
	// 主包 init() 应在导入时自动注册 nsq（类似 db 默认注册 mysql）
	assert.NotNil(t, GetFactory("nsq"), "nsq 工厂应为默认注册，无需用户 import _")
}
