//go:build !linux
// +build !linux

package maxproc

import (
	"context"

	"code.byted.org/volcengine-support/shibin-code/ossre/go/pkg/models"
)

// runMaxprocScenario 在非 Linux 平台上提供降级实现。
// 该模块依赖 Linux 的 /proc 与 cgroup 语义，这里仅返回一条信息级别的 Finding，说明场景不适用。
func runMaxprocScenario(ctx context.Context) ([]models.Finding, []models.Suggestion) {
	_ = ctx

	finding := models.Finding{
		ID:          "maxproc.thread.headroom",
		Title:       "进程线程创建余量场景当前操作系统不支持",
		Description: "maxproc 模块依赖 Linux 的 /proc 与 cgroup 接口，仅在 Linux 上可用；当前操作系统不支持，无法评估可创建线程数与首个阻断因素。",
		Severity:    models.SeverityInfo,
		Impact:      "仅影响 maxproc 模块的线程创建余量诊断，其他插件与场景不受影响。",
	}

	return []models.Finding{finding}, nil
}
