package broker

import (
	"context"
	"testing"

	"github.com/cago-frame/cago/configs"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/stretchr/testify/assert"
)

type fakeBroker struct{ broker2.Broker }

func TestRegisterAndGetFactory(t *testing.T) {
	called := false
	RegisterBroker("test-fake", func(ctx context.Context, cfg *configs.Config) (broker2.Broker, error) {
		called = true
		return &fakeBroker{}, nil
	})
	f := GetFactory("test-fake")
	assert.NotNil(t, f)
	b, err := f(context.Background(), nil)
	assert.Nil(t, err)
	assert.NotNil(t, b)
	assert.True(t, called)

	assert.Nil(t, GetFactory("not-registered"))
}
