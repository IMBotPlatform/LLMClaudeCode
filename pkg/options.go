package claudecode

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
