package kernel

import (
	"context"

	"code.byted.org/volcengine-support/shibin-code/ossre/go/internal/core"
	"code.byted.org/volcengine-support/shibin-code/ossre/go/pkg/models"
)

// PluginName 是内核诊断插件的名称常量。
const PluginName = "kernel"

// Plugin 实现了 core.Plugin 接口，用于执行内核相关诊断。
type Plugin struct{}

// New 创建一个新的内核诊断插件实例。
func New() core.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Description() string {
	return "内核参数与内核状态诊断（占位实现）"
}

func (p *Plugin) Run(ctx context.Context) (models.Result, error) {
	_ = ctx
	// TODO: 实现具体的内核参数采集与诊断逻辑。
	return models.Result{
		Plugin: PluginName,
		// 当前仅返回空结果，后续补充 Finding 与 Suggestion。
	}, nil
}
