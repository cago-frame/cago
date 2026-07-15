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
	dialer := &net.Dialer{Timeout: time.Second * 4}
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: config.Address,
		Username:  config.Username,
		Password:  config.Password,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			DialContext:     dialer.DialContext,
		},
	})
	if err != nil {
		return err
	}
	es = client
	return nil
}

var es *elasticsearch.Client

func Ctx(ctx context.Context) *elasticsearch.Client {
	return es
}
