package types

// SudoCommand defines a privileged command request.
// It aggregates execution context (cwd, env) and a timestamp for validity checks.
type SudoCommand struct {
	Slug      string   `json:"slug"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	Cwd       string   `json:"cwd"`
	Env       []string `json:"env"`
	Timestamp int64    `json:"timestamp"`
}

// HistoryEvent is sent by the shell hook to record a command execution.
type HistoryEvent struct {
	Command   string `json:"command"`
	Cwd       string `json:"cwd"`
	Timestamp int64  `json:"timestamp"`
	ExitCode  int    `json:"exit_code"`
	Duration  int64  `json:"duration_ms"`
}
