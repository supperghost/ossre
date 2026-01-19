package core

import (
	"context"

	"code.byted.org/volcengine-support/shibin-code/ossre/go/pkg/models"
)

// Plugin 定义了所有诊断插件需要实现的最小接口。
type Plugin interface {
	// Name 返回插件的唯一名称，用于 CLI 与配置中引用。
	Name() string
	// Description 返回插件的简要说明，便于 list 命令展示。
	Description() string
	// Run 执行一次诊断。
	// TODO: 后续可以接受配置、日志接口等参数。
	Run(ctx context.Context) (models.Result, error)
}

// RunResult 表示单个插件执行后的结果，包含插件名称和诊断结果。
type RunResult struct {
	PluginName string
	Result     models.Result
}
