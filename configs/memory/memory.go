package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cago-frame/cago/configs/source"
)

type Memory struct {
	config map[string]interface{}
}

func NewSource(config map[string]interface{}) source.Source {
	if _, ok := config["debug"]; !ok {
		config["debug"] = true
	}
	if _, ok := config["env"]; !ok {
		config["env"] = "dev"
	}
	if _, ok := config["source"]; !ok {
		config["source"] = ""
	}
	return &Memory{
		config: config,
	}
}

func (e *Memory) Scan(ctx context.Context, key string, value interface{}) error {
	if v, ok := e.config[key]; ok {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, value)
	}
	return fmt.Errorf("memory %w: %s", source.ErrNotFound, key)
}

func (e *Memory) Has(ctx context.Context, key string) (bool, error) {
	_, ok := e.config[key]
	return ok, nil
}

func (e *Memory) Watch(ctx context.Context, key string, callback func(event source.Event)) error {
	return nil
}
