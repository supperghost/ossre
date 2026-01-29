//go:build linux
// +build linux

package maxproc

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/supperghost/ossre/pkg/models"
)

const (
	threadHeadroomFindingID = "maxproc.thread.headroom"
	threadHeadroomUnlimited = int64(999999999)
)

// runMaxprocScenario 在 Linux 上实现“还能创建多少线程”与“首个阻断因素”场景。
// 逻辑等同于 kernel.thread.headroom 的 Linux 版本，通过 /proc、/sys 以及 cgroup v1/v2 估算线程创建余量。
func runMaxprocScenario(ctx context.Context) ([]models.Finding, []models.Suggestion) {
	pid := resolveTargetPID(ctx)

	finding, suggestion := evaluateThreadCreationHeadroom(ctx, pid)

	findings := []models.Finding{finding}
	var suggestions []models.Suggestion
	if suggestion.FindingID != "" || suggestion.Title != "" || suggestion.Details != "" {
		suggestions = append(suggestions, suggestion)
	}

	return findings, suggestions
}

// resolveTargetPID 从 ctx 中解析目标 PID；若未指定或非法，则回退为当前进程 PID。
func resolveTargetPID(ctx context.Context) int {
	v := ctx.Value("ossre.pid")
	switch vv := v.(type) {
	case int:
		if vv > 0 {
			return vv
		}
	case int32:
		if vv > 0 {
			return int(vv)
		}
	case int64:
		if vv > 0 {
			return int(vv)
		}
	case float64:
		if vv > 0 {
			return int(vv)
		}
	case string:
		if vv != "" {
			if n, err := strconv.Atoi(vv); err == nil && n > 0 {
				return n
			}
		}
	}
	return os.Getpid()
}

// evaluateThreadCreationHeadroom 基于 /proc 与 cgroup 信息估算线程创建余量。
func evaluateThreadCreationHeadroom(ctx context.Context, pid int) (models.Finding, models.Suggestion) {
	procDir := fmt.Sprintf("/proc/%d", pid)
	if st, err := os.Stat(procDir); err != nil || !st.IsDir() {
		desc := fmt.Sprintf("目标 PID=%d 对应的 %s 不存在或不可访问，无法评估线程创建余量。", pid, procDir)
		finding := models.Finding{
			ID:          threadHeadroomFindingID,
			Title:       "无法评估线程创建余量：目标进程不存在或 /proc 不可访问",
			Description: desc,
			Severity:    models.SeverityError,
			Impact:      "无法基于该进程的资源限制估算可创建线程数，请确认 PID 是否正确且进程仍在运行。",
		}
		suggestion := models.Suggestion{
			FindingID: threadHeadroomFindingID,
			Title:     "检查 PID 是否正确以及 /proc 是否挂载",
			Details:   fmt.Sprintf("请确认 PID=%d 对应的进程是否仍在运行；在容器场景中，确保 /proc 已正确挂载为宿主的 /proc。", pid),
		}
		return finding, suggestion
	}

	// 1. 当前线程数：统计 /proc/<pid>/task 条目数
	curThreads := countThreadsOfProcess(procDir)

	// 2. /proc/<pid>/limits：Max processes、Max stack size、Max address space
	maxProc, maxProcUnlimited, stackBytes, stackUnlimited, addrBytes, addrUnlimited := parseProcLimits(procDir)

	// 3. /proc/<pid>/status：VmSize（kB）
	vmSizeKB := readVmSizeKB(procDir)

	// 4. cgroup pids：v2 或 v1
	cgInfo := readCgroupPidsInfo(pid)

	// 5. 系统级：threads-max 与系统当前线程数
	kernelThreadsMax := readIntFromFile("/proc/sys/kernel/threads-max")
	sysThreads := countSystemThreads(ctx)

	// 6. 逐项计算还能创建多少线程：A/B/C/D
	// A: nproc 剩余
	var aLeft int64
	if maxProcUnlimited || curThreads <= 0 {
		aLeft = threadHeadroomUnlimited
	} else {
		aLeft = maxProc - curThreads
	}

	// B: cgroup 剩余（PIDS_MAX - PIDS_CUR + CUR_THREADS）
	var bLeft int64
	if cgInfo.PidsMaxUnlimited || cgInfo.Type == "none" || curThreads <= 0 {
		bLeft = threadHeadroomUnlimited
	} else {
		bLeft = cgInfo.PidsMax - cgInfo.PidsCurrent + curThreads
	}

	// C: kernel threads-max 剩余（threads-max - SYS_THREADS）
	var cLeft int64
	if kernelThreadsMax <= 0 || sysThreads <= 0 {
		cLeft = threadHeadroomUnlimited
	} else {
		cLeft = kernelThreadsMax - sysThreads
	}

	// D: (MaxAddressSpace - VmSize) / StackSize
	var dLeft int64
	if addrUnlimited || stackUnlimited || stackBytes <= 0 || vmSizeKB <= 0 {
		dLeft = threadHeadroomUnlimited
	} else {
		vmLimitKB := addrBytes / 1024
		stackKB := stackBytes / 1024
		if stackKB <= 0 {
			dLeft = threadHeadroomUnlimited
		} else {
			vmRemainKB := vmLimitKB - vmSizeKB
			if vmRemainKB <= 0 {
				dLeft = 0
			} else {
				dLeft = vmRemainKB / stackKB
			}
		}
	}

	// 7. 取最小值及对应原因
	minLeft := aLeft
	reason := "nproc"

	if bLeft < minLeft {
		minLeft = bLeft
		reason = "cgroup pids"
	}
	if cLeft < minLeft {
		minLeft = cLeft
		reason = "kernel threads-max"
	}
	if dLeft < minLeft {
		minLeft = dLeft
		reason = "virtual memory / stack"
	}

	severity := models.SeverityInfo
	if minLeft <= 0 {
		severity = models.SeverityError
	}

	desc := fmt.Sprintf("目标进程 PID=%d 当前线程数约为 %d。按 nproc、cgroup pids、kernel.threads-max 以及虚拟内存/栈尺寸四个维度估算，可额外创建线程数约为 %d，首个阻断因素为 %s。", pid, curThreads, minLeft, reason)
	descDetails := fmt.Sprintf("A(nproc) 剩余: %d\nB(cgroup pids) 剩余: %d (类型: %s)\nC(kernel.threads-max) 剩余: %d\nD(虚拟内存/栈) 剩余: %d", aLeft, bLeft, cgInfo.Type, cLeft, dLeft)

	finding := models.Finding{
		ID:          threadHeadroomFindingID,
		Title:       "线程创建余量评估",
		Description: desc + "\n" + descDetails,
		Severity:    severity,
		Impact:      "当线程创建余量为 0 或负数时，目标进程后续创建线程将立即失败，可能表现为 OOM、资源暂时不可用或请求无法被处理。",
	}

	suggestion := buildThreadHeadroomSuggestion(reason)

	return finding, suggestion
}

// countThreadsOfProcess 统计 /proc/<pid>/task 目录下的任务数量。
func countThreadsOfProcess(procDir string) int64 {
	taskDir := filepath.Join(procDir, "task")
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return 0
	}
	return int64(len(entries))
}

// parseProcLimits 解析 /proc/<pid>/limits 中的关键限制项。
func parseProcLimits(procDir string) (maxProc int64, maxProcUnlimited bool, stackBytes int64, stackUnlimited bool, addrBytes int64, addrUnlimited bool) {
	path := filepath.Join(procDir, "limits")
	data, err := os.ReadFile(path)
	if err != nil {
		// 无法读取时按“无上限”处理，以避免错误告警
		maxProcUnlimited, stackUnlimited, addrUnlimited = true, true, true
		return
	}

	lines := strings.Split(string(data), "\n")
	var sawMaxProc, sawStack, sawAddr bool

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Limit") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// Soft/Hard/Unit 固定在行尾三列
		soft := fields[len(fields)-3]

		switch {
		case strings.HasPrefix(line, "Max processes"):
			sawMaxProc = true
			if soft == "unlimited" {
				maxProcUnlimited = true
			} else if v, err := strconv.ParseInt(soft, 10, 64); err == nil {
				maxProc = v
			} else {
				maxProcUnlimited = true
			}
		case strings.HasPrefix(line, "Max stack size"):
			sawStack = true
			if soft == "unlimited" {
				stackUnlimited = true
			} else if v, err := strconv.ParseInt(soft, 10, 64); err == nil {
				stackBytes = v
			} else {
				stackUnlimited = true
			}
		case strings.HasPrefix(line, "Max address space"):
			sawAddr = true
			if soft == "unlimited" {
				addrUnlimited = true
			} else if v, err := strconv.ParseInt(soft, 10, 64); err == nil {
				addrBytes = v
			} else {
				addrUnlimited = true
			}
		}
	}

	// 若未找到对应行，则按无限制处理
	if !sawMaxProc {
		maxProcUnlimited = true
	}
	if !sawStack {
		stackUnlimited = true
	}
	if !sawAddr {
		addrUnlimited = true
	}

	return
}

// readVmSizeKB 从 /proc/<pid>/status 中读取 VmSize（kB）。
func readVmSizeKB(procDir string) int64 {
	path := filepath.Join(procDir, "status")
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "VmSize:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if v, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					return v
				}
			}
			break
		}
	}

	return 0
}

// cgroupPidsInfo 保存 pids cgroup 相关信息。
type cgroupPidsInfo struct {
	Type             string
	PidsMax          int64
	PidsMaxUnlimited bool
	PidsCurrent      int64
}

// readCgroupPidsInfo 读取 cgroup v2 或 v1 的 pids.max 与 pids.current。
func readCgroupPidsInfo(pid int) cgroupPidsInfo {
	info := cgroupPidsInfo{
		Type:             "none",
		PidsMaxUnlimited: true, // 默认视为无限制
	}

	// cgroup v2：/sys/fs/cgroup/pids.max 存在
	v2MaxPath := "/sys/fs/cgroup/pids.max"
	if st, err := os.Stat(v2MaxPath); err == nil && !st.IsDir() {
		info.Type = "v2"
		info.PidsMax, info.PidsMaxUnlimited = readPidsLimitFile(v2MaxPath)
		info.PidsCurrent = readIntFromFile("/sys/fs/cgroup/pids.current")
		return info
	}

	// cgroup v1：根据 /proc/<pid>/cgroup 查找 pids 控制器路径
	cgroupPath := ""
	cgFile := fmt.Sprintf("/proc/%d/cgroup", pid)
	if data, err := os.ReadFile(cgFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 3)
			if len(parts) != 3 {
				continue
			}
			controllers := parts[1]
			if strings.Contains(controllers, "pids") {
				cgroupPath = parts[2]
				break
			}
		}
	}

	if cgroupPath != "" {
		base := "/sys/fs/cgroup/pids"
		maxPath := filepath.Join(base, cgroupPath, "pids.max")
		curPath := filepath.Join(base, cgroupPath, "pids.current")
		if st, err := os.Stat(maxPath); err == nil && !st.IsDir() {
			info.Type = "v1"
			info.PidsMax, info.PidsMaxUnlimited = readPidsLimitFile(maxPath)
			info.PidsCurrent = readIntFromFile(curPath)
			return info
		}
	}

	return info
}

// readPidsLimitFile 解析 pids.max 文件，支持 "max"/"unlimited" 语义。
func readPidsLimitFile(path string) (int64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, true
	}
	s := strings.TrimSpace(string(data))
	if s == "max" || s == "unlimited" {
		return 0, true
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, true
	}
	return v, false
}

// readIntFromFile 从给定文件中读取 int64 数值，失败时返回 0。
func readIntFromFile(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// countSystemThreads 优先使用 "ps -eLf | wc -l"，失败时回退为遍历 /proc/*/task 计数。
func countSystemThreads(ctx context.Context) int64 {
	if n, err := countSystemThreadsWithPs(ctx); err == nil && n > 0 {
		return n
	}
	if n, err := countSystemThreadsByProc(); err == nil && n > 0 {
		return n
	}
	return 0
}

func countSystemThreadsWithPs(ctx context.Context) (int64, error) {
	cmd := exec.CommandContext(ctx, "ps", "-eLf")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	if len(out) == 0 {
		return 0, fmt.Errorf("ps 输出为空")
	}
	var lines int64
	for _, b := range out {
		if b == '\n' {
			lines++
		}
	}
	if lines == 0 {
		return 0, fmt.Errorf("ps 输出没有有效行")
	}
	return lines, nil
}

func countSystemThreadsByProc() (int64, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	var total int64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, err := strconv.Atoi(name); err != nil {
			continue
		}
		taskDir := filepath.Join("/proc", name, "task")
		threads, err := os.ReadDir(taskDir)
		if err != nil {
			continue
		}
		total += int64(len(threads))
	}
	return total, nil
}

// buildThreadHeadroomSuggestion 根据首个阻断因素生成对应的建议。
func buildThreadHeadroomSuggestion(reason string) models.Suggestion {
	switch reason {
	case "nproc":
		return models.Suggestion{
			FindingID: threadHeadroomFindingID,
			Title:     "提升 per-user 进程数限制 (nproc) 以扩展线程创建余量",
			Details: "检测到首个阻断因素为 Max processes (nproc) 软限制。" +
				"\n\n" +
				"1. 临时提升当前会话限制（仅对当前 shell/服务进程生效）：\n" +
				"   ulimit -SHu 655350\n\n" +
				"2. 持久化为系统级配置（/etc/security/limits.conf 示例）：\n" +
				"   * soft nproc 655350\n" +
				"   * hard nproc 655350\n" +
				"   root soft nproc 655350\n" +
				"   root hard nproc 655350\n\n" +
				"修改完成后需重新登录或重启相关服务，使新的 nproc 限制生效。",
		}
	case "cgroup pids":
		return models.Suggestion{
			FindingID: threadHeadroomFindingID,
			Title:     "提升 cgroup pids.max 以扩展线程创建余量",
			Details: "检测到首个阻断因素为 cgroup pids 限制 (pids.max)。" +
				"\n\n" +
				"1. 在 cgroup v2 环境中，可通过以下方式调整：\n" +
				"   echo <新上限> > /sys/fs/cgroup/pids.max\n\n" +
				"2. 在 cgroup v1 环境中，可在对应 pids 层级下调整：\n" +
				"   echo <新上限> > /sys/fs/cgroup/pids/<cgroup>/pids.max\n\n" +
				"3. 若使用 systemd / 容器编排（如 Docker、Kubernetes），建议通过服务单元或 Pod 配置中的 pids 限制字段进行调整，以便配置可持久化与复现。",
		}
	case "kernel threads-max":
		return models.Suggestion{
			FindingID: threadHeadroomFindingID,
			Title:     "调整 kernel.threads-max 提升系统级线程上限",
			Details: "检测到首个阻断因素为内核参数 kernel.threads-max。" +
				"\n\n" +
				"1. 临时调整（重启失效）：\n" +
				"   sysctl -w kernel.threads-max=<新上限>\n\n" +
				"2. 持久化配置（/etc/sysctl.conf 示例）：\n" +
				"   kernel.threads-max = <新上限>\n" +
				"   sysctl -p\n\n" +
				"注意：提升线程总数上限会增加内核内存与调度开销，请结合实际负载与内存容量评估合适的值。",
		}
	case "virtual memory / stack":
		return models.Suggestion{
			FindingID: threadHeadroomFindingID,
			Title:     "通过调整虚拟内存上限或线程栈大小释放线程创建空间",
			Details: "检测到首个阻断因素为 Max address space 与 Max stack size（虚拟内存/栈大小组合）。" +
				"\n\n" +
				"可选操作方向：\n" +
				"1. 降低单线程栈大小（影响 Max stack size）：例如在启动服务前设置 \"ulimit -s\" 为较小值，或在 /etc/security/limits.conf 中调整栈大小限制。\n" +
				"2. 提升进程可用虚拟地址空间上限（Max address space），在 /etc/security/limits.conf 或相关 PAM 配置中放宽该限制。\n" +
				"3. 从应用侧控制并发线程数，避免单进程创建过多线程占用虚拟内存。",
		}
	default:
		return models.Suggestion{}
	}
}
