package elasticsearch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/elastic/go-elasticsearch/v8"
)

type Config struct {
	Address            []string
	Username           string
	Password           string
	Cert               string
	InsecureSkipVerify bool `yaml:"insecureSkipVerify"`
}

func Elasticsearch(ctx context.Context, cfg *configs.Config) error {
	config := &Config{}
	if err := cfg.Scan(ctx, "elasticsearch", config); err != nil {
		return err
	}
	var tlsConfig *tls.Config
	if config.Cert != "" {
		ca, err := os.ReadFile(config.Cert)
		if err != nil {
			return err
		}
		certs := x509.NewCertPool()
		if ok := certs.AppendCertsFromPEM(ca); !ok {
			return err
		}
		tlsConfig = &tls.Config{
			RootCAs:            certs,
			InsecureSkipVerify: config.InsecureSkipVerify,
		}
	}
	client, err := elasticsearch.NewClient(clientConfig(config, tlsConfig, trace.Default()))
	if err != nil {
		return err
	}
	es = client
	return nil
}

// clientConfig 构建客户端配置；配置了链路追踪（tp 非 nil）时给每个请求产生 span。
// 请求要用 WithContext(ctx) 传入请求的 ctx，span 才会挂在调用方的链路下。
func clientConfig(config *Config, tlsConfig *tls.Config, tp oteltrace.TracerProvider) elasticsearch.Config {
	dialer := &net.Dialer{Timeout: time.Second * 4}
	esCfg := elasticsearch.Config{
		Addresses: config.Address,
		Username:  config.Username,
		Password:  config.Password,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			DialContext:     dialer.DialContext,
		},
	}
	if tp != nil {
		// 不记录查询体：搜索词可能含用户输入，且会让 span 变大
		esCfg.Instrumentation = elasticsearch.NewOpenTelemetryInstrumentation(tp, false)
	}
	return esCfg
}

var es *elasticsearch.Client

func Ctx(ctx context.Context) *elasticsearch.Client {
	return es
}
