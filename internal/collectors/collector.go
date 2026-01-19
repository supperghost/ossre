package collectors

import "context"

// TODO: 本包负责从 /proc、/sys 以及其他系统接口中采集原始数据，供诊断插件复用。
// 这里仅提供占位定义，后续可拆分为多个源文件（如 procfs.go、sysfs.go 等）。

// Sample is a占位结构体，用于表示一次采集到的原始样本。
type Sample struct {
	// TODO: 根据实际需求扩展字段，例如 Key、Value、Timestamp 等。
	Source string
	Data   string
}

// Collector 是采集器接口的占位定义。
type Collector interface {
	// Collect 执行一次采集并返回零个或多个样本。
	Collect(ctx context.Context) ([]Sample, error)
}
