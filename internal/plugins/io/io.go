package io

import (
	"context"

	"github.com/supperghost/ossre/internal/core"
	"github.com/supperghost/ossre/pkg/models"
)

// PluginName 是 I/O 诊断插件的名称常量。
const PluginName = "io"

// Plugin 实现了 core.Plugin 接口，用于执行 I/O 相关诊断。
type Plugin struct{}

// New 创建一个新的 I/O 诊断插件实例。
func New() core.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Description() string {
	return "磁盘与文件系统 I/O 诊断（占位实现）"
}

func (p *Plugin) Run(ctx context.Context) (models.Result, error) {
	_ = ctx
	// TODO: 实现对磁盘、文件系统和 I/O 延迟的采集与诊断。
	return models.Result{
		Plugin: PluginName,
	}, nil
}
