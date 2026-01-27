package claudecode

import "time"

// OutputMode 控制输出内容的详细程度。
type OutputMode int

const (
	// OutputModeText 仅输出最终文本 (默认行为)。
	OutputModeText OutputMode = iota
	// OutputModeVerbose 输出文本 + 工具调用摘要。
	OutputModeVerbose
	// OutputModeFull 输出完整的 Agent 执行轨迹。
	OutputModeFull
)

// String 返回 OutputMode 的字符串表示。
func (m OutputMode) String() string {
	switch m {
	case OutputModeText:
		return "text"
	case OutputModeVerbose:
		return "verbose"
	case OutputModeFull:
		return "full"
	default:
		return "unknown"
	}
}

// ToolEventType 工具事件类型。
type ToolEventType int

const (
	// ToolEventUse 工具调用请求。
	ToolEventUse ToolEventType = iota
	// ToolEventResult 工具执行结果。
	ToolEventResult
)

// String 返回 ToolEventType 的字符串表示。
func (t ToolEventType) String() string {
	switch t {
	case ToolEventUse:
		return "tool_use"
	case ToolEventResult:
		return "tool_result"
	default:
		return "unknown"
	}
}

// ToolEvent 工具调用事件。
type ToolEvent struct {
	Type      ToolEventType  // 事件类型
	ToolName  string         // 工具名称, e.g. "read_file", "run_command"
	ToolID    string         // 工具调用 ID
	Input     map[string]any // tool_use 时的输入参数
	Output    string         // tool_result 时的输出内容
	Timestamp time.Time      // 事件时间戳
}

// ToolEventHook 工具事件回调函数类型。
type ToolEventHook func(event ToolEvent)

// Options defines the configuration for Claude Code CLI integration.
type Options struct {
	// CLIPath is the explicit path to the Claude Code CLI binary.
	CLIPath string
	// Model specifies the Claude model name.
	Model string
	// SystemPrompt is the global system prompt passed to the CLI.
	SystemPrompt string
	// Cwd is the working directory for the CLI process.
	Cwd string
	// PermissionMode controls CLI tool permissions (e.g. bypassPermissions).
	PermissionMode string
	// Tools overrides the CLI base tool set.
	Tools []string
	// AllowedTools restricts which tools are allowed to run.
	AllowedTools []string
	// DisallowedTools restricts which tools are explicitly blocked.
	DisallowedTools []string
	// Env provides extra environment variables for the CLI process.
	Env map[string]string
	// ExtraArgs provides additional CLI flags (flag -> value). Empty value means boolean flag.
	ExtraArgs map[string]string
	// MaxBufferSize sets the maximum stdout line size for stream-json parsing.
	MaxBufferSize int
	// OutputMode 控制输出内容的详细程度。
	OutputMode OutputMode
	// ToolEventHook 工具事件回调，当 Agent 调用工具时触发。
	ToolEventHook ToolEventHook

	// SessionID 指定会话 ID（UUID 格式），用于恢复/继续特定会话。
	// 当设置时，Claude CLI 将加载并继续该会话的对话历史。
	SessionID string
	// Resume 是否恢复会话。若 SessionID 非空则自动恢复该会话。
	Resume bool
	// ForkSession 恢复时是否创建新 session ID（与 Resume 配合使用）。
	ForkSession bool
	// NoSessionPersistence 禁用 session 持久化（仅 --print 模式有效）。
	NoSessionPersistence bool
}

// Option mutates Options.
type Option func(*Options)

const (
	defaultPermissionMode = "bypassPermissions"
	defaultMaxBufferSize  = 1024 * 1024
)

func defaultOptions() Options {
	return Options{
		PermissionMode: defaultPermissionMode,
		MaxBufferSize:  defaultMaxBufferSize,
		Env:            map[string]string{},
		ExtraArgs:      map[string]string{},
	}
}

// WithCLIPath sets the path to the Claude Code CLI binary.
func WithCLIPath(path string) Option {
	return func(o *Options) {
		o.CLIPath = path
	}
}

// WithModel sets the Claude model name.
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// WithSystemPrompt sets a global system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.SystemPrompt = prompt
	}
}

// WithCwd sets the working directory for the CLI process.
func WithCwd(cwd string) Option {
	return func(o *Options) {
		o.Cwd = cwd
	}
}

// WithPermissionMode sets the CLI permission mode.
func WithPermissionMode(mode string) Option {
	return func(o *Options) {
		o.PermissionMode = mode
	}
}

// WithTools sets the CLI base tool set.
func WithTools(tools ...string) Option {
	return func(o *Options) {
		o.Tools = append([]string{}, tools...)
	}
}

// WithAllowedTools sets the allowed tools list.
func WithAllowedTools(tools ...string) Option {
	return func(o *Options) {
		o.AllowedTools = append([]string{}, tools...)
	}
}

// WithDisallowedTools sets the disallowed tools list.
func WithDisallowedTools(tools ...string) Option {
	return func(o *Options) {
		o.DisallowedTools = append([]string{}, tools...)
	}
}

// WithEnv sets extra environment variables for the CLI process.
func WithEnv(env map[string]string) Option {
	return func(o *Options) {
		o.Env = make(map[string]string, len(env))
		for k, v := range env {
			o.Env[k] = v
		}
	}
}

// WithExtraArgs sets additional CLI flags.
func WithExtraArgs(args map[string]string) Option {
	return func(o *Options) {
		o.ExtraArgs = make(map[string]string, len(args))
		for k, v := range args {
			o.ExtraArgs[k] = v
		}
	}
}

// WithMaxBufferSize sets the maximum line size for stdout parsing.
func WithMaxBufferSize(size int) Option {
	return func(o *Options) {
		if size > 0 {
			o.MaxBufferSize = size
		}
	}
}

// WithOutputMode sets the output detail level.
// 参数：mode 为 OutputMode 枚举值。
func WithOutputMode(mode OutputMode) Option {
	return func(o *Options) {
		o.OutputMode = mode
	}
}

// WithToolEventHook sets the tool event callback hook.
// 参数：hook 为工具事件回调函数，当 Agent 调用工具时触发。
func WithToolEventHook(hook ToolEventHook) Option {
	return func(o *Options) {
		o.ToolEventHook = hook
	}
}

// WithSessionID sets the session ID for conversation continuity.
// 参数：sessionID 为 UUID 格式的会话 ID。
// 设置后 Claude CLI 将加载并继续该会话的对话历史。
func WithSessionID(sessionID string) Option {
	return func(o *Options) {
		o.SessionID = sessionID
	}
}

// WithResume enables session resumption.
// 参数：resume 为是否启用会话恢复。
// 若 SessionID 非空，则恢复该会话；否则恢复最近的会话。
func WithResume(resume bool) Option {
	return func(o *Options) {
		o.Resume = resume
	}
}

// WithForkSession creates a new session ID when resuming.
// 参数：fork 为是否在恢复时创建新分支。
// 需与 Resume 配合使用。
func WithForkSession(fork bool) Option {
	return func(o *Options) {
		o.ForkSession = fork
	}
}

// WithNoSessionPersistence disables session persistence.
// 参数：disabled 为是否禁用会话持久化。
// 仅在 --print 模式下有效。
func WithNoSessionPersistence(disabled bool) Option {
	return func(o *Options) {
		o.NoSessionPersistence = disabled
	}
}
