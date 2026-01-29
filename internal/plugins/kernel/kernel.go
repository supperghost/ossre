package kernel

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"code.byted.org/volcengine-support/shibin-code/ossre/go/internal/core"
	"code.byted.org/volcengine-support/shibin-code/ossre/go/pkg/models"
)

// PluginName 是内核诊断插件的名称常量。
const PluginName = "kernel"

// 根据不同操作系统定义正确的资源常量
var (
	// 进程数限制常量
	RLIMIT_PROCESS_COUNT int
)

type RLimit struct {
	Soft int64
	Hard int64
}

func GetRlimitNprocFromProc() (*RLimit, error) {
	f, err := os.Open("/proc/self/limits")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// 示例行：
		// Max processes            4096                 4096                 processes
		if strings.HasPrefix(line, "Max processes") {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				return nil, fmt.Errorf("unexpected format: %q", line)
			}

			parse := func(s string) (int64, error) {
				if s == "unlimited" {
					return -1, nil
				}
				return strconv.ParseInt(s, 10, 64)
			}

			soft, err := parse(fields[2])
			if err != nil {
				return nil, err
			}
			hard, err := parse(fields[3])
			if err != nil {
				return nil, err
			}

			return &RLimit{
				Soft: soft,
				Hard: hard,
			}, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("Max processes not found in /proc/self/limits")
}

// 初始化进程数限制常量
func init() {
	// 在Linux上使用RLIMIT_NPROC，在macOS上使用RLIMIT_MAXPROC
	// 这里使用运行时判断而不是编译时条件编译，以便在跨平台编译时更灵活
	if nr, err := GetRlimitNprocFromProc(); err == nil {
		RLIMIT_PROCESS_COUNT = int(nr.Soft)
	}
}

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
	// 这里明确说明目前主要实现的是“网络相关内核参数基线”场景
	return "内核参数与内核状态诊断（包含网络相关内核参数基线检查）"
}

// sysctlExpectation 表示对单个 sysctl 参数的期望值。
type sysctlExpectation struct {
	Key         string
	Expected    string
	Description string
}

// 对应原 Python 脚本 suggested_sysctl_params_basic
var netSysctlBaseline = []sysctlExpectation{
	{
		Key:         "net.ipv4.tcp_syncookies",
		Expected:    "1",
		Description: "开启 TCP SYN Cookies，在 SYN flood 等场景下避免连接队列被打满",
	},
	{
		Key:         "net.core.somaxconn",
		Expected:    "4096",
		Description: "控制 listen backlog 的上限，过小会导致高并发场景下丢连接",
	},
	{
		Key:         "net.netfilter.nf_conntrack_max",
		Expected:    "655350",
		Description: "连接跟踪表最大项数量，过小会导致 `nf_conntrack: table full, dropping packet`",
	},
	{
		Key:         "net.ipv4.tcp_max_syn_backlog",
		Expected:    "8192",
		Description: "半连接队列大小，过小会放大 SYN 攻击及瞬时峰值影响",
	},
	{
		Key:         "net.ipv4.ip_local_port_range",
		Expected:    "1024 65000",
		Description: "本地可用临时端口范围，过窄时易耗尽本地端口",
	},
	{
		Key:         "net.ipv4.tcp_max_tw_buckets",
		Expected:    "50000",
		Description: "TIME_WAIT 连接上限，过小会出现 Time wait bucket table overflow",
	},
	{
		Key:         "net.netfilter.nf_conntrack_tcp_timeout_established",
		Expected:    "1200",
		Description: "已建立连接的超时时间，过大可能导致连接表长时间占用资源",
	},
	{
		Key:         "net.ipv4.tcp_timestamps",
		Expected:    "1",
		Description: "TCP 时间戳，一般保持开启（配合 tcp_tw_recycle=0 使用）",
	},
	{
		Key:         "net.ipv4.tcp_tw_recycle",
		Expected:    "0",
		Description: "TCP TIME_WAIT 快速回收，开启在 NAT 场景下易导致连接异常，建议关闭",
	},
	{
		Key:         "net.ipv4.tcp_tw_reuse",
		Expected:    "1",
		Description: "允许重用 TIME_WAIT socket，有助于减少大量 TIME_WAIT 带来的端口压力",
	},
	{
		Key:         "net.ipv4.tcp_fin_timeout",
		Expected:    "30",
		Description: "FIN-WAIT2 超时时间，过大时会积累过多 FIN_WAIT2 连接",
	},
}

// Run 执行一次诊断。
// 这里我们实现一个“场景”：网络相关内核参数基线检查 + ulimit 基线检查。
func (p *Plugin) Run(ctx context.Context) (models.Result, error) {
	_ = ctx

	var (
		allFindings    []models.Finding
		allSuggestions []models.Suggestion
	)

	// 场景 1：网络相关内核参数基线
	f1, s1 := runNetSysctlBaselineScenario()
	allFindings = append(allFindings, f1...)
	allSuggestions = append(allSuggestions, s1...)

	// 场景 2：进程/文件句柄 ulimit 基线
	f2, s2 := runLimitBaselineScenario()
	allFindings = append(allFindings, f2...)
	allSuggestions = append(allSuggestions, s2...)

	return models.Result{
		Plugin:      PluginName,
		Findings:    allFindings,
		Suggestions: allSuggestions,
	}, nil
}

// runNetSysctlBaselineScenario 实现“网络相关内核参数基线检查”场景。
// 场景 ID 示例：kernel.net.baseline
func runNetSysctlBaselineScenario() ([]models.Finding, []models.Suggestion) {
	const scenarioID = "kernel.net.baseline"

	var (
		findings    []models.Finding
		suggestions []models.Suggestion
	)

	for _, item := range netSysctlBaseline {
		current, err := readSysctl(item.Key)
		if err != nil {
			// 无法读取时给出 warning，方便后续排查权限或环境问题
			id := fmt.Sprintf("%s.sysctl.%s.read_error", scenarioID, sanitizeID(item.Key))
			findings = append(findings, models.Finding{
				ID:          id,
				Title:       fmt.Sprintf("无法读取内核参数 %s", item.Key),
				Description: fmt.Sprintf("尝试从 /proc/sys 读取 %s 失败: %v", item.Key, err),
				Severity:    models.SeverityWarning,
				Impact:      "无法评估该参数是否符合网络基线，可能影响对网络异常的诊断准确性。",
			})
			continue
		}

		// 对 ip_local_port_range 这类带空格的值直接按字符串比较即可
		if strings.TrimSpace(current) == item.Expected {
			continue
		}

		findingID := fmt.Sprintf("%s.sysctl.%s", scenarioID, sanitizeID(item.Key))
		findings = append(findings, models.Finding{
			ID:    findingID,
			Title: fmt.Sprintf("内核参数 %s 不符合推荐值", item.Key),
			Description: fmt.Sprintf(
				"当前值为 %q，推荐值为 %q。该参数用于：%s。",
				current, item.Expected, item.Description,
			),
			Severity: models.SeverityWarning,
			Impact:   "在高并发或异常流量场景下，可能放大网络丢包、TIME_WAIT 过多或连接耗尽等问题。",
		})

		suggestions = append(suggestions, models.Suggestion{
			FindingID: findingID,
			Title:     fmt.Sprintf("将内核参数 %s 调整为推荐值 %s", item.Key, item.Expected),
			Details: fmt.Sprintf(
				"临时生效（重启失效）：\n  sysctl -w %s=%s\n"+
					"持久化配置（推荐）：\n  1. 编辑 /etc/sysctl.conf，确保存在如下配置行：\n     %s = %s\n  2. 执行 sysctl -p 使配置立即生效。\n",
				item.Key, item.Expected, item.Key, item.Expected,
			),
		})
	}

	return findings, suggestions
}

// runLimitBaselineScenario 实现“进程/文件句柄 ulimit 基线检查”场景。
// 对应原 Python 脚本中对 max open files / max user processes 的检查和优化。
func runLimitBaselineScenario() ([]models.Finding, []models.Suggestion) {
	const (
		scenarioID        = "kernel.limit.baseline"
		targetMaxOpenFile = uint64(655350)
		targetMaxProc     = uint64(655350)
	)

	var (
		findings    []models.Finding
		suggestions []models.Suggestion
	)

	// 检查 RLIMIT_NOFILE（相当于 ulimit -n）
	if rl, err := getRlimit(syscall.RLIMIT_NOFILE); err == nil {
		if rl.Cur < targetMaxOpenFile {
			id := scenarioID + ".ulimit.nofile"
			findings = append(findings, models.Finding{
				ID:    id,
				Title: "进程最大文件句柄数 (RLIMIT_NOFILE) 低于推荐值",
				Description: fmt.Sprintf(
					"当前进程软限制为 %d，推荐不小于 %d。过低时在高并发网络/IO 场景下容易触发 'too many open files'。",
					rl.Cur, targetMaxOpenFile,
				),
				Severity: models.SeverityWarning,
				Impact:   "可能导致服务在峰值流量下无法建立足够多的网络连接或打开文件句柄。",
			})
			suggestions = append(suggestions, models.Suggestion{
				FindingID: id,
				Title:     "提升最大文件句柄数到 655350",
				Details: "建议：\n" +
					"1. 临时调整（当前 shell 会话）：\n" +
					"   ulimit -SHn 655350\n\n" +
					"2. 持久化配置（/etc/security/limits.conf）：\n" +
					"   root soft nofile 655350\n" +
					"   root hard nofile 655350\n" +
					"   *    soft nofile 655350\n" +
					"   *    hard nofile 655350\n" +
					"修改后需要重新登录或重启对应服务进程生效。",
			})
		}
	}

	// 检查进程数限制（相当于 ulimit -u）
	if rl, err := getRlimit(RLIMIT_PROCESS_COUNT); err == nil {
		if rl.Cur < targetMaxProc {
			id := scenarioID + ".ulimit.nproc"
			findings = append(findings, models.Finding{
				ID:    id,
				Title: "进程最大数 (RLIMIT_PROCESS_COUNT) 低于推荐值",
				Description: fmt.Sprintf(
					"当前进程软限制为 %d，推荐不小于 %d。过低时在多进程/多线程场景下容易触发 'resource temporarily unavailable' 等错误。",
					rl.Cur, targetMaxProc,
				),
				Severity: models.SeverityWarning,
				Impact:   "可能限制业务水平扩展能力，并在压力场景下导致服务无法拉起新的工作进程。",
			})
			suggestions = append(suggestions, models.Suggestion{
				FindingID: id,
				Title:     "提升最大进程数到 655350",
				Details: "建议：\n" +
					"1. 临时调整（当前 shell 会话）：\n" +
					"   ulimit -SHu 655350\n\n" +
					"2. 持久化配置（/etc/security/limits.conf）：\n" +
					"   root soft nproc 655350\n" +
					"   root hard nproc 655350\n" +
					"   *    soft nproc 655350\n" +
					"   *    hard nproc 655350\n" +
					"修改后需要重新登录或重启对应服务进程生效。",
			})
		}
	}

	return findings, suggestions
}

// readSysctl 通过 /proc/sys 读取 sysctl 参数的当前值。
func readSysctl(key string) (string, error) {
	// 例如 net.ipv4.tcp_syncookies -> /proc/sys/net/ipv4/tcp_syncookies
	path := "/proc/sys/" + strings.ReplaceAll(key, ".", "/")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// getRlimit 读取指定资源的 RLIMIT。
func getRlimit(resource int) (*syscall.Rlimit, error) {
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(resource, &rl); err != nil {
		return nil, err
	}
	return &rl, nil
}

// sanitizeID 将 sysctl key 转成适合作为 Finding.ID 的形式。
func sanitizeID(key string) string {
	// net.ipv4.tcp_syncookies -> net_ipv4_tcp_syncookies
	return strings.ReplaceAll(key, ".", "_")
}
