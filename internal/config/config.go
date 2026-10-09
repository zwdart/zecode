package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port                 int    `toml:"port"`
	StaticDir            string `toml:"static_dir"`
	UseEmbeddedStatic    bool   `toml:"use_embedded_static"`
	MaxUploadSizeMB      int    `toml:"max_upload_size_mb"`
	DecodeTimeoutSeconds int    `toml:"decode_timeout_seconds"`
	MaxQueryLength       int    `toml:"max_query_length"`
	QRSize               int    `toml:"qr_size"`
}

// Default 返回内置默认配置
func Default() *Config {
	return &Config{
		Port:                 1880,
		StaticDir:            "./statics",
		UseEmbeddedStatic:    true,
		MaxUploadSizeMB:      2,
		DecodeTimeoutSeconds: 5,
		MaxQueryLength:       2048,
		QRSize:               256,
	}
}

// Load 从指定路径加载 TOML 配置；文件不存在时使用默认值。
func Load(path string) (*Config, error) {
	cfg := Default()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("[警告] 未找到配置文件 %s，将使用内置默认值\n", path)
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
	}
	return cfg, nil
}

// MaxUploadSize 返回字节数
func (c *Config) MaxUploadSize() int64 {
	return int64(c.MaxUploadSizeMB) * 1024 * 1024
}
