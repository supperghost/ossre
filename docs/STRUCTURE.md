
# Go 诊断框架结构规划

本文档定义了 `ossre/go` 模块的目录结构和核心组件职责，旨在为 Linux 环境提供一个可扩展的诊断框架。

## 目录结构

```
shibin-code/ossre/go/
├── Makefile                # [可选] 构建、测试和代码检查的快捷命令
├── go.mod                  # Go 模块定义
├── go.sum                  # Go 模块依赖项 checksum
├── cmd/
│   └── ossre/              # CLI 入口
│       └── main.go         # 程序主函数
├── internal/
│   ├── core/               # 核心编排与执行引擎
│   │   ├── executor.go     # TODO: 定义执行流程、插件加载和调度逻辑
│   │   └── interfaces.go   # TODO: 定义插件必须实现的接口
│   ├── plugins/            # 独立的诊断插件集合
│   │   ├── io/             # I/O 相关诊断插件
│   │   │   └── io.go       # TODO: 实现 I/O 诊断逻辑
│   │   ├── kernel/         # 内核参数相关诊断插件
│   │   │   └── kernel.go   # TODO: 实现内核诊断逻辑
│   │   ├── net/            # 网络相关诊断插件
│   │   │   └── net.go      # TODO: 实现网络诊断逻辑
│   │   └── system/         # 操作系统通用诊断插件
│   │       └── system.go   # TODO: 实现系统诊断逻辑
│   └── collectors/         # 原子化的信息采集器
│       └── procfs.go       # TODO: 从 /proc, /sys 等收集信息的函数
├── pkg/
│   ├── config/             # 配置解析
│   │   └── config.go       # TODO: 定义配置加载逻辑 (YAML/TOML)
│   └── models/             # 共享的数据模型
│       ├── finding.go      # TODO: 定义诊断发现、结果和建议的数据结构
│       └── types.go        # TODO: 定义通用基础类型
├── configs/                # 示例配置文件
│   └── config.example.yaml # TODO: 提供一个基础的配置模板
├── docs/                   # 项目文档
│   ├── README.md           # TODO: 项目概览和快速入门指南
│   └── STRUCTURE.md        # (本文档) 结构规划
└── tests/                  # 测试代码
    └── core_test.go        # TODO: 核心功能的单元测试骨架
```

## 组件职责

- **`cmd/ossre`**:
  - **职责**: 命令行接口 (CLI) 的入口。
  - **功能**: 解析命令行参数，加载配置，并启动核心执行引擎。

- **`internal/core`**:
  - **职责**: 框架的核心调度与编排引擎。
  - **功能**: 负责插件的加载、初始化、执行和结果汇总。定义插件必须遵循的统一接口 (`interface`)。

- **`internal/plugins/*`**:
  - **职责**: 实现具体的诊断逻辑。
  - **功能**: 每个子包 (`kernel`, `io`, `net`, `system`) 关注一个特定的诊断领域。插件以模块化形式存在，可独立开发和测试。

- **`internal/collectors`**:
  - **职责**: 提供原子化的信息采集能力。
  - **功能**: 从系统（如 `/proc`, `/sys`）安全地读取原始数据，供插件使用。此模块不包含诊断逻辑。

- **`pkg/models`**:
  - **职责**: 定义整个项目共享的数据结构。
  - **功能**: 提供标准化的诊断结果、发现 (`Finding`) 和修复建议 (`Suggestion`) 的数据类型，确保各组件间数据交换的一致性。

- **`pkg/config`**:
  - **职责**: 配置加载与解析。
  - **功能**: 读取并解析配置文件（如 YAML, TOML），为框架提供运行时参数。

- **`configs/`**:
  - **职责**: 提供开箱即用的示例配置文件。
  - **功能**: 帮助用户快速上手。

- **`docs/`**:
  - **职责**: 存放项目文档。
  - **功能**: 提供高层设计、快速入门和 API 参考。

- **`tests/`**:
  - **职责**: 存放测试代码。
  - **功能**: 包含单元测试、集成测试，确保代码质量和稳定性。
