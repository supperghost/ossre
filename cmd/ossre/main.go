package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/supperghost/ossre/internal/core"
	"github.com/supperghost/ossre/internal/plugins/io"
	"github.com/supperghost/ossre/internal/plugins/kernel"
	"github.com/supperghost/ossre/internal/plugins/maxproc"
	"github.com/supperghost/ossre/internal/plugins/net"
	"github.com/supperghost/ossre/internal/plugins/system"
	"github.com/supperghost/ossre/pkg/models"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "list":
		handleList()
	case "run":
		handleRun(os.Args[2:])
	case "version":
		handleVersion()
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func newRunner() *core.Runner {
	// TODO: 后续可从配置中动态选择启用的插件
	plugins := []core.Plugin{
		kernel.New(),
		maxproc.New(),
		io.New(),
		net.New(),
		system.New(),
	}
	return core.NewRunner(plugins)
}

func handleList() {
	r := newRunner()
	plugins := r.ListPlugins()
	for _, p := range plugins {
		fmt.Printf("%s\t%s\n", p.Name(), p.Description())
	}
}

func handleRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	module := fs.String("module", "", "要运行的诊断模块名称")
	pid := fs.Int("pid", 0, "目标进程 PID，可选；不指定时默认使用自身 PID")
	format := fs.String("format", "json", "输出格式: json 或 plain")
	_ = fs.Parse(args)

	if *module == "" {
		fmt.Fprintln(os.Stderr, "必须通过 --module 指定诊断模块名称")
		fs.Usage()
		os.Exit(1)
	}

	r := newRunner()
	ctx := context.Background()
	if *pid > 0 {
		ctx = context.WithValue(ctx, "ossre.pid", *pid)
	}
	result, err := r.Run(ctx, *module)
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行模块 %s 失败: %v\n", *module, err)
		os.Exit(1)
	}

	// 确保空结果也序列化为 [] 而不是 null
	if result.Findings == nil {
		result.Findings = []models.Finding{}
	}
	if result.Suggestions == nil {
		result.Suggestions = []models.Suggestion{}
	}

	// 根据格式选择输出方式
	switch *format {
	case "plain":
		outputPlainText(result)
	default:
		// 默认输出JSON格式
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化模块 %s 结果为 JSON 失败: %v\n", *module, err)
			os.Exit(1)
		}
		_, _ = os.Stdout.Write(append(data, '\n'))
	}
}

// outputPlainText 以格式化文本方式输出诊断结果
func outputPlainText(result models.Result) {
	fmt.Printf("=== %s 诊断结果 ===\n\n", result.Plugin)

	if len(result.Findings) > 0 {
		fmt.Println("发现问题:")
		for i, finding := range result.Findings {
			fmt.Printf("\n%d. %s\n", i+1, finding.Title)
			fmt.Printf("   严重程度: %s\n", finding.Severity)
			fmt.Printf("   描述: %s\n", finding.Description)
			if finding.Impact != "" {
				fmt.Printf("   影响: %s\n", finding.Impact)
			}
		}
		fmt.Println()
	}

	if len(result.Suggestions) > 0 {
		fmt.Println("建议:")
		for i, suggestion := range result.Suggestions {
			fmt.Printf("\n%d. %s\n", i+1, suggestion.Title)
			fmt.Printf("   %s\n", suggestion.Details)
		}
		fmt.Println()
	}

	if len(result.Findings) == 0 && len(result.Suggestions) == 0 {
		fmt.Println("未发现问题。")
	}
}

func handleVersion() {
	fmt.Printf("ossre 诊断框架版本: %s\n", version)
}

func usage() {
	fmt.Fprintf(os.Stderr, `用法: %s <命令> [选项]

命令:
  list                列出可用诊断模块
  run --module=<name> 运行指定诊断模块
  version             显示版本信息

选项:
  --module=<name>     指定要运行的诊断模块名称
                      kernel 内核参数优化
					  maxproc 最大进程数诊断
					  io I/O 诊断
					  net 网络诊断
					  system 系统通用诊断
  --pid=<pid>         目标进程 PID，可选；不指定时默认使用自身 PID
  --format=<format>   输出格式，可选值: json (默认), plain (格式化文本)

示例:
  %s list
  %s run --module=kernel
  %s run --module=maxproc --pid=1 --format=plain
  %s run --module=kernel --format=plain
  %s version
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
