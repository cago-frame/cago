package kafka

import (
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

// Config Kafka broker 配置。
type Config struct {
	// Brokers kafka 集群的 bootstrap server 列表，必填，如 ["kafka1:9092", "kafka2:9092"]
	Brokers []string `yaml:"brokers"`
	// ClientID 客户端标识，可选
	ClientID string `yaml:"clientID"`
	// SASL 认证配置，nil 表示无认证
	SASL *SASLConfig `yaml:"sasl"`
	// TLS TLS 配置，nil 表示明文连接
	TLS *TLSConfig `yaml:"tls"`
}

// SASLConfig SASL 认证配置。Mechanism 可选 "PLAIN" / "SCRAM-SHA-256" / "SCRAM-SHA-512"。
type SASLConfig struct {
	Mechanism string `yaml:"mechanism"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

// TLSConfig TLS 配置。
type TLSConfig struct {
	Enable             bool   `yaml:"enable"`
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	CAFile             string `yaml:"caFile"`
	CertFile           string `yaml:"certFile"`
	KeyFile            string `yaml:"keyFile"`
}

// keyOptionKey 是存入 PublishOptions.Values 的 map key 类型，
// 未导出确保跨包不冲突。
type keyOptionKey struct{}

// WithKey 设置本次 Publish 的 Kafka Partition Key。
// 相同 key 的消息会路由到同一 partition，保证分区内顺序。
// 其他 broker 后端会忽略此选项。
func WithKey(key string) broker2.PublishOption {
	return func(o *broker2.PublishOptions) {
		if o.Values == nil {
			o.Values = make(map[any]any)
		}
		o.Values[keyOptionKey{}] = key
	}
}

// getKey 从 PublishOptions 中取出 key，没设置则返回空串。
func getKey(o *broker2.PublishOptions) string {
	if o == nil || o.Values == nil {
		return ""
	}
	if v, ok := o.Values[keyOptionKey{}].(string); ok {
		return v
	}
	return ""
}
