package mongo

import (
	"context"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

type Config struct {
	URI      string `yaml:"uri"`
	Database string `json:"database"`
}

var defaultClient *Client

func Mongo(ctx context.Context, config *configs.Config) error {
	cfg := &Config{}
	if err := config.Scan(ctx, "mongo", cfg); err != nil {
		return err
	}
	opts := options.Client()
	if trace.Default() != nil {
		opts.Monitor = otelmongo.NewMonitor(otelmongo.WithTracerProvider(trace.Default()))
	}
	opts.ApplyURI(cfg.URI)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return err
	}
	defaultClient = &Client{client: client, database: cfg.Database}
	return nil
}

func Default() *Client {
	return defaultClient
}

func Ctx(ctx context.Context) *CtxMongoDatabase {
	return defaultClient.Database(ctx)
}
