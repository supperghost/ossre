package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"code.byted.org/volcengine-support/shibin-code/ossre/go/internal/core"
	"code.byted.org/volcengine-support/shibin-code/ossre/go/internal/plugins/io"
	"code.byted.org/volcengine-support/shibin-code/ossre/go/internal/plugins/kernel"
	"code.byted.org/volcengine-support/shibin-code/ossre/go/internal/plugins/net"
	"code.byted.org/volcengine-support/shibin-code/ossre/go/internal/plugins/system"
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
	_ = fs.Parse(args)

	if *module == "" {
		fmt.Fprintln(os.Stderr, "必须通过 --module 指定诊断模块名称")
		fs.Usage()
		os.Exit(1)
	}

	r := newRunner()
	ctx := context.Background()
	result, err := r.Run(ctx, *module)
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行模块 %s 失败: %v\n", *module, err)
		os.Exit(1)
	}

	// 当前仅输出占位信息，后续可扩展为结构化输出或 JSON
	fmt.Printf("模块 %s 运行完成。诊断结果条目数: %d\n", result.Plugin, len(result.Findings))
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

示例:
  %s list
  %s run --module=kernel
  %s version
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
