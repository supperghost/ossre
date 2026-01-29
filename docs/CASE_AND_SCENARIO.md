# ossre 场景与案例开发指南

本文档旨在为 `ossre` 框架的开发者提供一份清晰的指南，说明如何新增诊断场景（Scenario）和案例（Case）。我们将以“CLI 输出格式变更为 JSON”为例，贯穿整个开发与使用流程。

## 1. 核心概念

为了保证诊断逻辑的模块化与可扩展性，框架定义了三个核心概念：插件、场景和案例。

-   **插件 (Plugin)**
    -   **定义**：插件是最高层级的诊断单元，对应一个具体的诊断领域，如 `kernel`（内核）、`net`（网络）、`io`（磁盘 I/O）等。
    -   **实现**：每个插件都是一个独立的 Go 包，需实现 `internal/core.Plugin` 接口。它由 CLI 的 `run --module=<name>` 命令直接调用。
    -   **职责**：一个插件内部可以包含一个或多个相关的诊断“场景”。

-   **场景 (Scenario)**
    -   **定义**：场景是面向一个“真实问题域”的诊断集合，聚焦于一个明确的主题。
    -   **命名与 ID**：建议使用层级化 ID 描述场景，格式为 `plugin_name.domain.topic`。例如：
        -   `kernel.net.baseline`：网络相关的内核参数基线检查场景。
        -   `net.tcp.connection_drop`：TCP 连接异常丢弃场景。
    -   **实现**：在代码中，一个场景通常对应插件内部的一个独立函数，该函数负责执行一系列具体的检查。

-   **案例 (Case)**
    -   **定义**：案例是场景内部最细粒度的检查项，对应一条具体的诊断发现 (`Finding`) 和相关的修复建议 (`Suggestion`)。
    -   **命名与 ID**：案例 ID 继承自场景 ID，并进一步细化，格式为 `scenario_id.type.target`。例如：
        -   `kernel.net.baseline.sysctl.net_ipv4_tcp_max_tw_buckets`：检查 `tcp_max_tw_buckets` 内核参数是否合规的案例。
        -   `kernel.limit.baseline.ulimit.nofile`：检查最大文件句柄数 `ulimit -n` 是否达标的案例。
    -   **实现**：在场景函数内部，一个案例通常是一段独立的逻辑，用于采集数据、进行判断，并在不满足条件时生成 `Finding` 和 `Suggestion`。

通过这种层级化的设计，我们可以确保每一个诊断点都具备清晰的归属和唯一的标识，便于追踪、管理和自动化集成。

## 2. 开发步骤

以下是在框架中新增一个完整诊断场景（以`kernel`插件为例）的标准步骤。

### 步骤 1：选择或创建插件

根据诊断逻辑的领域，选择一个最合适的现有插件（如 `kernel`, `net`, `io`, `system`）。如果无合适插件，可在 `internal/plugins/` 目录下创建新的插件包，并在 `cmd/ossre/main.go` 的 `newRunner` 函数中完成注册。

### 步骤 2：在插件中新增场景函数

在选定的插件包内（例如 `internal/plugins/kernel/kernel.go`），为新场景创建一个独立的函数。该函数应遵循以下约定：

-   **函数签名**：`func runMyNewScenario() ([]models.Finding, []models.Suggestion)`
-   **职责**：函数内部封装该场景的所有采集、判断逻辑，并返回其产生的所有 `Finding` 和 `Suggestion`。

```go
// in: internal/plugins/kernel/kernel.go

// runNetSysctlBaselineScenario 实现“网络相关内核参数基线检查”场景。
func runNetSysctlBaselineScenario() ([]models.Finding, []models.Suggestion) {
    const scenarioID = "kernel.net.baseline"

    var findings []models.Finding
    var suggestions []models.Suggestion

    // ... 此处实现具体的案例检查逻辑 ...

    return findings, suggestions
}
```

### 步骤 3：在场景函数中实现案例

在场景函数内部，为每一个具体案例编写检查逻辑。

1.  **定义案例 ID**:
    ```go
    findingID := fmt.Sprintf("%s.sysctl.%s", scenarioID, "net_ipv4_tcp_max_tw_buckets")
    ```

2.  **执行检查并生成 Finding**: 如果检查不通过，则创建一个 `Finding` 对象。
    ```go
    findings = append(findings, models.Finding{
        ID:          findingID,
        Title:       "内核参数 net.ipv4.tcp_max_tw_buckets 不符合推荐值",
        Description: "当前值为 '...', 推荐值为 '50000'。此参数过小可能导致 'Time wait bucket table overflow'。",
        Severity:    models.SeverityWarning,
        Impact:      "在高并发短连接场景下，系统可能因 TIME_WAIT 连接耗尽而无法建立新连接。",
    })
    ```

3.  **提供修复建议 Suggestion**:
    ```go
    suggestions = append(suggestions, models.Suggestion{
        FindingID: findingID, // 关键：将建议与发现关联
        Title:     "将 net.ipv4.tcp_max_tw_buckets 调整为 50000",
        Details:   "临时生效：\n  sysctl -w net.ipv4.tcp_max_tw_buckets=50000\n\n持久化配置：\n  在 /etc/sysctl.conf 中添加或修改 'net.ipv4.tcp_max_tw_buckets = 50000'，然后执行 'sysctl -p'。",
    })
    ```

### 步骤 4：在插件 `Run` 方法中汇总结果

最后，修改插件主入口 `Run` 方法，调用所有场景函数，并将它们的结果统一汇总到最终的 `models.Result` 中。

```go
// in: internal/plugins/kernel/kernel.go

func (p *Plugin) Run(ctx context.Context) (models.Result, error) {
    _ = ctx

    var allFindings []models.Finding
    var allSuggestions []models.Suggestion

    // 调用网络基线场景
    f1, s1 := runNetSysctlBaselineScenario()
    allFindings = append(allFindings, f1...)
    allSuggestions = append(allSuggestions, s1...)

    // 调用其他场景...
    // f2, s2 := runAnotherScenario()
    // allFindings = append(allFindings, f2...)
    // allSuggestions = append(allSuggestions, s2...)

    return models.Result{
        Plugin:      PluginName,
        Findings:    allFindings,
        Suggestions: allSuggestions,
    }, nil
}
```

## 3. 运行与输出

`ossre` 的 CLI 提供了统一的运行和输出机制。

### CLI 使用

通过 `run` 子命令并指定 `--module` 参数来执行诊断。

```bash
# 运行内核诊断插件
./ossre run --module=kernel
```

### JSON 输出格式

`run` 命令默认输出格式为 JSON。这便于上层自动化系统解析，也方便开发者直接查看结构化的诊断结果。

**最小化示例输出**：
当一个插件（如 `kernel`）的占位实现运行时，它会返回一个包含空列表的 `Result` 对象。

```bash
$ ./ossre run --module=kernel
{
  "Plugin": "kernel",
  "Findings": [],
  "Suggestions": []
}
```

**包含诊断结果的示例**：
当 `kernel` 插件检测到不合规的内核参数时，输出将包含具体的 `Finding` 和 `Suggestion`。

```json
{
  "Plugin": "kernel",
  "Findings": [
    {
      "ID": "kernel.net.baseline.sysctl.net_ipv4_tcp_max_tw_buckets",
      "Title": "内核参数 net.ipv4.tcp_max_tw_buckets 不符合推荐值",
      "Description": "当前值为 '4096'，推荐值为 '50000'。此参数过小可能导致 'Time wait bucket table overflow'。",
      "Severity": "warning",
      "Impact": "在高并发短连接场景下，系统可能因 TIME_WAIT 连接耗尽而无法建立新连接。"
    }
  ],
  "Suggestions": [
    {
      "FindingID": "kernel.net.baseline.sysctl.net_ipv4_tcp_max_tw_buckets",
      "Title": "将 net.ipv4.tcp_max_tw_buckets 调整为 50000",
      "Details": "临时生效：\n  sysctl -w net.ipv4.tcp_max_tw_buckets=50000\n\n持久化配置：\n  在 /etc/sysctl.conf 中添加或修改 'net.ipv4.tcp_max_tw_buckets = 50000'，然后执行 'sysctl -p'。"
    }
  ]
}
```

## 4. 示例与最佳实践

### 保证空切片序列化为 `[]`

在 Go 中，未初始化的 `slice` 默认是 `nil`，序列化为 JSON 时会变成 `null`。为了保证输出的一致性，我们期望空结果是 `[]`。

**CLI 层已统一处理**：
开发者无需在插件中关心此问题。`cmd/ossre/main.go` 的 `handleRun` 函数在序列化之前，会检查 `Findings` 和 `Suggestions` 是否为 `nil`，如果是，则将其替换为空切片。

```go
// in: cmd/ossre/main.go
if result.Findings == nil {
    result.Findings = []models.Finding{}
}
if result.Suggestions == nil {
    result.Suggestions = []models.Suggestion{}
}
data, err := json.MarshalIndent(result, "", "  ")
// ...
```

### 编写高质量的建议详情 (Details)

`Suggestion.Details` 字段是直接面向运维人员或开发者的，其质量决定了问题能否被快速解决。

-   **提供可直接执行的命令**：给出临时修复和持久化修复的完整命令。
-   **解释命令的作用**：简要说明命令为什么能解决问题。
-   **注明生效方式**：明确指出是立即生效、重启服务生效还是需要重新登录会话。
-   **使用 Markdown 格式**：`Details` 字段支持多行文本和 Markdown 语法，可以利用代码块、列表等增强可读性。

## 5. 后续扩展建议

为进一步提升框架的健壮性和易用性，建议开发者在贡献场景和案例的同时，考虑以下方向：

-   **沉淀通用采集器**：将原子化的数据采集逻辑（如读取 procfs、sysfs、执行命令）抽象到 `internal/collectors` 包中，供不同插件复用。这能让插件逻辑更聚焦于“诊断”而非“采集”。
-   **编写测试用例**：为新增的场景和案例编写单元测试（Unit Test）或集成测试（Integration Test），确保其逻辑的正确性，并防止未来重构时引入回归问题。测试代码应放在项目根目录的 `tests/` 下。
-   **丰富插件类型**：根据实际需求，可以引入更多维度的插件，例如针对特定应用（如 Redis、MySQL）的诊断插件。


## 7. 模块 maxproc (进程线程创建余量)

`maxproc` 模块是一个与 `kernel` 模块并列的独立诊断插件，专门用于评估指定进程（PID）的线程创建余量。它将原 `kernel.thread.headroom` 场景独立出来，便于开发者在不关心其他内核参数时，快速、专门地进行线程余量诊断。

### 7.1 用途与运行方式

- **用途**：评估单个进程在 `nproc`、`cgroup pids`、`kernel.threads-max`、虚拟内存/栈四个维度下的线程创建能力，找出首个瓶颈。
- **运行方式**：
  ```bash
  # 评估指定 PID 的线程创建余量
  ./ossre run --module=maxproc --pid=<PID>
  
  # 不指定 PID 时，默认评估 ossre 自身进程
  ./ossre run --module=maxproc
  ```
- **与 `kernel.thread.headroom` 的关系**：
  - `maxproc` 模块的实现逻辑与 `kernel` 插件中的 `thread.headroom` 场景完全相同。
  - 它提供了一个独立的、更轻量级的入口，方便用户仅关注线程余量问题，而无需运行 `kernel` 模块中的所有其他场景。
  - `maxproc` 模块的 Finding ID 为 `maxproc.thread.headroom`，以区别于 `kernel` 插件。

### 7.2 平台兼容性

- **Linux**：
  - `maxproc` 模块在 Linux 上功能完整，能够通过 `/proc` 和 `/sys/fs/cgroup` 采集所有必要数据，并给出精确的余量估算和阻断因素。
  - Linux 实现位于 `internal/plugins/maxproc/maxproc_linux.go`，并带有 `//go:build linux` 标签。
- **非 Linux 平台**：
  - 在 macOS、Windows 等非 Linux 操作系统上，由于缺少 `/proc` 和 `cgroup` 的兼容接口，`maxproc` 模块会优雅地降级。
  - 它会返回一条 `Severity=info` 级别的 `Finding`，明确告知“当前操作系统不支持该场景”，而不会产生 `Suggestion`。
  - 降级实现位于 `internal/plugins/maxproc/maxproc_others.go`，并带有 `//go:build !linux` 标签。

这种设计保证了 `ossre` 在所有平台都能顺利编译和运行，同时为不同环境提供了清晰的能力边界说明。
