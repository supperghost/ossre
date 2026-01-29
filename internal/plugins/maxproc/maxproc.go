package maxproc

import (
	"context"

	"github.com/supperghost/ossre/internal/core"
	"github.com/supperghost/ossre/pkg/models"
)

// PluginName 是 maxproc 诊断插件的名称常量。
const PluginName = "maxproc"

// Plugin 实现了 core.Plugin 接口，用于执行进程线程创建余量诊断。
type Plugin struct{}

// New 创建一个新的 maxproc 诊断插件实例。
func New() core.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Description() string {
	return "进程线程创建余量诊断（Linux 专属，其他平台降级提示）"
}

// Run 调用平台相关的场景实现，返回诊断结果。
func (p *Plugin) Run(ctx context.Context) (models.Result, error) {
	findings, suggestions := runMaxprocScenario(ctx)

	return models.Result{
		Plugin:      PluginName,
		Findings:    findings,
		Suggestions: suggestions,
	}, nil
}
