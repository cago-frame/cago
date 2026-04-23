package kafka

import (
	"testing"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/stretchr/testify/assert"
)

func TestWithKey_StoresInValues(t *testing.T) {
	opts := broker2.PublishOptions{}
	WithKey("user-123")(&opts)
	assert.Equal(t, "user-123", getKey(&opts))
}

func TestWithKey_OverridesPrevious(t *testing.T) {
	opts := broker2.PublishOptions{}
	WithKey("a")(&opts)
	WithKey("b")(&opts)
	assert.Equal(t, "b", getKey(&opts))
}

func TestGetKey_EmptyOptions(t *testing.T) {
	opts := broker2.PublishOptions{}
	assert.Equal(t, "", getKey(&opts))
}

func TestGetKey_WrongType(t *testing.T) {
	opts := broker2.PublishOptions{Values: map[any]any{keyOptionKey{}: 123}}
	assert.Equal(t, "", getKey(&opts))
}
