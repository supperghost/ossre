package net

import (
	"context"

	"github.com/supperghost/ossre/internal/core"
	"github.com/supperghost/ossre/pkg/models"
)

// PluginName 是网络诊断插件的名称常量。
const PluginName = "net"

// Plugin 实现了 core.Plugin 接口，用于执行网络相关诊断。
type Plugin struct{}

// New 创建一个新的网络诊断插件实例。
func New() core.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Description() string {
	return "网络连通性与性能诊断（占位实现）"
}

func (p *Plugin) Run(ctx context.Context) (models.Result, error) {
	_ = ctx
	// TODO: 实现对网络连通性、带宽、丢包等指标的采集与诊断。
	return models.Result{
		Plugin: PluginName,
	}, nil
}
