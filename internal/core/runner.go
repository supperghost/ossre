package core

import (
	"context"
	"fmt"

	"github.com/supperghost/ossre/pkg/models"
)

// Runner 负责插件注册、列出和按名称运行。
// TODO: 后续可在此注入上下文信息（如主机信息）和配置对象。
type Runner struct {
	plugins map[string]Plugin
}

// NewRunner 使用给定的插件集合创建一个新的 Runner。
func NewRunner(plugins []Plugin) *Runner {
	m := make(map[string]Plugin, len(plugins))
	for _, p := range plugins {
		if p == nil {
			continue
		}
		name := p.Name()
		if name == "" {
			continue
		}
		m[name] = p
	}
	return &Runner{plugins: m}
}

// ListPlugins 返回已注册的插件列表，遍历顺序未定义。
func (r *Runner) ListPlugins() []Plugin {
	result := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		result = append(result, p)
	}
	return result
}

// Run 根据名称运行指定插件。
func (r *Runner) Run(ctx context.Context, name string) (models.Result, error) {
	p, ok := r.plugins[name]
	if !ok {
		return models.Result{}, fmt.Errorf("unknown plugin: %s", name)
	}
	// TODO: 统一的前后钩子、日志、超时控制等
	return p.Run(ctx)
}
