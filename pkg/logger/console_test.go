package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewConsole(t *testing.T) {
	var output bytes.Buffer
	log := NewConsole(&output, "info")

	log.Debug("hidden")
	log.Info("generated", zap.String("file", "user.go"))

	require.Equal(t, "INFO\tgenerated\t{\"file\": \"user.go\"}\n", output.String())
}
