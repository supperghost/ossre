package system

import (
	"context"

	"github.com/supperghost/ossre/internal/core"
	"github.com/supperghost/ossre/pkg/models"
)

// PluginName 是系统通用诊断插件的名称常量。
const PluginName = "system"

// Plugin 实现了 core.Plugin 接口，用于执行系统通用诊断。
type Plugin struct{}

// New 创建一个新的系统诊断插件实例。
func New() core.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Description() string {
	return "系统通用资源与健康状态诊断（占位实现）"
}

func (p *Plugin) Run(ctx context.Context) (models.Result, error) {
	_ = ctx
	// TODO: 实现对 CPU、内存、负载等系统指标的采集与诊断。
	return models.Result{
		Plugin: PluginName,
	}, nil
}
