package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// 配置文件里写的驼峰键要能落到 LogFileConfig 上；yaml.v3 对没 tag 的字段只认全小写键。
func TestLogFileConfigYAML(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, yaml.Unmarshal([]byte(`
logFile:
  enable: true
  filename: ./runtime/logs/app.log
  maxSize: 50
  maxBackups: -1
  maxAge: 7
  compress: true
`), cfg))

	assert.Equal(t, LogFileConfig{
		Enable:     true,
		Filename:   "./runtime/logs/app.log",
		MaxSize:    50,
		MaxBackups: -1,
		MaxAge:     7,
		Compress:   true,
	}, cfg.LogFile)
}

func TestNewRotateWriter(t *testing.T) {
	t.Run("不配轮转参数与改动前一致", func(t *testing.T) {
		w := newRotateWriter("a.log", LogFileConfig{})
		assert.Equal(t, "a.log", w.Filename)
		assert.Equal(t, 2, w.MaxSize)
		assert.Equal(t, 10, w.MaxBackups)
		assert.Equal(t, 30, w.MaxAge)
		assert.True(t, w.LocalTime)
		assert.False(t, w.Compress)
	})

	t.Run("配了就按配置", func(t *testing.T) {
		w := newRotateWriter("a.log", LogFileConfig{MaxSize: 50, MaxBackups: 60, MaxAge: 7, Compress: true})
		assert.Equal(t, 50, w.MaxSize)
		assert.Equal(t, 60, w.MaxBackups)
		assert.Equal(t, 7, w.MaxAge)
		assert.True(t, w.Compress)
	})

	// lumberjack 里 0 表示不删；零值已经留给「用默认值」，所以用负数表达不限。
	t.Run("负数表示不按个数或天数清理", func(t *testing.T) {
		w := newRotateWriter("a.log", LogFileConfig{MaxBackups: -1, MaxAge: -1})
		assert.Equal(t, 0, w.MaxBackups)
		assert.Equal(t, 0, w.MaxAge)
	})
}
