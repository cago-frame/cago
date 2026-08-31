package redis_stream

import "time"

// Config Redis Stream broker 配置。
//
// 客户端有两种来源：
//   - 配置 Addr，由 broker 自己创建 *redis.Client；
//   - 不配置 Addr，改为通过 SetClient 注入一个已有的 redis 客户端
//     （例如复用 database/redis 的全局实例）。
type Config struct {
	// Addr redis 地址，如 127.0.0.1:6379。为空时必须先调用 SetClient 注入客户端
	Addr string `yaml:"addr"`
	// Password redis 密码
	Password string `yaml:"password"`
	// DB redis 库序号
	DB int `yaml:"db"`
	// MaxLen stream 的最大长度，采用近似裁剪（XADD MAXLEN ~）。
	// Redis Stream 没有 kafka 那样的自动 retention，不设置会无限增长。
	// <=0 表示不裁剪。
	MaxLen int64 `yaml:"maxLen"`
	// Count 单次 XREADGROUP 最多拉取的消息条数，默认 16
	Count int64 `yaml:"count"`
	// Block XREADGROUP 的阻塞等待时长，默认 5s
	Block time.Duration `yaml:"block"`
	// ClaimMinIdle 消息在 pending 列表中空闲多久后可被重新投递，默认 30s。
	// 消费者崩溃或 handler 失败未 Ack 的消息靠它恢复。
	ClaimMinIdle time.Duration `yaml:"claimMinIdle"`
	// ClaimInterval 后台 XAUTOCLAIM 的扫描间隔，默认 10s
	ClaimInterval time.Duration `yaml:"claimInterval"`
}

const (
	defaultCount         = int64(16)
	defaultBlock         = 5 * time.Second
	defaultClaimMinIdle  = 30 * time.Second
	defaultClaimInterval = 10 * time.Second
)

// withDefaults 补齐零值配置项，避免出现 Block=0 这种“永久阻塞”的危险默认。
func (c Config) withDefaults() Config {
	if c.Count <= 0 {
		c.Count = defaultCount
	}
	if c.Block <= 0 {
		c.Block = defaultBlock
	}
	if c.ClaimMinIdle <= 0 {
		c.ClaimMinIdle = defaultClaimMinIdle
	}
	if c.ClaimInterval <= 0 {
		c.ClaimInterval = defaultClaimInterval
	}
	return c
}
