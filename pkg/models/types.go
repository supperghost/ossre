package models

// Severity 表示诊断发现的严重级别。
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Finding 表示一次诊断中的单条发现。
type Finding struct {
	// 插件内部的发现 ID，便于排错与归档。
	ID string
	// 简要标题。
	Title string
	// 详细描述。
	Description string
	// 严重级别。
	Severity Severity
	// 可选：影响范围或影响描述。
	Impact string
}

// Suggestion 表示针对某个 Finding 给出的修复建议。
type Suggestion struct {
	// 与某个 Finding 关联的 ID，留空表示通用建议。
	FindingID string
	// 建议的简要标题。
	Title string
	// 具体操作建议或说明。
	Details string
}

// Result 表示某个插件一次执行的整体结果。
type Result struct {
	// 产出该结果的插件名称。
	Plugin string
	// 诊断发现列表。
	Findings []Finding
	// 建议列表。
	Suggestions []Suggestion
}
