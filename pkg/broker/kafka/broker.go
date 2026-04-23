package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cago-frame/cago/pkg/broker/broker"
	kgo "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type kafkaBroker struct {
	config Config
	writer *kgo.Writer
	dialer *kgo.Dialer
}

// NewBroker 构造一个 Kafka broker。
// 失败会返回 error，上层组件会 panic（遵循 Core fail-fast 风格）。
func NewBroker(cfg Config) (broker.Broker, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka: Brokers must not be empty")
	}
	dialer, err := buildDialer(cfg)
	if err != nil {
		return nil, err
	}
	transport, err := buildTransport(cfg)
	if err != nil {
		return nil, err
	}
	w := &kgo.Writer{
		Addr:         kgo.TCP(cfg.Brokers...),
		Balancer:     &kgo.Hash{}, // 有 Key 时按 Key 分区；无 Key 时 kafka-go 内部 round-robin
		RequiredAcks: kgo.RequireAll,
		Async:        false,
		Transport:    transport,
	}
	return &kafkaBroker{
		config: cfg,
		writer: w,
		dialer: dialer,
	}, nil
}

func (b *kafkaBroker) Publish(ctx context.Context, topic string, data *broker.Message, opts ...broker.PublishOption) error {
	pubOpts := broker.NewPublishOptions(opts...)
	msg := buildKafkaMessage(topic, data, &pubOpts)
	return b.writer.WriteMessages(ctx, msg)
}

func (b *kafkaBroker) Subscribe(_ context.Context, topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	return newSubscribe(b, topic, h, broker.NewSubscribeOptions(opts...))
}

func (b *kafkaBroker) Close() error {
	return b.writer.Close()
}

func (b *kafkaBroker) String() string { return "kafka" }

// buildKafkaMessage 把 broker.Message 转成 kafka-go 的 Message。
func buildKafkaMessage(topic string, data *broker.Message, opts *broker.PublishOptions) kgo.Message {
	msg := kgo.Message{
		Topic: topic,
		Value: data.Body,
	}
	if key := getKey(opts); key != "" {
		msg.Key = []byte(key)
	}
	if len(data.Header) > 0 {
		msg.Headers = make([]kgo.Header, 0, len(data.Header))
		for k, v := range data.Header {
			msg.Headers = append(msg.Headers, kgo.Header{Key: k, Value: []byte(v)})
		}
	}
	return msg
}

// buildDialer 构造 Reader 用的 Dialer（SASL + TLS）。
func buildDialer(cfg Config) (*kgo.Dialer, error) {
	d := &kgo.Dialer{
		Timeout:  10 * time.Second,
		ClientID: cfg.ClientID,
	}
	if cfg.SASL != nil {
		m, err := buildSASL(cfg.SASL)
		if err != nil {
			return nil, err
		}
		d.SASLMechanism = m
	}
	if cfg.TLS != nil && cfg.TLS.Enable {
		tc, err := buildTLS(cfg.TLS)
		if err != nil {
			return nil, err
		}
		d.TLS = tc
	}
	return d, nil
}

// buildTransport 构造 Writer 用的 Transport（SASL + TLS）。
func buildTransport(cfg Config) (*kgo.Transport, error) {
	t := &kgo.Transport{
		ClientID:    cfg.ClientID,
		DialTimeout: 10 * time.Second,
	}
	if cfg.SASL != nil {
		m, err := buildSASL(cfg.SASL)
		if err != nil {
			return nil, err
		}
		t.SASL = m
	}
	if cfg.TLS != nil && cfg.TLS.Enable {
		tc, err := buildTLS(cfg.TLS)
		if err != nil {
			return nil, err
		}
		t.TLS = tc
	}
	return t, nil
}

func buildSASL(cfg *SASLConfig) (sasl.Mechanism, error) {
	switch cfg.Mechanism {
	case "PLAIN":
		return plain.Mechanism{Username: cfg.Username, Password: cfg.Password}, nil
	case "SCRAM-SHA-256":
		return scram.Mechanism(scram.SHA256, cfg.Username, cfg.Password)
	case "SCRAM-SHA-512":
		return scram.Mechanism(scram.SHA512, cfg.Username, cfg.Password)
	default:
		return nil, fmt.Errorf("kafka: unsupported SASL mechanism %q", cfg.Mechanism)
	}
}

func buildTLS(cfg *TLSConfig) (*tls.Config, error) {
	tc := &tls.Config{
		// InsecureSkipVerify 由用户显式开启
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}
	if cfg.CAFile != "" {
		caBytes, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("kafka: read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caBytes) {
			return nil, errors.New("kafka: failed to parse CA PEM")
		}
		tc.RootCAs = pool
	}
	if (cfg.CertFile == "") != (cfg.KeyFile == "") {
		return nil, errors.New("kafka: certFile and keyFile must be set together")
	}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("kafka: load cert/key: %w", err)
		}
		tc.Certificates = []tls.Certificate{cert}
	}
	return tc, nil
}
