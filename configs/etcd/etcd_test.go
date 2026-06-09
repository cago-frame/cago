package etcd

import (
	"testing"

	"github.com/cago-frame/cago/configs/file"
	"github.com/stretchr/testify/assert"
)

// 文件配置源使用 yaml.v3(file.Yaml())解码 config.yaml。etcd 配置块为扁平结构,
// 必须能绑定到 Config。回归点:内嵌的 dbetcd.Config 仅用 `mapstructure:",squash"`
// 摊平,而 yaml.v3 不认该 tag,导致 Endpoints/Username 解析为空,clientv3 启动即报
// "etcdclient: no available endpoints"。
func TestConfig_BindsFlatEtcdBlock(t *testing.T) {
	raw := []byte("endpoints:\n  - 127.0.0.1:2379\nusername: root\nprefix: /config\n")

	var cfg Config
	err := file.Yaml().Unmarshal(raw, &cfg)
	assert.NoError(t, err)

	assert.Equal(t, []string{"127.0.0.1:2379"}, cfg.Endpoints)
	assert.Equal(t, "root", cfg.Username)
	assert.Equal(t, "/config", cfg.Prefix)
}
