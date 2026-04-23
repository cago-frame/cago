package broker

import (
	"context"
	"testing"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/configs/memory"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/stretchr/testify/assert"
)

func TestNewWithConfig_UnknownType(t *testing.T) {
	cfg, err := configs.NewConfig("test", configs.WithSource(
		memory.NewSource(map[string]interface{}{
			"broker": map[string]interface{}{
				"type": "nonexistent",
			},
		}),
	))
	assert.Nil(t, err)

	b, err := NewWithConfig(context.Background(), cfg)
	assert.Nil(t, b)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not registered")
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestNewWithConfig_RegisteredType(t *testing.T) {
	RegisterBroker("test-new-config", func(ctx context.Context, cfg *configs.Config) (broker2.Broker, error) {
		return &fakeBroker{}, nil
	})
	cfg, err := configs.NewConfig("test", configs.WithSource(
		memory.NewSource(map[string]interface{}{
			"broker": map[string]interface{}{
				"type": "test-new-config",
			},
		}),
	))
	assert.Nil(t, err)

	b, err := NewWithConfig(context.Background(), cfg)
	assert.Nil(t, err)
	assert.NotNil(t, b)
}
