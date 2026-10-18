// Package doctor runs environment pre-flight checks so the CLI
// (`llamaherd doctor`) and the GUI (first-run panel/banner) can tell the user
// what is misconfigured and how to fix it. Run() never panics and never needs
// the control API or a running daemon.
package doctor

// Status is the outcome of a single check.
type Status string

const (
	StatusOK   Status = "ok"   // good
	StatusWarn Status = "warn" // works, but a recommendation is unmet
	StatusFail Status = "fail" // will not work until fixed
	StatusInfo Status = "info" // purely informational
	StatusSkip Status = "skip" // not applicable on this platform/setup
)

// Check is one environment probe.
type Check struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

// Report is the full diagnostic result.
type Report struct {
	OS          string  `json:"os"`
	Arch        string  `json:"arch"`
	Platform    string  `json:"platform"`
	GeneratedAt string  `json:"generatedAt"`
	Checks      []Check `json:"checks"`
	OK          bool    `json:"ok"`      // true when no check is StatusFail
	Summary     string  `json:"summary"` // e.g. "1 failure, 2 warnings"
}
