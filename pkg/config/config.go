package config

import (
	"fmt"
	"io"
	"os"
)

// Config 表示框架的运行时配置。
// TODO: 根据实际需求将字段拆分为更细的结构体，并补充 YAML/TOML 标签。
type Config struct {
	// 原始文件路径，仅用于调试。
	Source string
	// 原始配置内容的占位字段，后续可替换为结构化字段。
	Raw []byte
}

// LoadFromFile 从给定路径加载配置文件。
// 当前实现仅读取原始字节，不解析 YAML/TOML，后续可在不引入第三方依赖的前提下补充解析逻辑。
func LoadFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	return &Config{
		Source: path,
		Raw:    data,
	}, nil
}

// NewDefault 返回一个占位的默认配置实例。
// TODO: 根据项目需求填充合理的默认值。
func NewDefault() *Config {
	return &Config{}
}
